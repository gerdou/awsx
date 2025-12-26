package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/gerdou/awsx/cmd/internal"
)

var removeProfiles []string

var configRemoveCmd = &cobra.Command{
	Use:               "remove",
	Short:             "Removes awsx's Configuration",
	Long:              `Removes awsx's Configuration`,
	DisableAutoGenTag: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(removeProfiles) > 0 {
			configName := "default"
			if len(args) > 0 {
				configName = args[0]
			}
			if len(args) > 1 {
				return fmt.Errorf("when using --profile, specify at most one config name")
			}
			return internal.RemoveProfilesFromConfig(configName, removeProfiles)
		}

		configNames := []string{"default"}
		if len(args) > 0 {
			configNames = args
		}
		return internal.RemoveInternalConfig(configNames)
	},
}

func init() {
	configRemoveCmd.Flags().StringSliceVarP(&removeProfiles, "profile", "p", []string{}, "Profile(s) to remove from the named config")
	configCmd.AddCommand(configRemoveCmd)
}
