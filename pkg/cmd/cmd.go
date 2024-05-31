package cmd

import (
	"bytes"
	"fmt"

	"github.com/rss3-network/node-automated-deployer/pkg/compose"
	"github.com/rss3-network/node/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	file = "config.yaml"
)

var rootCmd = cobra.Command{
	Use:   "compose",
	Short: "Compose is a tool for defining and running multi-container Docker applications.",
	Long: `Compose is a tool for defining and running multi-container Docker applications.
With Compose, you use a YAML file to configure your application's services.
Then, with a single command, you create and start all the services from your configuration.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		// read config file

		cfg, err := config.Setup(file)
		if err != nil {
			return err
		}

		version, err := compose.NodeVersion()
		if err != nil {
			return err
		}

		composeFile := compose.NewCompose(
			compose.WithWorkers(cfg.Component.Decentralized),
			compose.SetNodeVersion(version),
			compose.SetNodeVolume(),
		)

		var b bytes.Buffer
		e := yaml.NewEncoder(&b)
		e.SetIndent(2)

		if err := e.Encode(composeFile); err != nil {
			return err
		}

		fmt.Println(b.String())

		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&file, "file", "f", file, "Specify the config.yaml file (default: config.yaml)")
}
