package main

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/fabric8-services/fabric8-auth-cli/cmd"
	"github.com/fabric8-services/fabric8-common/log"
)

func main() {

	logrus.SetLevel(logrus.WarnLevel)
	log.InitializeLogger(false, logrus.WarnLevel.String())
	rootCmd := cmd.NewRootCommand()
	// rootCmd.PersistentFlags().MarkShorthandDeprecated("help", "please use --help")
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
