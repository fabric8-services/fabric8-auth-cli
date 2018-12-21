package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fabric8-services/fabric8-common/auth"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// newDescribeCommand a command to describe a given token
func newDescribeCommand() *cobra.Command {
	describeCmd := &cobra.Command{
		Short: "describes the content of a given OSIO token",
		Use:   "describe",
		Run:   describe,
		Args:  cobra.ExactArgs(1),
	}
	describeCmd.Flags().StringVarP(&target, "target", "t", "preview", "the target platform to log in: 'preview' or 'production', to retrieve the public key to verify the signature")
	return describeCmd
}

const (
	previewKeysURL    string = "https://auth.prod-preview.openshift.io/"
	productionKeysURL string = "https://auth.openshift.io/"
)

func describe(cmd *cobra.Command, args []string) {

	var targetURL string
	switch target {
	case "production":
		targetURL = productionKeysURL
	default:
		targetURL = previewKeysURL
	}
	config := getConfig(targetURL)
	tokenManager, err := auth.DefaultManager(config)
	if err != nil {
		logrus.WithError(err).Fatal("failed to parse token")
	}
	t, err := tokenManager.Parse(context.Background(), args[0])
	if err != nil {
		logrus.WithError(err).Fatal("failed to parse token")
	}
	b, err := json.MarshalIndent(t.Claims, "", "  ")
	if err == nil {
		fmt.Println(string(b))
	}
}

func getConfig(targetURL string) auth.ManagerConfiguration {
	return configuration{
		targetURL: targetURL,
	}
}

type configuration struct {
	targetURL string
}

func (c configuration) GetAuthServiceURL() string {
	return c.targetURL
}

func (c configuration) GetDevModePrivateKey() []byte {
	return nil
}
