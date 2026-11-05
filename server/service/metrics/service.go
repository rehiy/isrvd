// Package metrics 是指标统一读出口：本机与容器出口开关、网关指标透传与状态汇总。
package metrics

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

// ErrGatewayUnknown 表示请求的网关未配置。
var ErrGatewayUnknown = errors.New("未知的网关")

// source 一个可被抓取的网关指标来源，由网关所在模块登记；本包不感知具体网关。
type source struct {
	name    string
	url     string
	enabled func(ctx context.Context) (bool, error) // 查询网关指标是否已开启；网关管理服务不可用时为 nil
}

// Service 指标统一读出口：本机与容器出口开关、网关指标透传与状态汇总。
// 网关指标的开关由各网关模块自行修改，本服务不写入。
type Service struct {
	sources             []source
	client              *http.Client
	containersAvailable bool
}

const fetchTimeout = 5 * time.Second

// NewService 创建指标服务；网关通过 AddGateway 登记。
func NewService(containersAvailable bool) *Service {
	return &Service{
		containersAvailable: containersAvailable,
		client: &http.Client{
			Timeout: fetchTimeout,
			// 抓取地址只来自管理员配置，不跟随重定向，避免被引导到其他内部地址
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
}

// AddGateway 登记一个已配置抓取地址的网关。enabled 为 nil 表示网关管理服务不可用，状态中仅报告抓取结果。
func (s *Service) AddGateway(name, url string, enabled func(ctx context.Context) (bool, error)) {
	s.sources = append(s.sources, source{name: name, url: url, enabled: enabled})
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
	Configured     bool   `json:"configured"`               // 是否已配置抓取地址；本机与容器始终为 true
	Available      bool   `json:"available"`                // 本机采集能力或网关管理服务是否可用
	Enabled        bool   `json:"enabled"`                  // 是否已开启指标
	EnabledUnknown bool   `json:"enabledUnknown,omitempty"` // 网关开关查询失败，enabled 不代表关闭
	Scrapable      bool   `json:"scrapable"`                // 本机出口就绪状态或网关抓取检查结果
	URL            string `json:"url"`                      // 抓取地址
	Error          string `json:"error,omitempty"`          // 最近一次检查失败的原因
}

// Status 固定按 host、containers、caddy、apisix 返回四类来源的指标状态；未配置的网关 Configured 为 false。
func (s *Service) Status(ctx context.Context) []SourceStatus {
	return []SourceStatus{
		s.localStatus(SourceHost, true),
		s.localStatus(SourceContainers, s.containersAvailable),
		s.gatewayStatus(ctx, GatewayCaddy),
		s.gatewayStatus(ctx, GatewayApisix),
	}
}

func (s *Service) localStatus(name string, available bool) SourceStatus {
	enabled := LocalEnabled(name)
	return SourceStatus{
		Name: name, Configured: true, Available: available, Enabled: enabled,
		Scrapable: available && enabled, URL: "/api/metrics/" + name,
	}
}

func (s *Service) gatewayStatus(ctx context.Context, name string) SourceStatus {
	st := SourceStatus{Name: name, URL: "/api/metrics/" + name}
	src := s.find(name)
	if src == nil {
		return st
	}
	st.Configured = true
	st.Available = src.enabled != nil
	if src.enabled != nil {
		enabled, err := src.enabled(ctx)
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
	return st
}

// LocalEnabled 返回本机或容器指标出口开关（系统配置 metrics 分区），未知来源不放行。
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

// ─── 内部方法 ───

func (s *Service) find(name string) *source {
	for i := range s.sources {
		if s.sources[i].name == name {
			return &s.sources[i]
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
