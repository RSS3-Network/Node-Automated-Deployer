package cmd

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"os"
	"path"
	"strings"

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

		if cfg.Discovery.Server.AccessToken == "" {
			generatedAccessToken := "sk-" + randomString(32)
			err = patchConfigFileWithAccessToken(file, generatedAccessToken)
			if err != nil {
				return err
			}
		}

		composeFile := compose.NewCompose(
			compose.WithWorkers(cfg.Component.Decentralized),
			compose.SetDependsOnCRDB(),
			compose.SetNodeVersion(version),
			compose.SetNodeVolume(),
			compose.SetRestartPolicy(),
		)

		var b bytes.Buffer
		e := yaml.NewEncoder(&b)
		e.SetIndent(2)

		if err := e.Encode(composeFile); err != nil {
			return err
		}

		// Remove null values in the yaml output
		output := strings.ReplaceAll(b.String(), " null", "")
		fmt.Println(output)

		return nil
	},
}

func randomString(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"

	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}

	for i := range b {
		b[i] = letterBytes[int(b[i])%len(letterBytes)]
	}

	return string(b)
}

func patchConfigFileWithAccessToken(file string, accessToken string) error {
	discovered, err := discoverConfigFile(file)
	if err != nil {
		return fmt.Errorf("patch config file with generated access token, discover config file, %w", err)
	}

	f, err := os.Open(discovered)
	if err != nil {
		return fmt.Errorf("patch config file with generated access token, open config file, %w", err)
	}

	// we do not unmashal DSL node config.File directly because it does not have yaml struct tags and it does not play well with yaml encoder
	// as a workaround, we unmarshal the file into yaml.Node then manually patch the access token
	var root yaml.Node
	if err = yaml.NewDecoder(f).Decode(&root); err != nil {
		return fmt.Errorf("patch config file with generated access token, decode config file, %w", err)
	}

	if len(root.Content) > 0 {
		discoveryNode, err := findYamlNode("discovery", root.Content[0])
		if err != nil {
			return fmt.Errorf("patch config file with generated access token, find discovery node, %w", err)
		}

		if discoveryNode == nil {
			return fmt.Errorf("patch config file with generated access token, discovery node not found")
		}

		serverNode, err := findYamlNode("server", discoveryNode)
		if err != nil {
			return fmt.Errorf("patch config file with generated access token, find server node, %w", err)
		}

		if serverNode == nil {
			return fmt.Errorf("patch config file with generated access token, server node not found")
		}

		accessTokenNode, err := findYamlNode("access_token", serverNode)
		if err != nil {
			return fmt.Errorf("patch config file with generated access token, find access_token node, %w", err)
		}

		if accessTokenNode == nil {
			serverNode.Content = append(serverNode.Content, &yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
				Value: "access_token",
			})
			serverNode.Content = append(serverNode.Content, &yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
				Value: accessToken,
			})
		} else {
			accessTokenNode.Kind = yaml.ScalarNode
			accessTokenNode.Tag = "!!str"
			accessTokenNode.Value = accessToken
		}
	}

	// dump patched yaml node to file
	if err = f.Close(); err != nil {
		return fmt.Errorf("patch config file with generated access token, close config file, %w", err)
	}

	f, err = os.OpenFile(discovered, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("patch config file with generated access token, open config file, %w", err)
	}

	err = yaml.NewEncoder(f).Encode(&root)
	if err != nil {
		return fmt.Errorf("patch config file with generated access token, encode config file, %w", err)
	}

	f.Close()

	return nil
}

func discoverConfigFile(file string) (string, error) {
	_, err := os.Stat(file)
	if err == nil {
		return file, nil
	}

	if os.IsNotExist(err) {
		_, err = os.Stat(path.Join("config", file))
		if err == nil {
			return path.Join("config", file), nil
		}

		if os.IsNotExist(err) {
			return "", fmt.Errorf("config file %s not found", file)
		}

		return "", err
	}

	return "", err
}

func findYamlNode(fieldName string, parent *yaml.Node) (*yaml.Node, error) {
	if parent == nil {
		return nil, fmt.Errorf("find yaml node with field %s, parent node is nil", fieldName)
	}

	for i, node := range parent.Content {
		if node.Value == fieldName {
			return parent.Content[i+1], nil
		}
	}

	return nil, nil
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&file, "file", "f", file, "Specify the config.yaml file (default: config.yaml)")
}
