package apisix

import (
	"context"
	"fmt"

	"isrvd/pkgs/apisix"
)

// ConsumerList 获取 Consumer 列表
func (s *Service) ConsumerList(ctx context.Context) ([]apisix.Consumer, error) {
	list, err := s.client.ConsumerList(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取消费者列表失败: %w", err)
	}
	return list, nil
}

// ConsumerCreate 创建 Consumer，支持传入完整 plugins
func (s *Service) ConsumerCreate(ctx context.Context, username, desc string, plugins map[string]any) (*apisix.Consumer, error) {
	if username == "" {
		return nil, fmt.Errorf("用户名不能为空")
	}
	consumer, err := s.client.ConsumerCreate(ctx, username, desc, plugins)
	if err != nil {
		return nil, fmt.Errorf("创建消费者失败: %w", err)
	}
	return consumer, nil
}

// ConsumerUpdate 更新 Consumer（支持 plugins，自动替换脱敏值）
func (s *Service) ConsumerUpdate(ctx context.Context, username, desc string, plugins map[string]any) error {
	if username == "" {
		return fmt.Errorf("用户名不能为空")
	}
	if err := s.client.ConsumerUpdate(ctx, username, desc, plugins); err != nil {
		return fmt.Errorf("更新消费者失败: %w", err)
	}
	return nil
}

// ConsumerDelete 删除 Consumer
func (s *Service) ConsumerDelete(ctx context.Context, username string) error {
	if username == "" {
		return fmt.Errorf("用户名不能为空")
	}
	if err := s.client.ConsumerDelete(ctx, username); err != nil {
		return fmt.Errorf("删除消费者失败: %w", err)
	}
	return nil
}
