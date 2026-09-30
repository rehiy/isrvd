package apisix

import (
	"context"
	"fmt"

	"github.com/rehiy/libgo/strutil"

	"isrvd/pkgs/apisix"
)

// PluginConfigList 获取 Plugin Config 列表
func (s *Service) PluginConfigList(ctx context.Context) ([]apisix.PluginConfig, error) {
	list, err := s.client.PluginConfigList(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取插件配置列表失败: %w", err)
	}
	return list, nil
}

// PluginConfigInspect 获取单个 Plugin Config 详情
func (s *Service) PluginConfigInspect(ctx context.Context, configID string) (*apisix.PluginConfig, error) {
	if configID == "" {
		return nil, fmt.Errorf("Plugin Config ID 不能为空")
	}
	config, err := s.client.PluginConfigInspect(ctx, configID)
	if err != nil {
		return nil, fmt.Errorf("获取插件配置详情失败: %w", err)
	}
	return config, nil
}

// PluginConfigCreate 创建 Plugin Config
func (s *Service) PluginConfigCreate(ctx context.Context, req apisix.PluginConfig) (*apisix.PluginConfig, error) {
	if len(req.Plugins) == 0 {
		return nil, fmt.Errorf("插件配置不能为空")
	}
	req.ID = strutil.NewString()
	config, err := s.client.PluginConfigCreate(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("创建插件配置失败: %w", err)
	}
	return config, nil
}

// PluginConfigUpdate 更新 Plugin Config
func (s *Service) PluginConfigUpdate(ctx context.Context, configID string, req apisix.PluginConfig) (*apisix.PluginConfig, error) {
	if configID == "" {
		return nil, fmt.Errorf("Plugin Config ID 不能为空")
	}
	if len(req.Plugins) == 0 {
		return nil, fmt.Errorf("插件配置不能为空")
	}
	config, err := s.client.PluginConfigUpdate(ctx, configID, req)
	if err != nil {
		return nil, fmt.Errorf("更新插件配置失败: %w", err)
	}
	return config, nil
}

// PluginConfigDelete 删除 Plugin Config
func (s *Service) PluginConfigDelete(ctx context.Context, configID string) error {
	if configID == "" {
		return fmt.Errorf("Plugin Config ID 不能为空")
	}
	if err := s.client.PluginConfigDelete(ctx, configID); err != nil {
		return fmt.Errorf("删除插件配置失败: %w", err)
	}
	return nil
}
