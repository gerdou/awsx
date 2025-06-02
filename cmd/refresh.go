package cmd

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/vahid-haghighat/awsx/cmd/internal"
	"github.com/vahid-haghighat/awsx/utilities"
	"sort"
)

var refreshCmd = &cobra.Command{
	Use:               "refresh",
	Short:             "Refreshes your previously used credentials.",
	Long:              `Refreshes your previously used credentials.`,
	DisableAutoGenTag: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var configNames []string

		configs, err := internal.ReadInternalConfig()
		if err != nil {
			if err = configCmd.RunE(cmd, args); err != nil {
				return err
			}

			configs, err = internal.ReadInternalConfig()
			if err != nil {
				return err
			}
		}

		var errs []error
		prompter := internal.Prompter{}

		if len(args) == 0 {
			profileNames := utilities.Keys(configs)
			profileIndexes, err := prompter.MultiSelect("Select the configs to refresh", profileNames, nil)
			if err != nil {
				return err
			}

			args = make([]string, 0, len(profileIndexes))
			for _, index := range profileIndexes {
				args = append(args, profileNames[index])
			}
		}

		configNames = args

	Configs:
		for _, configName := range configNames {
			config, ok := configs[configName]
			if !ok {
				if err = configCmd.RunE(cmd, []string{configName}); err != nil {
					errs = append(errs, err)
					continue Configs
				}

				config = configs[configName]
			}

			var selectedProfiles []*internal.Profile
			if len(configs[configName].Profiles) > 1 {
				profiles := utilities.Keys(configs[configName].Profiles)
				sort.Strings(profiles)

				indexes, err := prompter.MultiSelect(fmt.Sprintf("Select the profiles for config \"%s\"", configName), profiles, nil)
				if err != nil {
					errs = append(errs, err)
					continue Configs
				}

				selectedProfiles = make([]*internal.Profile, 0, len(indexes))
				for _, index := range indexes {
					selectedProfiles = append(selectedProfiles, configs[configName].Profiles[profiles[index]])
				}
			} else {
				for _, p := range configs[configName].Profiles {
					selectedProfiles = []*internal.Profile{p}
				}
			}

			if len(selectedProfiles) == 0 {
				errs = append(errs, errors.New(fmt.Sprintf("no profile selected for \"%s\"", configName)))
				continue Configs
			}

			for _, profile := range selectedProfiles {
				if profile == nil {
					errs = append(errs, errors.New(fmt.Sprintf("profile is nil for config \"%s\"", configName)))
					continue
				}
				if profile.Name == "" {
					errs = append(errs, errors.New(fmt.Sprintf("profile name is empty for config \"%s\"", configName)))
					continue
				}
				if profile.Region == "" {
					errs = append(errs, errors.New(fmt.Sprintf("no region is set for profile \"%s\" in config \"%s\"", profile.Name, configName)))
					continue Configs
				}

				oidcApi, ssoApi := internal.InitClients(configs[configName])
				err = internal.RefreshCredentials(configName, profile, oidcApi, ssoApi, config, prompter)
				if err != nil {
					errs = append(errs, err)
				}
			}
		}

		if len(errs) > 0 {
			message := ""
			for _, err := range errs {
				message += fmt.Sprintf("%s\n", err.Error())
			}

			return errors.New(message)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(refreshCmd)
}
