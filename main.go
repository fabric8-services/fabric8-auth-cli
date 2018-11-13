package main

import (
	"fmt"
	"os"

	"github.com/fabric8-services/fabric8-auth-cli/cmd"
)

func main() {

	rootCmd := cmd.NewRootCommand()
	// rootCmd.PersistentFlags().MarkShorthandDeprecated("help", "please use --help")
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
