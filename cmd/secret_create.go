/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var secretCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "A brief description of your command",
	Long: `
.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("secret create called")
	},
}

func init() {
	secretCmd.AddCommand(secretCreateCmd)
}
