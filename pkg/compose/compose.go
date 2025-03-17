package compose

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/rss3-network/node/v2/config"
	"github.com/rss3-network/node/v2/schema/worker/federated"
	"github.com/rss3-network/protocol-go/schema/network"
)

type Compose struct {
	Services map[string]Service
	Volumes  map[string]*string
}

type Healthcheck struct {
	Test     []string      `yaml:"test,omitempty"`
	Interval time.Duration `yaml:"interval,omitempty"`
	Timeout  time.Duration `yaml:"timeout,omitempty"`
	Retries  int           `yaml:"retries,omitempty"`
}

type DependsOn struct {
	Condition string `yaml:"condition,omitempty"`
}

type Service struct {
	Command       string               `yaml:"command,omitempty"`
	ContainerName string               `yaml:"container_name,omitempty"`
	Environment   map[string]string    `yaml:"environment,omitempty"`
	Expose        []string             `yaml:"expose,omitempty"`
	Image         string               `yaml:"image"`
	Restart       string               `yaml:"restart,omitempty"`
	Ports         []string             `yaml:"ports,omitempty"`
	Volumes       []string             `yaml:"volumes,omitempty"`
	Healthcheck   Healthcheck          `yaml:"healthcheck,omitempty"`
	DependsOn     map[string]DependsOn `yaml:"depends_on,omitempty"`
}

type AIComponentParameters struct {
	OpenAIAPIKey  string            `json:"openai_api_key" mapstructure:"openai_api_key"`
	OllamaHost    string            `json:"ollama_host" mapstructure:"ollama_host"`
	KaitoAPIToken string            `json:"kaito_api_token" mapstructure:"kaito_api_token"`
	Twitter       TwitterParameters `json:"twitter" mapstructure:"twitter"`
}

type TwitterParameters struct {
	BearerToken       string `json:"bearer_token" mapstructure:"bearer_token"`
	APIKey            string `json:"api_key" mapstructure:"api_key"`
	APISecret         string `json:"api_secret" mapstructure:"api_secret"`
	AccessToken       string `json:"access_token" mapstructure:"access_token"`
	AccessTokenSecret string `json:"access_token_secret" mapstructure:"access_token_secret"`
}

type Option func(*Compose)

// use a prefix to avoid conflict with other containers
const dockerComposeContainerNamePrefix = "rss3_node"

func NewCompose(options ...Option) *Compose {
	alloydbVolume := "alloydb"

	compose := &Compose{
		Services: map[string]Service{
			fmt.Sprintf("%s_redis", dockerComposeContainerNamePrefix): {
				ContainerName: fmt.Sprintf("%s_redis", dockerComposeContainerNamePrefix),
				Expose:        []string{"6379"},
				Image:         "redis:7-alpine",
				Healthcheck: Healthcheck{
					Test:     []string{"CMD", "redis-cli", "ping"},
					Interval: 5 * time.Second,
					Timeout:  10 * time.Second,
					Retries:  3,
				},
			},
			fmt.Sprintf("%s_alloydb", dockerComposeContainerNamePrefix): {
				ContainerName: fmt.Sprintf("%s_alloydb", dockerComposeContainerNamePrefix),
				Expose:        []string{"5432"},
				Image:         "google/alloydbomni:latest",
				Volumes:       []string{fmt.Sprintf("%s:/var/lib/postgresql/data", alloydbVolume)},
				Environment: map[string]string{
					"DATA_DIR":          "/var/lib/postgresql/data",
					"HOST_PORT":         "5432",
					"POSTGRES_PASSWORD": "password",
				},
				Healthcheck: Healthcheck{
					Test:     []string{"CMD-SHELL", "pg_isready -U postgres"},
					Interval: 5 * time.Second,
					Timeout:  5 * time.Second,
					Retries:  5,
				},
			},
			fmt.Sprintf("%s_core", dockerComposeContainerNamePrefix): {
				Command:       "--module=core",
				ContainerName: fmt.Sprintf("%s_core", dockerComposeContainerNamePrefix),
				Ports:         []string{"8080:80"},
				Image:         "ghcr.io/rss3-network/node",
			},
			fmt.Sprintf("%s_monitor", dockerComposeContainerNamePrefix): {
				Command:       "--module=monitor",
				ContainerName: fmt.Sprintf("%s_monitor", dockerComposeContainerNamePrefix),
				Image:         "ghcr.io/rss3-network/node",
			},
			fmt.Sprintf("%s_broadcaster", dockerComposeContainerNamePrefix): {
				Command:       "--module=broadcaster",
				ContainerName: fmt.Sprintf("%s_broadcaster", dockerComposeContainerNamePrefix),
				Image:         "ghcr.io/rss3-network/node",
			},
		},
		Volumes: map[string]*string{
			alloydbVolume: nil,
		},
	}

	for _, option := range options {
		option(compose)
	}

	return compose
}

