/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var secretCmd = &cobra.Command{
	Use:   "secret",
	Short: "Secret Management",
	Long: `
Secret Management is a subcommand to manage secrets in the WASP Project.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("secret called")
	},
}

func init() {
	rootCmd.AddCommand(secretCmd)
}
