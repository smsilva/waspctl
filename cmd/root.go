package cmd

import (
	"os"
	"path/filepath"

	"github.com/smsilva/waspctl/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "waspctl",
	Short: "Manage the WASP multi-tenant Kubernetes platform",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initConfig(cfgFile)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ~/.wasp/config.yaml)")
	rootCmd.PersistentFlags().StringP("output", "o", "table", "Output format: table|json|yaml")
	rootCmd.PersistentFlags().BoolP("yes", "y", false, "Skip confirmation prompts")
}

func initConfig(path string) error {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		path = filepath.Join(home, ".wasp", "config.yaml")
	}
	config.SetConfigPath(path)
	viper.SetConfigFile(path)
	// Ignore "file not found" — created on first --set
	_ = viper.ReadInConfig()
	return nil
}
