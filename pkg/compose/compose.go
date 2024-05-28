package compose

import (
	"fmt"
	"strings"

	"github.com/rss3-network/node/config"
)

type Compose struct {
	Version  string
	Services map[string]Service
}

type Service struct {
	ContainerName string            `yaml:"container_name,omitempty"`
	Image         string            `yaml:"image"`
	Ports         []string          `yaml:"ports,omitempty"`
	Environment   map[string]string `yaml:"environment,omitempty"`
	Volumes       []string          `yaml:"volumes,omitempty"`
	Command       string            `yaml:"command,omitempty"`
}

type Option func(*Compose)

func NewCompose(options ...Option) *Compose {
	compose := &Compose{
		Version: "3.8",
		Services: map[string]Service{
			"redis": {
				ContainerName: "redis",
				Image:         "redis:7-alpine",
			},
			"cockroachdb": {
				ContainerName: "cockroachdb",
				Image:         "cockroachdb/cockroach:v23.2.5",
				Ports:         []string{"26257:26257", "8080:8080"},
				Volumes:       []string{"cockroachdb:/cockroach/cockroach-data"},
				Command:       "start-single-node --cluster-name=node --insecure",
			},
			"core": {
				ContainerName: "core",
				Image:         "rss3/node:latest",
			},
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
