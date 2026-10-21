package internal

import (
	"context"
	"errors"
	"log"
	"os"
	"path"
	"slices"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/sso"
	ssoTypes "github.com/aws/aws-sdk-go-v2/service/sso/types"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	"github.com/aws/smithy-go"
	"github.com/gerdou/awsx/utilities"
	"gopkg.in/ini.v1"
	"gopkg.in/yaml.v3"
)

// SyncProfiles makes the profiles of the config match the accounts the user can access on AWS SSO.
// Every profile of an account that is no longer accessible is removed, and a profile is added for
// every newly accessible account, unless the account is in the config's ignored accounts.
// Profiles are matched to accounts by account id only, so an account can have several profiles.
func SyncProfiles(config *Config, oidcClient *ssooidc.Client, ssoClient *sso.Client) error {
	clientInformation, err := ProcessClientInformation(config.Name, config.GetStartUrl(), oidcClient)
	if err != nil {
		return err
	}

	accounts, err := listAllAccounts(clientInformation.AccessToken, ssoClient)
	if hasErrorCode(err, "UnauthorizedException") {
		log.Println("Access token invalid or expired. Re-authenticating...")
		clientInformation, err = HandleOutdatedAccessToken(config.Name, config.GetStartUrl(), clientInformation, oidcClient)
		if err != nil {
			return err
		}
		accounts, err = listAllAccounts(clientInformation.AccessToken, ssoClient)
	}
	if err != nil {
		return err
	}

	// An empty list is far more likely to be a glitch than having lost access to every account
	if len(accounts) == 0 {
		return errors.New("AWS SSO returned no accounts")
	}

	accessibleAccounts := make(map[string]string, len(accounts))
	for _, account := range accounts {
		accessibleAccounts[*account.AccountId] = *account.AccountName
	}

	changed := false
	var removedProfiles []string
	configuredAccounts := make(map[string]bool)

	profileNames := utilities.Keys(config.Profiles)
	slices.Sort(profileNames)
	for _, profileName := range profileNames {
		defaultAccount := config.Profiles[profileName].DefaultAccount
		if defaultAccount == nil {
			continue
		}

		accountName, accessible := accessibleAccounts[defaultAccount.AccountId]
		if !accessible {
			log.Printf("Removed profile %s from %s config: you no longer have access to account %s [%s]", profileName, config.Name, defaultAccount.AccountName, defaultAccount.AccountId)
			delete(config.Profiles, profileName)
			removedProfiles = append(removedProfiles, profileName)
			changed = true
			continue
		}

		if defaultAccount.AccountName != accountName {
			log.Printf("Account %s of profile %s in %s config was renamed from %s to %s", defaultAccount.AccountId, profileName, config.Name, defaultAccount.AccountName, accountName)
			defaultAccount.AccountName = accountName
			changed = true
		}
		configuredAccounts[defaultAccount.AccountId] = true
	}

	for _, account := range sortAccounts(accounts) {
		if configuredAccounts[*account.AccountId] || slices.Contains(config.IgnoredAccounts, *account.AccountId) {
			continue
		}

		profileName := strings.ToLower(strings.ReplaceAll(*account.AccountName, " ", "-"))
		if _, exists := config.Profiles[profileName]; exists {
			profileName = profileName + "-" + *account.AccountId
		}
		if _, exists := config.Profiles[profileName]; exists {
			log.Printf("Could not add account %s [%s] to %s config: profile %s already exists", *account.AccountName, *account.AccountId, config.Name, profileName)
			continue
		}

		config.Profiles[profileName] = &Profile{
			Region: config.SsoRegion,
			DefaultAccount: &UsageInformation{
				AccountId:   *account.AccountId,
				AccountName: *account.AccountName,
			},
			Name: profileName,
		}
		log.Printf("Added profile %s to %s config: you have access to new account %s [%s]", profileName, config.Name, *account.AccountName, *account.AccountId)
		changed = true
	}

	if !changed {
		return nil
	}

	configs, err := ReadInternalConfig()
	if err != nil {
		return err
	}
	configs[config.Name] = config
	err = WriteInternalConfig(configs)
	if err != nil {
		return err
	}

	if len(removedProfiles) == 0 {
		return nil
	}

	err = RemoveUsageInformationForProfiles(config.Name, removedProfiles)
	if err != nil {
		return err
	}

	// Credentials sections are shared by all configs, so keep the ones another config still uses
	var unusedProfiles []string
	for _, profileName := range removedProfiles {
		used := false
		for _, c := range configs {
			if _, exists := c.Profiles[profileName]; exists {
				used = true
				break
			}
		}
		if !used {
			unusedProfiles = append(unusedProfiles, profileName)
		}
	}

	return RemoveAwsCredentialsProfiles(unusedProfiles)
}

func listAllAccounts(accessToken string, ssoClient *sso.Client) ([]ssoTypes.AccountInfo, error) {
	var maxSize int32 = 100 // max allowed by the SSO ListAccounts API
	lai := &sso.ListAccountsInput{AccessToken: &accessToken, MaxResults: &maxSize}

	var accountList []ssoTypes.AccountInfo
	paginator := sso.NewListAccountsPaginator(ssoClient, lai)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.Background())
		if err != nil {
			return nil, err
		}
		accountList = append(accountList, page.AccountList...)
	}

	return accountList, nil
}

// roleRevoked reports whether the role in the input is no longer one of the user's roles in the account
func roleRevoked(rci *sso.GetRoleCredentialsInput, ssoClient *sso.Client) (bool, error) {
	lari := &sso.ListAccountRolesInput{AccountId: rci.AccountId, AccessToken: rci.AccessToken}

	paginator := sso.NewListAccountRolesPaginator(ssoClient, lari)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.Background())
		if err != nil {
			return false, err
		}
		for _, role := range page.RoleList {
			if *role.RoleName == *rci.RoleName {
				return false, nil
			}
		}
	}

	return true, nil
}

func hasErrorCode(err error, code string) bool {
	var e smithy.APIError
	return errors.As(err, &e) && e.ErrorCode() == code
}

func RemoveUsageInformationForProfiles(configName string, profileNames []string) error {
	usageInformationFile, err := ReadUsageInformationFile()
	if err != nil {
		return nil
	}

	usageInformation, exists := usageInformationFile.LastUsageInformation[configName]
	if !exists {
		return nil
	}

	for _, profileName := range profileNames {
		delete(usageInformation, profileName)
	}

	content, err := yaml.Marshal(usageInformationFile)
	if err != nil {
		return err
	}

	return os.WriteFile(defaultLastUsageFileName, content, 0700)
}

func RemoveAwsCredentialsProfiles(profileNames []string) error {
	if len(profileNames) == 0 {
		return nil
	}

	credentialsFileName := path.Join(defaultAwsCredentialsPath, defaultAwsCredentialsFileName)
	awsCredentialsFile, err := ini.Load(credentialsFileName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	for _, profileName := range profileNames {
		awsCredentialsFile.DeleteSection(profileName)
	}

	return awsCredentialsFile.SaveTo(credentialsFileName)
}
