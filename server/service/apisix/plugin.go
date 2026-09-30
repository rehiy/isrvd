package apisix

import (
	"context"
	"fmt"
)

// PluginList 获取可用插件列表
func (s *Service) PluginList(ctx context.Context) (any, error) {
	list, err := s.client.PluginList(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取插件列表失败: %w", err)
	}
	return list, nil
}
