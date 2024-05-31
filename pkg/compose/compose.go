package compose

import (
	"fmt"
	"strings"

	"github.com/rss3-network/node/config"
)

type Compose struct {
	Services map[string]Service
	Volumes  map[string]*string
}

type Service struct {
	ContainerName string            `yaml:"container_name,omitempty"`
	Image         string            `yaml:"image"`
	Ports         []string          `yaml:"ports,omitempty"`
	Environment   map[string]string `yaml:"environment,omitempty"`
	Volumes       []string          `yaml:"volumes,omitempty"`
	Command       string            `yaml:"command,omitempty"`
	Restart       string            `yaml:"restart,omitempty"`
}

type Option func(*Compose)

func NewCompose(options ...Option) *Compose {
	// use a prefix to avoid conflict with other containers
	prefix := "rss3_node"
	cockroachdbVolume := "cockroachdb"

	compose := &Compose{
		Services: map[string]Service{
			fmt.Sprintf("%s_redis", prefix): {
				ContainerName: fmt.Sprintf("%s_redis", prefix),
				Image:         "redis:7-alpine",
			},
			fmt.Sprintf("%s_cockroachdb", prefix): {
				ContainerName: fmt.Sprintf("%s_cockroachdb", prefix),
				Image:         "cockroachdb/cockroach:v23.2.5",
				Ports:         []string{"26257:26257", "8080:8080"},
				Volumes:       []string{fmt.Sprintf("%s:/cockroach/cockroach-data", cockroachdbVolume)},
				Command:       "start-single-node --cluster-name=node --insecure",
			},
			fmt.Sprintf("%s_core", prefix): {
				ContainerName: fmt.Sprintf("%s_core", prefix),
				Image:         "rss3/node:latest",
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
				services[k] = Service{
					ContainerName: v.ContainerName,
					Image:         fmt.Sprintf("rss3/node:%s", version),
					Environment:   v.Environment,
					Volumes:       v.Volumes,
					Command:       v.Command,
				}
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
				services[k] = Service{
					ContainerName: v.ContainerName,
					Image:         v.Image,
					Environment:   v.Environment,
					Volumes:       []string{"./config:/etc/rss3"},
					Command:       v.Command,
				}
			}
		}

		c.Services = services
	}
}

func SetRestartPolicy() Option {
	return func(c *Compose) {
		services := c.Services
		for k, v := range services {
			services[k] = Service{
				ContainerName: v.ContainerName,
				Image:         v.Image,
				Environment:   v.Environment,
				Volumes:       []string{"${PWD}/config:/etc/rss3/node"},
				Command:       v.Command,
				Restart:       "unless-stopped",
			}
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
				ContainerName: name,
				Image:         "rss3/node",
				Command:       fmt.Sprintf("--module=worker --worker.id=%s", worker.ID),
			}
		}

		c.Services = services
	}
}
