// Package metrics 提供本机与容器指标出口开关、APISIX / Caddy 网关指标透传。
package metrics

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"isrvd/server/config"
	"isrvd/server/i18n"
)

// 指标来源标识
const (
	SourceHost       = "host"
	SourceContainers = "containers"
	GatewayCaddy     = "caddy"
	GatewayApisix    = "apisix"
)

var (
	ErrGatewayUnknown     = errors.New("未知的网关")
	ErrGatewayUnavailable = errors.New("网关服务不可用")
)

// Switch 网关指标开关，由 service/caddy、service/apisix 实现。
type Switch interface {
	MetricsStatus(ctx context.Context) (bool, error)
	MetricsUpdate(ctx context.Context, enabled bool) error
}

type source struct {
	name string
	url  string
	sw   Switch // 对应网关服务不可用时为 nil
}

// Service 指标开关与网关透传服务
type Service struct {
	sources             []*source
	client              *http.Client
	containersAvailable bool
}

const fetchTimeout = 5 * time.Second

// NewService 创建指标服务。switches 以网关标识为键，网关服务不可用时不要放入。
// 只为已配置 Admin URL 的网关创建抓取源。
func NewService(switches map[string]Switch, containersAvailable bool) *Service {
	cfg := config.Current()
	s := &Service{
		containersAvailable: containersAvailable,
		client: &http.Client{
			Timeout: fetchTimeout,
			// 抓取地址只来自管理员配置，不跟随重定向，避免被引导到其他内部地址
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
	if cfg.Apisix != nil && cfg.Apisix.AdminURL != "" {
		if u := apisixMetricsURL(cfg.Apisix); u != "" {
			s.sources = append(s.sources, &source{name: GatewayApisix, url: u, sw: switches[GatewayApisix]})
		}
	}
	if cfg.Caddy != nil && cfg.Caddy.AdminURL != "" {
		u := strings.TrimRight(cfg.Caddy.AdminURL, "/") + "/metrics"
		s.sources = append(s.sources, &source{name: GatewayCaddy, url: u, sw: switches[GatewayCaddy]})
	}
	return s
}

// Scrape 抓取指定网关的指标原文，原样返回。
func (s *Service) Scrape(ctx context.Context, gateway string) ([]byte, error) {
	src := s.find(gateway)
	if src == nil {
		return nil, ErrGatewayUnknown
	}
	return s.fetch(ctx, src.url)
}

// SourceStatus 单个来源的指标状态
type SourceStatus struct {
	Name           string `json:"name"`                     // host / containers / caddy / apisix
	Available      bool   `json:"available"`                // 本机采集能力或网关管理服务是否可用
	Enabled        bool   `json:"enabled"`                  // 是否已开启指标
	EnabledUnknown bool   `json:"enabledUnknown,omitempty"` // 网关开关查询失败，enabled 不代表关闭
	Scrapable      bool   `json:"scrapable"`                // 本机出口就绪状态或网关抓取检查结果
	URL            string `json:"url"`                      // 抓取地址
	Error          string `json:"error,omitempty"`          // 最近一次检查失败的原因
}

// Status 返回本机、容器及已配置网关的指标状态。
func (s *Service) Status(ctx context.Context) []SourceStatus {
	out := make([]SourceStatus, 0, len(s.sources)+2)
	for _, name := range []string{SourceHost, SourceContainers} {
		available := name == SourceHost || s.containersAvailable
		enabled := LocalEnabled(name)
		out = append(out, SourceStatus{
			Name: name, Available: available, Enabled: enabled,
			Scrapable: available && enabled, URL: "/api/metrics/" + name,
		})
	}
	for _, src := range s.sources {
		st := SourceStatus{Name: src.name, Available: src.sw != nil, URL: "/api/metrics/" + src.name}
		if src.sw != nil {
			enabled, err := src.sw.MetricsStatus(ctx)
			st.Enabled = enabled
			if err != nil {
				st.EnabledUnknown = true
				st.Error = i18n.TC(ctx, "查询网关指标开关失败")
			}
		}
		if _, err := s.fetch(ctx, src.url); err == nil {
			st.Scrapable = true
		} else if st.Error == "" {
			st.Error = i18n.TC(ctx, err.Error())
		}
		out = append(out, st)
	}
	return out
}

// LocalEnabled 返回本机或容器指标出口开关，未知来源不放行。
func LocalEnabled(name string) bool {
	cfg := config.Current().Metrics
	switch name {
	case SourceHost:
		return cfg.HostEnabled
	case SourceContainers:
		return cfg.ContainersEnabled
	default:
		return false
	}
}

// Update 开启或关闭指定来源的指标。
func (s *Service) Update(ctx context.Context, name string, enabled bool) error {
	if name == SourceHost || name == SourceContainers {
		return config.Update(func(draft *config.Snapshot) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			if name == SourceHost {
				draft.Metrics.HostEnabled = enabled
			} else {
				draft.Metrics.ContainersEnabled = enabled
			}
			return nil
		})
	}
	src := s.find(name)
	if src == nil {
		return ErrGatewayUnknown
	}
	if src.sw == nil {
		return ErrGatewayUnavailable
	}
	return src.sw.MetricsUpdate(ctx, enabled)
}

// ─── 内部方法 ───

func (s *Service) find(name string) *source {
	for _, src := range s.sources {
		if src.name == name {
			return src
		}
	}
	return nil
}

const maxBodyBytes = 16 << 20 // 单次抓取响应上限

// fetch 抓取指标端点原文
func (s *Service) fetch(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, errors.New("创建指标请求失败")
	}
	// 指定经典文本格式，避免对端按 OpenMetrics 返回
	req.Header.Set("Accept", "text/plain;version=0.0.4")
	resp, err := s.client.Do(req)
	if err != nil {
		var requestErr *url.Error
		if errors.As(err, &requestErr) {
			err = requestErr.Err // 外层 URL 可能含查询令牌，不返回给调用者
		}
		return nil, fmt.Errorf("请求指标端点失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("指标端点返回状态码 %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes+1))
	if err != nil {
		return nil, fmt.Errorf("读取指标响应失败: %w", err)
	}
	if len(data) > maxBodyBytes {
		return nil, errors.New("指标响应过大")
	}
	return data, nil
}

// ─── 辅助函数 ───

// apisixMetricsURL 优先使用显式配置；否则取 Admin URL 的主机加 APISIX 默认导出端口 9091。
func apisixMetricsURL(cfg *config.ApisixConfig) string {
	if cfg.MetricsURL != "" {
		return cfg.MetricsURL
	}
	u, err := url.Parse(cfg.AdminURL)
	if err != nil || u.Hostname() == "" {
		return ""
	}
	return u.Scheme + "://" + net.JoinHostPort(u.Hostname(), "9091") + "/apisix/prometheus/metrics"
}
