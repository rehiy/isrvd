package registry

import (
	"context"
	"fmt"

	"github.com/docker/docker/client"
	"github.com/rehiy/pango/logman"

	"isrvd/config"
	"isrvd/pkgs/docker"
	"isrvd/pkgs/swarm"
)

// initDocker 初始化 Docker 服务
func (r *Registry) initDocker() error {
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

	r.dockerService = svc
	r.dockerClient = svc.GetClient()
	r.swarmManager = swarm.NewSwarmManager(r.dockerClient)

	return nil
}

// GetDocker 获取 Docker 服务实例
func (r *Registry) GetDocker() *docker.DockerService {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.dockerService
}

// GetSwarm 获取 Swarm 管理器实例
func (r *Registry) GetSwarm() *swarm.SwarmManager {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.swarmManager
}

// GetDockerClient 获取原始 Docker 客户端
func (r *Registry) GetDockerClient() *client.Client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.dockerClient
}

// IsDockerAvailable 检查 Docker 是否可用
func (r *Registry) IsDockerAvailable(ctx context.Context) bool {
	r.mu.RLock()
	svc := r.dockerService
	r.mu.RUnlock()

	if svc == nil {
		return false
	}
	_, err := svc.GetInfo(ctx)
	return err == nil
}

// IsSwarmAvailable 检查 Swarm 是否可用
func (r *Registry) IsSwarmAvailable(ctx context.Context) bool {
	r.mu.RLock()
	mgr := r.swarmManager
	r.mu.RUnlock()

	if mgr == nil {
		return false
	}
	_, err := mgr.GetClient().SwarmInspect(ctx)
	return err == nil
}