func SetNodeVersion(version string) Option {
	return func(c *Compose) {
		services := c.Services
		for k, v := range services {
			if strings.Contains(v.Image, "ghcr.io/rss3-network/node") {
				v.Image = fmt.Sprintf("ghcr.io/rss3-network/node:%s", version)
				c.Services[k] = v
			}
		}

		c.Services = services
	}
}

func SetNodeVolume() Option {
	return func(c *Compose) {
		services := c.Services
		for k, v := range services {
			if strings.Contains(v.Image, "ghcr.io/rss3-network/node") {
				v.Volumes = append(v.Volumes, "${PWD}/config:/etc/rss3/node")
				c.Services[k] = v
			}
		}

		c.Services = services
	}
}

func SetRestartPolicy() Option {
	return func(c *Compose) {
		services := c.Services

		for k, v := range services {
			v.Restart = "unless-stopped"
			c.Services[k] = v
		}

		c.Services = services
	}
}

type OptionParameter struct {
	Port int64 `json:"port"`
}

func WithWorkers(workers []*config.Module) Option {
	return func(c *Compose) {
		services := c.Services

		for _, worker := range workers {
			name := fmt.Sprintf("node-%s", worker.ID)
			services[name] = Service{
				Command:       fmt.Sprintf("--module=worker --worker.id=%s", worker.ID),
				ContainerName: name,
				Image:         "ghcr.io/rss3-network/node",
			}

			// set port for mastodon federated core
			if worker.Network == network.Mastodon && worker.Worker == federated.Mastodon {
				// default port
				var port int64 = 8181

				if optionParameter := new(OptionParameter); worker.Parameters.Decode(optionParameter) == nil && optionParameter.Port > 0 {
					port = optionParameter.Port
				}

				portStr := strconv.FormatInt(port, 10)
				service := services[name]
				service.Ports = []string{fmt.Sprintf("%s:%s", portStr, portStr)}
				services[name] = service
			}
		}

		c.Services = services
	}
}

// SetDependsOnAlloyDB would set all the rss3 node service to depend on the AlloyDB service
func SetDependsOnAlloyDB() Option {
	return func(c *Compose) {
		services := c.Services

		for k, v := range services {
			if strings.Contains(v.Image, "ghcr.io/rss3-network/node") {
				v.DependsOn = map[string]DependsOn{
					fmt.Sprintf("%s_alloydb", dockerComposeContainerNamePrefix): {
						Condition: "service_healthy",
					},
					fmt.Sprintf("%s_redis", dockerComposeContainerNamePrefix): {
						Condition: "service_healthy",
					},
				}

				c.Services[k] = v
			}
		}
	}
}

