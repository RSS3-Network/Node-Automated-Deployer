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
	Command       string            `yaml:"command,omitempty"`
	ContainerName string            `yaml:"container_name,omitempty"`
	Environment   map[string]string `yaml:"environment,omitempty"`
	Expose        []string          `yaml:"expose,omitempty"`
	Image         string            `yaml:"image"`
	Restart       string            `yaml:"restart,omitempty"`
	Ports         []string          `yaml:"ports,omitempty"`
	Volumes       []string          `yaml:"volumes,omitempty"`
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
				Expose:        []string{"6379"},
				Image:         "redis:7-alpine",
			},
			fmt.Sprintf("%s_cockroachdb", prefix): {
				Command:       "start-single-node --cluster-name=node --insecure",
				ContainerName: fmt.Sprintf("%s_cockroachdb", prefix),
				Expose:        []string{"26257", "8080"},
				Image:         "cockroachdb/cockroach:v23.2.5",
				Volumes:       []string{fmt.Sprintf("%s:/cockroach/cockroach-data", cockroachdbVolume)},
			},
			fmt.Sprintf("%s_core", prefix): {
				Command:       "--module=core",
				ContainerName: fmt.Sprintf("%s_core", prefix),
				Ports:         []string{"8080:80"},
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
