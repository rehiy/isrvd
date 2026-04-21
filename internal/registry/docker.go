package registry

import (
	"context"
	"fmt"

	"github.com/rehiy/pango/logman"

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
	SwarmService = swarm.NewSwarmService(svc.GetClient())

	return nil
}

// IsDockerAvailable 检查 Docker 是否可用
func IsDockerAvailable(ctx context.Context) bool {
	if DockerService == nil {
		logman.Warn("Docker service not initialized")
		return false
	}

	_, err := DockerService.GetInfo(ctx)
	if err != nil {
		logman.Error("Docker service not available", "error", err)
		return false
	}
	return true
}

// IsSwarmAvailable 检查 Swarm 是否可用
func IsSwarmAvailable(ctx context.Context) bool {
	if SwarmService == nil {
		logman.Warn("Swarm service not initialized")
		return false
	}

	_, err := SwarmService.GetClient().SwarmInspect(ctx)
	if err != nil {
		logman.Error("Swarm not available", "error", err)
		return false
	}
	return true
}
