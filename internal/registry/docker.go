package registry

import (
	"fmt"

	"github.com/rehiy/libgo/logman"

	"isrvd/config"
	"isrvd/pkgs/docker"
	"isrvd/pkgs/swarm"
)

var DockerService *docker.DockerService
var SwarmService *swarm.SwarmService

// initDocker 初始化 Docker 服务
func initDocker() error {
	var registries []*docker.RegistryConfig
	for _, reg := range config.Docker.Registries {
		registries = append(registries, &docker.RegistryConfig{
			Name:        reg.Name,
			Description: reg.Description,
			URL:         reg.URL,
			Username:    reg.Username,
			Password:    reg.Password,
		})
	}

	cfg := &docker.DockerConfig{
		Host:          config.Docker.Host,
		ContainerRoot: config.Docker.ContainerRoot,
		Registries:    registries,
	}

	svc, err := docker.NewDockerService(cfg)
	if err != nil {
		logman.Warn("Docker service initialization failed", "error", err)
		return fmt.Errorf("docker init failed: %w", err)
	}

	DockerService = svc
	SwarmService = swarm.NewSwarmService(svc.Client())

	return nil
}
