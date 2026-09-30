// Package apisix 提供 Apisix 业务服务层
package apisix

import (
	"context"
	"fmt"

	"isrvd/pkgs/apisix"
	"isrvd/server/config"
)

// Service Apisix 业务服务
type Service struct {
	client *apisix.Client
}

// NewService 创建 Apisix 业务服务
func NewService(ctx context.Context) (*Service, error) {
	cfg := config.Current().Apisix
	if cfg.AdminURL == "" {
		return nil, fmt.Errorf("Apisix 未配置")
	}
	client := apisix.NewClient(cfg.AdminURL, cfg.AdminKey)
	// 验证连通性，服务不可达时拒绝初始化
	if _, err := client.RouteList(ctx); err != nil {
		return nil, fmt.Errorf("Apisix 不可达: %w", err)
	}
	return &Service{client: client}, nil
}

// CheckAvailability 检测 Apisix 可用性
func (s *Service) CheckAvailability(ctx context.Context) bool {
	if s.client == nil {
		return false
	}
	_, err := s.client.RouteList(ctx)
	return err == nil
}
