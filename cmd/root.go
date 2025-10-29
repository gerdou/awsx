package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vahid-haghighat/awsx/version"
)

var versionFlag bool

var rootCmd = &cobra.Command{
	Use:               "awsx",
	Short:             "Retrieve short-living credentials via AWS SSO",
	Long:              `Retrieve short-living credentials via AWS SSO`,
	DisableAutoGenTag: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if versionFlag {
			fmt.Println(version.Version)
			return nil
		}
		return refreshCmd.RunE(cmd, args)
	},
	Args: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolVarP(&versionFlag, "version", "v", false, "Prints awsx's version")
}
