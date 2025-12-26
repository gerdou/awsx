package cmd

import (
	"github.com/spf13/cobra"
	"github.com/gerdou/awsx/cmd/internal"
	"github.com/gerdou/awsx/utilities"
	"log"
	"sort"
)

var configCmd = &cobra.Command{
	Use:               "config",
	Short:             "Configures awsx",
	Long:              `Configures one or more AWS SSO configurations`,
	Example:           "awsx config my-sso-config",
	DisableAutoGenTag: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		configNames := []string{"default"}
		if len(args) > 0 {
			configNames = args
		}
		return configArgs(configNames)
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}

func configArgs(configNames []string) error {
	configs, _ := internal.ReadInternalConfig()
	if configs == nil || len(configs) == 0 {
		configs = make(map[string]*internal.Config)
	}
	prompter := internal.Prompter{}

	for _, configName := range configNames {
		if configName == "" {
			continue
		}

		config, ok := configs[configName]
		if !ok {
			config = &internal.Config{
				Id:        "",
				SsoRegion: "",
				Profiles:  make(map[string]*internal.Profile),
			}
		}
		config.Complete = false

		var err error
		config.Id, err = prompter.Prompt("Start URL Id", config.Id)
		if err != nil {
			log.Printf("Failed to prompt for start URL Id for %s\n", configName)
			continue
		}

		config.SsoRegion, err = prompter.Prompt("SSO Region", config.SsoRegion)
		if err != nil {
			log.Printf("Failed to prompt for sso region for %s\n", configName)
			continue
		}

		if config.SsoRegion == "" {
			log.Println("SSO Region cannot be empty")
			continue
		}

		profileNames := utilities.Keys(config.Profiles)
		sort.SliceStable(profileNames, func(i, j int) bool {
			return profileNames[i] < profileNames[j]
		})

		var profileName string
		var region string
		profilesConfigured := 0
		for {
			defaultProfileName := "default"
			if len(profileNames) > 0 {
				defaultProfileName = profileNames[0]
			}
			profileName, err = prompter.Prompt("Profile name to configure", defaultProfileName)
			if err != nil {
				log.Printf("Failed to prompt for %s config argument: %s\n", configName, err)
				break
			}
			if profileName == "" {
				log.Println("Profile name cannot be empty")
				break
			}

			if profileName == defaultProfileName && len(profileNames) > 1 {
				profileNames = profileNames[1:]
			}

			defaultRegion, found := config.Profiles[profileName]
			if !found {
				defaultRegion = &internal.Profile{
					Region: "",
				}
			}

			region, err = prompter.Prompt("Default Profile Region", defaultRegion.Region)
			if err != nil {
				log.Printf("Failed to prompt for region for %s: %s\n", configName, err)
				break
			}
			if region == "" {
				log.Println("Region name cannot be empty")
				break
			}

			config.Profiles[profileName] = &internal.Profile{
				Region:         region,
				DefaultAccount: nil,
			}

			index, _, _ := prompter.Select("Do you wish to configure a default account for this profile?", []string{"Yes", "No"}, nil)
			if index == 0 {
				config.Profiles[profileName].DefaultAccount = configDefaultAccountForProfile(profileName, prompter)
			}

			profilesConfigured++

			index, _, _ = prompter.Select("Do you wish to add another profile to this config?", []string{"Yes", "No"}, nil)
			if index != 0 {
				break
			}
		}

		if profilesConfigured == 0 {
			continue
		}

		config.Complete = true
		configs[configName] = config
	}

	return internal.WriteInternalConfig(configs)
}

func configDefaultAccountForProfile(profile string, prompter internal.Prompter) *internal.UsageInformation {
	defaultAccount := &internal.UsageInformation{}

	var err error
	defaultAccount.AccountId, err = prompter.Prompt("Default Account Id for this profile", "")
	if err != nil {
		log.Printf("Failed to prompt for default account id for profile: %s\n", err)
		return nil
	}

	if defaultAccount.AccountId == "" {
		log.Println("Default Account Id cannot be empty")
		return nil
	}

	defaultAccount.AccountName, err = prompter.Prompt("Default Account Name for this profile", "")
	if err != nil {
		log.Printf("Failed to prompt for default account name for profile: %s\n", err)
		return nil
	}

	if defaultAccount.AccountName == "" {
		log.Println("Default Account Name cannot be empty")
		return nil
	}

	defaultAccount.Role, err = prompter.Prompt("Default Role for this profile (optional)", "")
	if err != nil {
		log.Printf("Failed to prompt for default role for profile: %s\n", err)
	}

	return defaultAccount
}
