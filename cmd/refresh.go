package cmd

import (
	"github.com/spf13/cobra"
	"github.com/gerdou/awsx/cmd/internal"
	"github.com/gerdou/awsx/utilities"
)

var refreshCmd = &cobra.Command{
	Use:               "refresh",
	Short:             "Refreshes your previously used credentials.",
	Long:              `Refreshes your previously used credentials.`,
	DisableAutoGenTag: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		configName, configs, profileNames, err := processInputArgsForSelectAndRefresh(cmd, args)
		if err != nil {
			return err
		}

		oidcApi, ssoApi := internal.InitClients(configs[configName])

		if len(profileNames) >= 1 {
			if profileNames[0] == "all" {
				profileNames = utilities.Keys(configs[configName].Profiles)
			}

			return actionWithSpecifiedProfiles(configs[configName], profileNames, oidcApi, ssoApi, internal.Refresh)
		}

		return actionWithUnspecifiedProfiles(configs[configName], oidcApi, ssoApi, internal.Refresh)
	},
}

func init() {
	rootCmd.AddCommand(refreshCmd)
}
