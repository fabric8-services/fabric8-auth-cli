package cmd

import (
	"github.com/spf13/cobra"
)

var verbose = false

// NewRootCommand initializes the root command
func NewRootCommand() *cobra.Command {

	rootCmd := &cobra.Command{
		Use:   "fabric8-auth",
		Short: "fabric8-auth is a CLI tool to perform operations on behalf of the fabric8-auth service",
		// Run: func(cmd *cobra.Command, args []string) {
		// 	// Do Stuff Here
		// },
	}
	helpCommand := newHelpCommand()
	rootCmd.SetHelpCommand(helpCommand)
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose")
	rootCmd.AddCommand(newLoginCommand())
	rootCmd.AddCommand(newDescribeCommand())
	rootCmd.AddCommand(newAboutCommand())
	return rootCmd
}
