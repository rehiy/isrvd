package compose

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"

	"isrvd/pkgs/compose"
)

// dockerReuseAnonymousVolumes 为新配置中仍是匿名卷的目标复用旧服务的数据卷。
func (s *Service) dockerReuseAnonymousVolumes(ctx context.Context, specs []compose.DockerCreateSpec, previous []container.InspectResponse) error {
	const anonymousLabel = "com.docker.volume.anonymous"
	for _, spec := range specs {
		index := dockerPreviousService(spec, previous)
		if index < 0 {
			continue
		}
		mounts := previous[index].Mounts
		for i, target := range spec.HostConfig.Mounts {
			if target.Type != mount.TypeVolume || target.Source != "" {
				continue
			}
			old := slices.IndexFunc(mounts, func(volume container.MountPoint) bool {
				return volume.Type == mount.TypeVolume && volume.Destination == target.Target && volume.Name != ""
			})
			if old < 0 {
				continue
			}
			volume, err := s.docker.VolumeInspect(ctx, mounts[old].Name)
			if err != nil {
				return fmt.Errorf("检查容器 %s 的数据卷失败: %w", spec.Name, err)
			}
			if _, anonymous := volume.Labels[anonymousLabel]; anonymous {
				spec.HostConfig.Mounts[i].Source = mounts[old].Name
			}
		}
	}
	return nil
}

// 同名同服务优先；容器改名时用服务标签与主副本编号匹配，该候选必须唯一。
func dockerPreviousService(spec compose.DockerCreateSpec, previous []container.InspectResponse) int {
	service := spec.Config.Labels[compose.ComposeServiceLabel]
	index := slices.IndexFunc(previous, func(old container.InspectResponse) bool {
		oldService := old.Config.Labels[compose.ComposeServiceLabel]
		return strings.TrimPrefix(old.Name, "/") == spec.Name && (oldService == service || oldService == "")
	})
	if index >= 0 {
		return index
	}
	candidate := -1
	for i, old := range previous {
		labels := old.Config.Labels
		number := labels[compose.ComposeContainerNumberLabel]
		if labels[compose.ComposeServiceLabel] != service || (number != "1" && number != "") {
			continue
		}
		if candidate >= 0 {
			return -1
		}
		candidate = i
	}
	return candidate
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
