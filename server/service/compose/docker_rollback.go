package compose

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"

	"isrvd/pkgs/compose"
	"isrvd/pkgs/docker"
)

// dockerPreviousContainers 仅选择有项目标签的容器，以及反推配置时明确选中的单个容器。
func (s *Service) dockerPreviousContainers(ctx context.Context, current *ConfigDetail) ([]container.InspectResponse, error) {
	list, err := s.docker.ContainerListByLabel(ctx, compose.ComposeProjectLabel, current.ProjectName)
	if err != nil {
		return nil, err
	}
	list = slices.DeleteFunc(list, func(info container.InspectResponse) bool {
		return compose.DockerComposeProjectName(info) != current.ProjectName ||
			strings.EqualFold(info.Config.Labels[compose.ComposeOneoffLabel], "true")
	})
	if current.standaloneID != "" {
		info, err := s.docker.ContainerInspect(ctx, current.standaloneID)
		if err != nil {
			return nil, err
		}
		list = append(list, info)
	}
	for _, info := range list {
		if info.Config == nil || info.HostConfig == nil || info.State == nil || info.Image == "" {
			return nil, fmt.Errorf("容器 inspect 数据不完整: %s", info.ID)
		}
	}
	slices.SortFunc(list, func(a, b container.InspectResponse) int { return cmp.Compare(a.Created, b.Created) })
	return list, nil
}

func (s *Service) dockerRemovePrevious(ctx context.Context, previous []container.InspectResponse) ([]container.InspectResponse, error) {
	removed := make([]container.InspectResponse, 0, len(previous))
	for _, item := range previous {
		if err := s.docker.ContainerAction(ctx, item.ID, "remove"); err != nil {
			check, cancel := cleanupContext(ctx)
			_, inspectErr := s.docker.ContainerInspect(check, item.ID)
			cancel()
			if client.IsErrNotFound(inspectErr) {
				removed = append(removed, item)
			} else if inspectErr != nil {
				err = errors.Join(err, inspectErr)
			}
			return removed, fmt.Errorf("移除容器 %s 失败: %w", docker.ShortID(item.ID), err)
		}
		removed = append(removed, item)
	}
	return removed, nil
}

// dockerRestore 只恢复本次已移除的容器，并使用其镜像 ID 与运行状态。
func (s *Service) dockerRestore(ctx context.Context, removed []container.InspectResponse) error {
	if len(removed) == 0 {
		return nil
	}
	cleanup, cancel := cleanupContext(ctx)
	defer cancel()
	var failures []error
	for _, old := range removed {
		config := *old.Config
		config.Image = old.Image
		name := strings.TrimPrefix(old.Name, "/")
		networks := &network.NetworkingConfig{EndpointsConfig: map[string]*network.EndpointSettings{}}
		if old.NetworkSettings != nil {
			for key, endpoint := range old.NetworkSettings.Networks {
				if endpoint != nil {
					networks.EndpointsConfig[key] = &network.EndpointSettings{
						IPAMConfig: endpoint.IPAMConfig, Aliases: endpoint.Aliases, Links: endpoint.Links,
						DriverOpts: endpoint.DriverOpts, GwPriority: endpoint.GwPriority,
					}
				}
			}
		}
		host := restoreDockerVolumes(old)
		id, err := s.docker.ContainerCreate(cleanup, name, &config, &host, networks)
		if err == nil && old.State.Running {
			err = s.docker.ContainerStart(cleanup, id)
			if err == nil && old.State.Paused {
				err = s.docker.ContainerAction(cleanup, id, "pause")
			}
		}
		if err != nil {
			failures = append(failures, fmt.Errorf("恢复容器 %s 失败: %w", name, err))
		}
	}
	return errors.Join(failures...)
}

// restoreDockerVolumes 复用旧容器实际挂载的匿名卷，避免重建时分配空卷。
func restoreDockerVolumes(old container.InspectResponse) container.HostConfig {
	host := *old.HostConfig
	host.Mounts, host.Binds = slices.Clone(host.Mounts), slices.Clone(host.Binds)
	for _, volume := range old.Mounts {
		if volume.Type != mount.TypeVolume || volume.Name == "" {
			continue
		}
		if index := slices.IndexFunc(host.Mounts, func(m mount.Mount) bool { return m.Target == volume.Destination }); index >= 0 {
			host.Mounts[index].Source = volume.Name
			continue
		}
		bind := volume.Name + ":" + volume.Destination
		if index := slices.Index(host.Binds, volume.Destination); index >= 0 {
			host.Binds[index] = bind
		} else if !slices.ContainsFunc(host.Binds, func(value string) bool { return value == bind || strings.HasPrefix(value, bind+":") }) {
			host.Mounts = append(host.Mounts, mount.Mount{Type: mount.TypeVolume, Source: volume.Name, Target: volume.Destination, ReadOnly: !volume.RW})
		}
	}
	return host
}
