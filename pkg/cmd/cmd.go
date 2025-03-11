package cmd

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

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

		err = patchFileSetDatabaseConnectionURI(file, "postgres://postgres:password@rss3_node_alloydb:5432/postgres")
		if err != nil {
			return err
		}

		// Check if the AI endpoint is healthy by reading directly from the config file
		endpoint, err := readAIComponentEndpoint(file)
		if err != nil {
			return err
		}

		isAIEndpointHealthy := false
		if endpoint != "" {
			isAIEndpointHealthy = checkAIEndpointHealth(endpoint)
		}

		composeFile := compose.NewCompose(
			compose.WithWorkers(cfg.Component.Decentralized),
			compose.WithWorkers(cfg.Component.Federated),
			compose.SetDependsOnAlloyDB(),
			compose.SetNodeVersion(version),
			compose.SetNodeVolume(),
			compose.SetRestartPolicy(),
			compose.SetAIComponent(cfg, isAIEndpointHealthy),
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

// readConfigFile reads a configuration file and returns the parsed YAML content
// It supports two different output formats:
// 1. As a yaml.Node structure (for direct node manipulation)
// 2. As a map[string]interface{} (for simpler key-value access)
func readConfigFile(file string) (string, *yaml.Node, map[string]interface{}, error) {
	// Locate the configuration file
	discovered, err := discoverConfigFile(file)
	if err != nil {
		return "", nil, nil, fmt.Errorf("read config file, discover config file, %w", err)
	}

	// Open the file
	f, err := os.Open(discovered)
	if err != nil {
		return "", nil, nil, fmt.Errorf("read config file, open config file, %w", err)
	}
	defer f.Close()

	// Read as yaml.Node for node-based manipulation
	var rootNode yaml.Node
	// Create a copy of the file content to parse it in two different ways
	fileContent, err := io.ReadAll(f)
	if err != nil {
		return "", nil, nil, fmt.Errorf("read config file, read file content, %w", err)
	}

	// Parse as yaml.Node
	if err = yaml.Unmarshal(fileContent, &rootNode); err != nil {
		return "", nil, nil, fmt.Errorf("read config file, decode as yaml node, %w", err)
	}

	// Parse as map[string]interface{}
	var configMap map[string]interface{}
	if err = yaml.Unmarshal(fileContent, &configMap); err != nil {
		return "", nil, nil, fmt.Errorf("read config file, decode as map, %w", err)
	}

	return discovered, &rootNode, configMap, nil
}

// writeConfigFile writes the YAML node back to the configuration file
func writeConfigFile(filePath string, rootNode *yaml.Node) error {
	// Open the file for writing
	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("write config file, open file, %w", err)
	}
	defer f.Close()

	// Encode and write to file
	err = yaml.NewEncoder(f).Encode(rootNode)
	if err != nil {
		return fmt.Errorf("write config file, encode yaml, %w", err)
	}

	return nil
}

func patchFileSetDatabaseConnectionURI(file string, newConnectionURI string) error {
	discovered, rootNode, _, err := readConfigFile(file)
	if err != nil {
		return fmt.Errorf("patch config file with new database connection uri, %w", err)
	}

	if len(rootNode.Content) > 0 {
		databaseNode, err := findYamlNode("database", rootNode.Content[0])
		if err != nil {
			return fmt.Errorf("patch config file with new database connection uri, find database node, %w", err)
		}

		if databaseNode == nil {
			return fmt.Errorf("patch config file with new database connection uri, database node not found")
		}

		uriNode, err := findYamlNode("uri", databaseNode)
		if err != nil {
			return fmt.Errorf("patch config file with new database connection uri, find uri node, %w", err)
		}

		if uriNode == nil {
			return fmt.Errorf("patch config file with new database connection uri, uri node not found")
		}

		uriNode.Kind = yaml.ScalarNode
		uriNode.Tag = "!!str"
		uriNode.Value = newConnectionURI
	}

	// Write the updated config back to file
	return writeConfigFile(discovered, rootNode)
}

func patchConfigFileWithAccessToken(file string, accessToken string) error {
	discovered, rootNode, _, err := readConfigFile(file)
	if err != nil {
		return fmt.Errorf("patch config file with generated access token, %w", err)
	}

	if len(rootNode.Content) > 0 {
		discoveryNode, err := findYamlNode("discovery", rootNode.Content[0])
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

	// Write the updated config back to file
	return writeConfigFile(discovered, rootNode)
}

// checkAIEndpointHealth verifies if the provided AI endpoint is responsive and operational.
// It performs multiple attempts to account for potential network issues.
func checkAIEndpointHealth(endpoint string) bool {
	if endpoint == "" {
		return false
	}

	// Normalize endpoint URL
	endpoint = normalizeEndpointURL(endpoint)
	healthURL := endpoint + "api/v1/health"

	// Configure HTTP client with reasonable timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Perform multiple attempts to mitigate transient network issues
	const maxRetries = 3

	const retryInterval = time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Create a context for the HTTP request
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Use the context-aware Get method
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
		if err != nil {
			time.Sleep(retryInterval)
			continue
		}

		resp, err := client.Do(req)
		if err == nil {
			defer resp.Body.Close()
			return resp.StatusCode == http.StatusOK
		}

		// Wait before next retry
		time.Sleep(retryInterval)
	}

	return false
}

// normalizeEndpointURL ensures the endpoint URL has proper protocol and trailing slash
func normalizeEndpointURL(url string) string {
	// Add protocol if missing
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	// Add trailing slash if missing
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}

	return url
}

// readAIComponentEndpoint extracts the AI component endpoint from the configuration file.
// Returns an empty string if the configuration file doesn't contain an AI component endpoint.
func readAIComponentEndpoint(configFile string) (string, error) {
	_, _, configMap, err := readConfigFile(configFile)
	if err != nil {
		return "", fmt.Errorf("failed to read config file: %w", err)
	}

	// Extract the AI component endpoint using safe type assertions
	component, ok := configMap["component"].(map[string]interface{})
	if !ok {
		return "", nil
	}

	ai, ok := component["ai"].(map[string]interface{})
	if !ok {
		return "", nil
	}

	endpoint, ok := ai["endpoint"].(string)
	if !ok {
		return "", nil
	}

	return endpoint, nil
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
