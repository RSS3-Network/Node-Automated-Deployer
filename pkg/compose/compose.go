package compose

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/rss3-network/node/config"
	"github.com/rss3-network/node/schema/worker/federated"
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

type Option func(*Compose)

// use a prefix to avoid conflict with other containers
const dockerComposeContainerNamePrefix = "rss3_node"

func NewCompose(options ...Option) *Compose {
	alloydbVolume := "alloydb"

	compose := &Compose{
		Services: map[string]Service{
			fmt.Sprintf("%s_redis", dockerComposeContainerNamePrefix): {
				ContainerName: fmt.Sprintf("%s_redis", dockerComposeContainerNamePrefix),
				Expose:        []string{"6397"},
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
				Volumes:       []string{fmt.Sprintf("%s:/alloydb/alloydb-data", alloydbVolume)},
				Environment: map[string]string{
					"DATA_DIR":          "/alloydb/alloydb-data",
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
				Image:         "rss3/node",
			},
			fmt.Sprintf("%s_monitor", dockerComposeContainerNamePrefix): {
				Command:       "--module=monitor",
				ContainerName: fmt.Sprintf("%s_monitor", dockerComposeContainerNamePrefix),
				Image:         "rss3/node",
			},
			fmt.Sprintf("%s_broadcaster", dockerComposeContainerNamePrefix): {
				Command:       "--module=broadcaster",
				ContainerName: fmt.Sprintf("%s_broadcaster", dockerComposeContainerNamePrefix),
				Image:         "rss3/node",
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
			if strings.Contains(v.Image, "rss3/node") {
				v.Image = fmt.Sprintf("rss3/node:%s", version)
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
			if strings.Contains(v.Image, "rss3/node") {
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
				Image:         "rss3/node",
			}

			// set port for mastodon federated core
			if worker.Network == network.Mastodon && worker.Worker == federated.Core {
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
			if strings.Contains(v.Image, "rss3/node") {
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
