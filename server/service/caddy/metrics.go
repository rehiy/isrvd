package caddy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"isrvd/pkgs/caddy"
)

// MetricsStatus 返回全部 HTTP server 是否都已启用指标。
func (s *Service) MetricsStatus(ctx context.Context) (bool, error) {
	cfg, err := s.client.ConfigAll(ctx)
	if err != nil {
		return false, err
	}
	if cfg.Apps == nil || cfg.Apps.HTTP == nil || len(cfg.Apps.HTTP.Servers) == 0 {
		return false, nil
	}
	for _, srv := range cfg.Apps.HTTP.Servers {
		if srv != nil && srv.Metrics == nil {
			return false, nil
		}
	}
	return true, nil
}

// MetricsUpdate 为全部 HTTP server 开启或关闭指标。
// 开启时优先启用 per_host（指标带 host 标签，可按站点统计）；
// 当前 Caddy 版本不识别该选项而拒绝配置时，回退为不带 host 标签的基础指标。
func (s *Service) MetricsUpdate(ctx context.Context, enabled bool) error {
	err := s.metricsMutate(ctx, enabled, enabled)
	var upstream *caddy.HTTPError
	if enabled && errors.As(err, &upstream) && upstream.Path == "/load" &&
		upstream.StatusCode == http.StatusBadRequest {
		message := upstream.Body
		var response struct {
			Error string `json:"error"`
		}
		if json.Unmarshal([]byte(upstream.Body), &response) == nil {
			message = response.Error
		}
		if strings.Contains(message, `unknown field "per_host"`) {
			return s.metricsMutate(ctx, true, false)
		}
	}
	return err
}

// ─── 辅助函数 ───

func (s *Service) metricsMutate(ctx context.Context, enabled, perHost bool) error {
	return s.client.ConfigMutate(ctx, func(cfg *caddy.Config) error {
		if cfg.Apps == nil || cfg.Apps.HTTP == nil || len(cfg.Apps.HTTP.Servers) == 0 {
			return fmt.Errorf("%w：尚未配置任何 HTTP 服务", ErrServerNotFound)
		}
		for _, srv := range cfg.Apps.HTTP.Servers {
			if srv == nil {
				continue
			}
			if !enabled {
				srv.Metrics = nil
				delete(srv.Extras, "metrics")
				continue
			}
			if srv.Metrics != nil {
				continue // 已启用，保留用户已有的指标选项
			}
			srv.Metrics = map[string]any{}
			if perHost {
				srv.Metrics["per_host"] = true
				continue
			}
			// 空对象会被 omitempty 丢弃，需要通过 Extras 保留 "metrics": {}
			if srv.Extras == nil {
				srv.Extras = map[string]json.RawMessage{}
			}
			srv.Extras["metrics"] = json.RawMessage("{}")
		}
		return nil
	})
}
