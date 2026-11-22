package compose

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/docker/client"

	"isrvd/pkgs/compose"
)

// ensureNetworks 先检查所有引用，再创建缺失的托管网络。返回本次确认创建的完整 ID。
func (s *Service) ensureNetworks(ctx context.Context, project *types.Project, defaultDriver string) ([]string, error) {
	networks := compose.ProjectNetworks(project)
	missing := make([]string, 0, len(networks))
	for _, name := range slices.Sorted(maps.Keys(networks)) {
		_, err := s.docker.NetworkInspect(ctx, name)
		if err == nil {
			continue
		}
		if !client.IsErrNotFound(err) || bool(networks[name].External) {
			return nil, fmt.Errorf("检查网络 %s 失败: %w", name, err)
		}
		missing = append(missing, name)
	}
	var created []string
	for _, name := range missing {
		id, err := s.docker.NetworkCreate(ctx, name, cmp.Or(networks[name].Driver, defaultDriver), "")
		if err != nil {
			return created, fmt.Errorf("创建网络 %s 失败: %w", name, err)
		}
		created = append(created, id)
	}
	return created, nil
}

// rollbackNetworks 在容器或服务回滚之后清理本次创建的网络。
func (s *Service) rollbackNetworks(ctx context.Context, ids []string, resultErr *error) {
	if *resultErr == nil || len(ids) == 0 {
		return
	}
	cleanup, cancel := cleanupContext(ctx)
	defer cancel()
	for _, id := range ids {
		if err := s.docker.NetworkAction(cleanup, id, "remove"); err != nil && !client.IsErrNotFound(err) {
			*resultErr = errors.Join(*resultErr, fmt.Errorf("清理网络 %s 失败: %w", id, err))
		}
	}
}
