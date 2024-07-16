package compose

import (
	"fmt"
	"strings"
	"time"

	"github.com/rss3-network/node/config"
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
	cockroachdbVolume := "cockroachdb"

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
			fmt.Sprintf("%s_cockroachdb", dockerComposeContainerNamePrefix): {
				Command:       "start-single-node --cluster-name=node --insecure",
				ContainerName: fmt.Sprintf("%s_cockroachdb", dockerComposeContainerNamePrefix),
				Expose:        []string{"26257", "8080"},
				Image:         "cockroachdb/cockroach:v23.2.5",
				Volumes:       []string{fmt.Sprintf("%s:/cockroach/cockroach-data", cockroachdbVolume)},
				// we use similar healthcheck as the official cockroachdb operator
				// ref: https://github.com/cockroachdb/cockroach-operator/blob/28d139cb0c19d3c7984b2b2da1b25c5ba388d814/pkg/resource/testdata/TestStatefulSetBuilder/default_secure.golden#L76-L83
				Healthcheck: Healthcheck{
					Test:     []string{"CMD", "curl", "-f", "http://localhost:8080/health?ready=1"},
					Interval: 5 * time.Second,
					Timeout:  1 * time.Second,
					Retries:  3,
				},
			},
			fmt.Sprintf("%s_core", dockerComposeContainerNamePrefix): {
				Command:       "--module=core",
				ContainerName: fmt.Sprintf("%s_core", dockerComposeContainerNamePrefix),
				Ports:         []string{"8080:80"},
				Image:         "rss3/node",
			},
			fmt.Sprintf("%s_monitor", prefix): {
				Command:       "--module=monitor",
				ContainerName: fmt.Sprintf("%s_monitor", prefix),
				Image:         "rss3/node",
			},
			fmt.Sprintf("%s_broadcaster", prefix): {
				Command:       "--module=broadcaster",
				ContainerName: fmt.Sprintf("%s_broadcaster", prefix),
				Image:         "rss3/node",
			},
		},
		Volumes: map[string]*string{
			cockroachdbVolume: nil,
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
		}

		c.Services = services
	}
}

// SetDependsOnCRDB would set all the rss3 node service to depend on the cockroachdb service
func SetDependsOnCRDB() Option {
	return func(c *Compose) {
		services := c.Services
		for k, v := range services {
			if strings.Contains(v.Image, "rss3/node") {
				v.DependsOn = map[string]DependsOn{
					fmt.Sprintf("%s_cockroachdb", dockerComposeContainerNamePrefix): {
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
