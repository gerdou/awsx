package internal

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	"log"
	"time"
)

func Select(config *Config, profile *Profile, oidcClient *ssooidc.Client, ssoClient *sso.Client) error {
	log.Printf("Getting credentials for profile %s in %s config", profile.Name, config.Name)
	clientInformation, err := ProcessClientInformation(config.Name, config.GetStartUrl(), oidcClient)
	if err != nil {
		return err
	}

	log.Printf("Using Start URL %s", clientInformation.StartUrl)
	promptSelector := Prompter{}

	var accountId, accountName, roleName *string
	if profile.DefaultAccount == nil {
		accountInfo := RetrieveAccountInfo(clientInformation, ssoClient, promptSelector)
		accountName = accountInfo.AccountName
		roleInfo := RetrieveRoleInfo(accountInfo.AccountId, clientInformation, ssoClient, promptSelector)

		accountId = accountInfo.AccountId
		roleName = roleInfo.RoleName
	} else {
		accountId = aws.String(profile.DefaultAccount.AccountId)
		accountName = aws.String(profile.DefaultAccount.AccountName)

		if profile.DefaultAccount.Role == "" {
			roleInfo := RetrieveRoleInfo(accountId, clientInformation, ssoClient, promptSelector)
			roleName = roleInfo.RoleName
		} else {
			roleName = aws.String(profile.DefaultAccount.Role)
		}
	}

	_ = SaveUsageInformationForConfig(config.Name, &UsageInformation{
		AccountId:   *accountId,
		AccountName: *accountName,
		Role:        *roleName,
		Profile:     profile.Name,
	})

	rci := &sso.GetRoleCredentialsInput{AccountId: accountId, RoleName: roleName, AccessToken: &clientInformation.AccessToken}
	roleCredentials, err := ssoClient.GetRoleCredentials(context.Background(), rci)
	if err != nil {
		return err
	}

	err = WriteAwsConfigFile(profile.Name, config, roleCredentials.RoleCredentials)
	if err != nil {
		return err
	}

	log.Printf("Retrieved credentials for account %s [%s] successfully", *accountName, *accountId)
	log.Printf("Assumed role: %s", *roleName)
	log.Printf("Credentials expire at: %s\n", time.Unix(roleCredentials.RoleCredentials.Expiration/1000, 0))
	fmt.Println()
	return nil
}
