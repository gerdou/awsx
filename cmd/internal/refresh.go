package internal

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	"github.com/aws/smithy-go"
	"log"
	"strings"
	"time"
)

func Refresh(config *Config, profile *Profile, oidcClient *ssooidc.Client, ssoClient *sso.Client) error {
	log.Printf("Refreshing credentials for profile %s in config %s", profile.Name, config.Name)
	clientInformation, err := ProcessClientInformation(config.Name, config.GetStartUrl(), oidcClient)
	if err != nil {
		return err
	}

	log.Printf("Using Start URL %s", clientInformation.StartUrl)

	var accountId *string
	var roleName *string

	luis, err := GetUsageInformationForConfig(config.Name)

	var lui *UsageInformation
	for _, info := range luis {
		if info.Profile == profile.Name {
			lui = &info
		}
	}

	if lui == nil {
		log.Printf("Nothing to refresh yet for profile %s in config %s", profile.Name, config.Name)
		return Select(config, profile, oidcClient, ssoClient)
	}

	log.Printf("Attempting to refresh credentials for account %s with role %s", lui.AccountName, lui.Role)
	if err != nil {
		if strings.Contains(err.Error(), "no such file") {
			log.Println("Nothing to refresh yet.")
			return Select(config, profile, oidcClient, ssoClient)
		}
	} else {
		accountId = &lui.AccountId
		roleName = &lui.Role
	}

	rci := &sso.GetRoleCredentialsInput{AccountId: accountId, RoleName: roleName, AccessToken: &clientInformation.AccessToken}
	roleCredentials, err := ssoClient.GetRoleCredentials(context.Background(), rci)
	if err != nil {
		return unwrapSmithyError(err)
	}

	err = WriteAwsConfigFile(profile.Name, config, roleCredentials.RoleCredentials)
	if err != nil {
		return err
	}

	err = SaveUsageInformationForConfig(config.Name, lui)

	if accountId == nil || roleName == nil {
		return errors.New("no account or role found")
	}

	log.Printf("Retrieved credentials for account %s [%s] successfully", lui.AccountName, *accountId)
	log.Printf("Assumed role: %s", *roleName)
	log.Printf("Credentials expire at: %s\n", time.Unix(roleCredentials.RoleCredentials.Expiration/1000, 0))
	fmt.Println()
	return nil
}

func unwrapSmithyError(err error) error {
	var e *smithy.GenericAPIError
	if !errors.As(err, &e) {
		return err
	}

	switch {
	case e.ErrorCode() == "ForbiddenException":
		return errors.New("you do not have permission to assume the role. Please check your AWS SSO configuration")
	default:
		return err
	}
}
