package cmd

import (
	"github.com/gerdou/awsx/cmd/internal"
	"github.com/gerdou/awsx/utilities"

	"github.com/spf13/cobra"
)

var configImportPath string

// importCmd represents the import command
var importCmd = &cobra.Command{
	Use:               "import",
	Short:             "Imports awsx configs",
	Long:              `Imports awsx configs`,
	DisableAutoGenTag: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		configImportPath, err = utilities.AbsolutePath(configImportPath)
		if err != nil {
			return err
		}
		return internal.ImportInternalConfig(configImportPath)
	},
}

func init() {
	configCmd.AddCommand(importCmd)
}
