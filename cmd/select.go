package cmd

import (
	"github.com/spf13/cobra"
	"github.com/gerdou/awsx/cmd/internal"
	"github.com/gerdou/awsx/utilities"
)

var selectCmd = &cobra.Command{
	Use:               "select",
	Short:             "Lets you select a profile from available profiles on AWS SSO",
	Long:              `Lets you select a profile from available profiles on AWS SSO`,
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
			return actionWithSpecifiedProfiles(configs[configName], profileNames, oidcApi, ssoApi, internal.Select)
		}

		return actionWithUnspecifiedProfiles(configs[configName], oidcApi, ssoApi, internal.Select)
	},
}

func init() {
	rootCmd.AddCommand(selectCmd)
}
