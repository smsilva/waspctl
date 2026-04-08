package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/smsilva/waspctl/internal/config"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Read and write config file values",
	RunE:  runConfig,
}

var (
	setKey  string
	getKey  string
	listAll bool
)

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.Flags().StringVar(&setKey, "set", "", "Set a config value: --set <key> <value>")
	configCmd.Flags().StringVar(&getKey, "get", "", "Get a config value: --get <key>")
	configCmd.Flags().BoolVar(&listAll, "list", false, "List all config values")
}

func runConfig(cmd *cobra.Command, args []string) error {
	output, _ := cmd.Flags().GetString("output")

	switch {
	case listAll:
		return printList(output)
	case getKey != "":
		val, err := config.Get(getKey)
		if err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout, val)
	case setKey != "":
		if len(args) != 1 {
			return fmt.Errorf("--set requires exactly one argument: the value")
		}
		return config.Set(setKey, args[0])
	default:
		return cmd.Help()
	}
	return nil
}

func printList(output string) error {
	pairs := config.List()

	switch output {
	case "json":
		m := make(map[string]string, len(pairs))
		for _, p := range pairs {
			m[p.Key] = p.Value
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(m)
	case "yaml":
		m := make(map[string]string, len(pairs))
		for _, p := range pairs {
			m[p.Key] = p.Value
		}
		return yaml.NewEncoder(os.Stdout).Encode(m)
	default: // table
		for _, p := range pairs {
			fmt.Fprintf(os.Stdout, "%s=%s\n", p.Key, p.Value)
		}
	}
	return nil
}
