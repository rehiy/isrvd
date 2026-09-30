package apisix

import (
	"context"
	"fmt"

	"isrvd/pkgs/apisix"
)

// SSLList 获取 SSL 证书列表
func (s *Service) SSLList(ctx context.Context) ([]apisix.SSL, error) {
	list, err := s.client.SSLList(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取证书列表失败: %w", err)
	}
	return list, nil
}

// SSLInspect 获取单个 SSL 证书详情
func (s *Service) SSLInspect(ctx context.Context, sslID string) (*apisix.SSL, error) {
	if sslID == "" {
		return nil, fmt.Errorf("SSL 证书 ID 不能为空")
	}
	ssl, err := s.client.SSLInspect(ctx, sslID)
	if err != nil {
		return nil, fmt.Errorf("获取证书详情失败: %w", err)
	}
	return ssl, nil
}

// SSLCreate 创建 SSL 证书
func (s *Service) SSLCreate(ctx context.Context, req apisix.SSL) (*apisix.SSL, error) {
	if len(req.Snis) == 0 {
		return nil, fmt.Errorf("SNI 不能为空")
	}
	if req.Cert == "" {
		return nil, fmt.Errorf("证书内容不能为空")
	}
	if req.Key == "" {
		return nil, fmt.Errorf("私钥内容不能为空")
	}
	ssl, err := s.client.SSLCreate(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("创建证书失败: %w", err)
	}
	return ssl, nil
}

// SSLUpdate 更新 SSL 证书
func (s *Service) SSLUpdate(ctx context.Context, sslID string, req apisix.SSL) (*apisix.SSL, error) {
	if sslID == "" {
		return nil, fmt.Errorf("SSL 证书 ID 不能为空")
	}
	if len(req.Snis) == 0 {
		return nil, fmt.Errorf("SNI 不能为空")
	}
	ssl, err := s.client.SSLUpdate(ctx, sslID, req)
	if err != nil {
		return nil, fmt.Errorf("更新证书失败: %w", err)
	}
	return ssl, nil
}

// SSLDelete 删除 SSL 证书
func (s *Service) SSLDelete(ctx context.Context, sslID string) error {
	if sslID == "" {
		return fmt.Errorf("SSL 证书 ID 不能为空")
	}
	if err := s.client.SSLDelete(ctx, sslID); err != nil {
		return fmt.Errorf("删除证书失败: %w", err)
	}
	return nil
}
