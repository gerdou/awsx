package cmd

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	"github.com/spf13/cobra"
	"github.com/vahid-haghighat/awsx/cmd/internal"
	"github.com/vahid-haghighat/awsx/utilities"
	"log"
	"slices"
)

func processInputArgsForSelectAndRefresh(cmd *cobra.Command, args []string) (configName string, configs map[string]*internal.Config, profileNames []string, err error) {
	configName = "default"

	if len(args) >= 1 {
		configName = args[0]
	}

	if len(args) >= 2 {
		profileNames = args[1:]
	}

	configs, err = internal.ReadInternalConfig()
	if err != nil {
		log.Printf("Config file does not exist. Creating it...\n")
		if err = configCmd.RunE(cmd, []string{configName}); err != nil {
			return "", nil, nil, err
		}

		configs, err = internal.ReadInternalConfig()
		if err != nil {
			return "", nil, nil, err
		}
	}

	if _, exists := configs[configName]; !exists {
		log.Printf("Config \"%s\" does not exist. Creating it...\n", configName)
		if err = configCmd.RunE(cmd, []string{configName}); err != nil {
			return "", nil, nil, err
		}

		configs, err = internal.ReadInternalConfig()
		if err != nil {
			return "", nil, nil, err
		}
	}

	return configName, configs, profileNames, nil
}

func actionWithUnspecifiedProfiles(config *internal.Config, oidcApi *ssooidc.Client, ssoApi *sso.Client, action func(*internal.Config, *internal.Profile, *ssooidc.Client, *sso.Client) error) error {
	var selectedProfiles []*internal.Profile
	if len(config.Profiles) > 1 {
		prompt := internal.Prompter{}
		profiles := utilities.Keys(config.Profiles)
		slices.Sort(profiles)
		indexes, err := prompt.MultiSelect("Select the profile", profiles, nil)
		if err != nil {
			return err
		}

		for _, index := range indexes {
			selectedProfiles = append(selectedProfiles, config.Profiles[profiles[index]])
		}
	} else {
		for _, p := range config.Profiles {
			selectedProfiles = []*internal.Profile{p}
		}
	}

	if len(selectedProfiles) == 0 {
		return errors.New("no profile selected")
	}

	var errs []error
	for _, profile := range selectedProfiles {
		if profile == nil {
			return errors.New(fmt.Sprintf("profile is nil for config \"%s\"", config.Name))
		}

		if profile.Name == "" {
			return errors.New(fmt.Sprintf("profile name is empty for config \"%s\"", config.Name))
		}

		if profile.Region == "" {
			return errors.New("no region is set for this profile")
		}

		err := action(config, profile, oidcApi, ssoApi)
		if err != nil {
			errs = append(errs, errors.New(fmt.Sprintf("%s: %v", profile.Name, err)))
		}
	}

	return errors.Join(errs...)
}

func actionWithSpecifiedProfiles(config *internal.Config, profileNames []string, oidcApi *ssooidc.Client, ssoApi *sso.Client, action func(*internal.Config, *internal.Profile, *ssooidc.Client, *sso.Client) error) error {
	var validProfileNames []string
	var errs []error
	for _, profile := range config.Profiles {
		if slices.Contains(profileNames, profile.Name) {
			validProfileNames = append(validProfileNames, profile.Name)
			err := action(config, profile, oidcApi, ssoApi)
			if err != nil {
				errs = append(errs, errors.New(fmt.Sprintf("%s: %v", profile.Name, err)))
			}
		}
	}

	return errors.Join(errs...)
}