// SetAIComponent configures the AI component for the node services.
// If an external AI endpoint is provided and healthy, it's used directly.
// If no endpoint is provided or it's unhealthy, creates an agentdata service using the existing AlloyDB.
func SetAIComponent(cfg *config.File, isAIEndpointHealthy bool) Option {
	return func(c *Compose) {
		// Skip if AI endpoint is healthy or component not configured
		if isAIEndpointHealthy || cfg == nil || cfg.Component == nil || cfg.Component.AI == nil {
			return
		}

		// Find and validate the AlloyDB service
		alloydbServiceName := fmt.Sprintf("%s_alloydb", dockerComposeContainerNamePrefix)
		if _, exists := c.Services[alloydbServiceName]; !exists {
			log.Printf("Warning: AlloyDB service %s not found, cannot set up agentdata", alloydbServiceName)
			return
		}

		// Create base environment with database connection
		env := map[string]string{
			"DB_CONNECTION": fmt.Sprintf("postgresql://postgres:password@%s:5432/agent_data", alloydbServiceName),
		}

		// Extract and map AI parameters to environment variables if available
		if cfg.Component.AI.Parameters != nil {
			var params AIComponentParameters

			if err := cfg.Component.AI.Parameters.Decode(&params); err != nil {
				log.Printf("Warning: Failed to decode AI parameters: %v", err)
			} else {
				// Add AI-specific environment variables from parameters
				mapAIParamsToEnv(&params, env)
			}
		}

		// Create and configure the agentdata service
		agentdataServiceName := fmt.Sprintf("%s_agentdata", dockerComposeContainerNamePrefix)
		c.Services[agentdataServiceName] = Service{
			Image:         "ghcr.io/rss3-network/agentdata",
			ContainerName: agentdataServiceName,
			Restart:       "unless-stopped",
			Ports:         []string{"8887:8887"},
			Environment:   env,
			DependsOn: map[string]DependsOn{
				alloydbServiceName: {Condition: "service_healthy"},
			},
		}

		// Configure the AI endpoint for core RSS3 services
		configureAIEndpointForCoreServices(c, agentdataServiceName)
	}
}

// configureAIEndpointForCoreServices sets the AI endpoint environment variable
// for the core, monitor, and broadcaster services only
func configureAIEndpointForCoreServices(c *Compose, agentdataServiceName string) {
	// Target only these specific core services
	coreServices := []string{
		fmt.Sprintf("%s_core", dockerComposeContainerNamePrefix),
		fmt.Sprintf("%s_monitor", dockerComposeContainerNamePrefix),
		fmt.Sprintf("%s_broadcaster", dockerComposeContainerNamePrefix),
	}

	// Set the AI endpoint for each core service
	agentdataEndpoint := fmt.Sprintf("http://%s:8887", agentdataServiceName)

	for serviceName, service := range c.Services {
		// Skip services that aren't in our target list
		if !containsString(coreServices, serviceName) {
			continue
		}

		// Initialize environment map if needed
		if service.Environment == nil {
			service.Environment = make(map[string]string)
		}

		// Set the AI endpoint and update the service
		service.Environment["NODE_COMPONENT_AI_ENDPOINT"] = agentdataEndpoint
		c.Services[serviceName] = service
	}
}

// mapAIParamsToEnv adds non-empty AI parameters to the environment map
func mapAIParamsToEnv(params *AIComponentParameters, env map[string]string) {
	// Map AI service credentials
	if params.OpenAIAPIKey != "" {
		env["OPENAI_API_KEY"] = params.OpenAIAPIKey
	}

	if params.OllamaHost != "" {
		env["OLLAMA_HOST"] = params.OllamaHost
	}

	if params.KaitoAPIToken != "" {
		env["KAITO_API_TOKEN"] = params.KaitoAPIToken
	}

	// Map Twitter credentials
	twitter := params.Twitter
	if twitter.BearerToken != "" {
		env["TWITTER_BEARER_TOKEN"] = twitter.BearerToken
	}

	if twitter.APIKey != "" {
		env["TWITTER_API_KEY"] = twitter.APIKey
	}

	if twitter.APISecret != "" {
		env["TWITTER_API_SECRET"] = twitter.APISecret
	}

	if twitter.AccessToken != "" {
		env["TWITTER_ACCESS_TOKEN"] = twitter.AccessToken
	}

	if twitter.AccessTokenSecret != "" {
		env["TWITTER_ACCESS_TOKEN_SECRET"] = twitter.AccessTokenSecret
	}
}

// containsString checks if a string slice contains a specific string
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}

	return false
}
