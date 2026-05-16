package caddy

import (
	"context"
	"encoding/json"
	"fmt"

	"isrvd/config"
)

// ─── 概览与原始配置 ───

// Info Caddy 概览信息
type Info struct {
	AdminURL  string `json:"adminUrl"`
	Servers   int    `json:"servers"`
	Routes    int    `json:"routes"`
	HasTLS    bool   `json:"hasTls"`
	Available bool   `json:"available"`
}

// Info 获取概览
func (s *Service) Info(ctx context.Context) (*Info, error) {
	cfg, err := s.client.ConfigAll(ctx)
	if err != nil {
		return &Info{AdminURL: config.Caddy.AdminURL, Available: false}, nil
	}
	info := &Info{AdminURL: config.Caddy.AdminURL, Available: true}
	if cfg.Apps != nil && cfg.Apps.HTTP != nil {
		for _, srv := range cfg.Apps.HTTP.Servers {
			info.Servers++
			info.Routes += len(srv.Routes)
		}
	}
	if cfg.Apps != nil && cfg.Apps.TLS != nil {
		info.HasTLS = true
	}
	return info, nil
}

// ConfigAll 获取完整配置（原始 JSON）
func (s *Service) ConfigAll(ctx context.Context) (json.RawMessage, error) {
	raw, err := s.client.ConfigRaw(ctx, "")
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return json.RawMessage("null"), nil
	}
	return json.RawMessage(raw), nil
}

// ConfigLoad 整体替换配置
func (s *Service) ConfigLoad(ctx context.Context, raw json.RawMessage) error {
	if len(raw) == 0 {
		return fmt.Errorf("配置内容不能为空")
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil || obj == nil {
		return fmt.Errorf("config 必须是 JSON 对象")
	}
	return s.client.ConfigLoadRaw(ctx, raw)
}
