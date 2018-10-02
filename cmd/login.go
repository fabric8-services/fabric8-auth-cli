package cmd

import (
	"github.com/spf13/cobra"
)

// NewLoginCommand a command to login on `fabtic8-auth` service
func newLoginCommand() *cobra.Command {
	c := &cobra.Command{
		Short: "login",
		Use:   "login",
		// Args:  cobra.MinimumNArgs(1),
		Run: login,
	}

	return c
}

func login(cmd *cobra.Command, args []string) {

}
