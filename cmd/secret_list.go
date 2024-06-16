package cmd

import (
	"github.com/spf13/cobra"
)

var secretListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all secrets",
	Long:  `List all secrets managed by waspctl.`,
	Run: func(cmd *cobra.Command, args []string) {
		secrets := []map[string]string{
			{"name": "secret1", "value": "value1"},
			{"name": "secret2", "value": "value2"},
		}
		output(secrets)
	},
}

func init() {
	secretCmd.AddCommand(secretListCmd)
}
