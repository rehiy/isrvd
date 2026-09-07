// Package notify 提供告警通知能力（通用 Webhook 发送）。
//
// 设计：后端只负责发出结构统一的告警事件，各消息渠道（钉钉、飞书、企业微信等）
// 的格式差异由 WebhookConfig.Template 承载，模板内容由前端预置填充。
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"text/template"
	"time"

	"github.com/rehiy/libgo/logman"

	"isrvd/config"
)

// 单个请求的超时时间，避免接收方不可达时长时间占用
const sendTimeout = 10 * time.Second

var errWebhookTargetBlocked = errors.New("webhook 目标地址被安全策略拒绝")

// Event 告警事件，同时作为通用 JSON 载荷与模板渲染的数据源
type Event struct {
	Source    string         `json:"source"`    // 来源标识，固定为 isrvd
	Event     string         `json:"event"`     // 事件类型，如 resource.alert / resource.recover
	Level     string         `json:"level"`     // 级别：warning / critical / info
	Title     string         `json:"title"`     // 标题
	Message   string         `json:"message"`   // 详细描述
	Timestamp int64          `json:"timestamp"` // 触发时间（Unix 秒）
	Data      map[string]any `json:"data"`      // 事件相关数据
}

// Send 向所有已配置的 Webhook 发送告警事件；无可用通道时直接返回
func Send(evt *Event) {
	if config.Notify == nil || len(config.Notify.Webhooks) == 0 || evt == nil {
		return
	}
	if evt.Timestamp == 0 {
		evt.Timestamp = time.Now().Unix()
	}
	if evt.Source == "" {
		evt.Source = "isrvd"
	}

	for _, hook := range config.Notify.Webhooks {
		if hook == nil || strings.TrimSpace(hook.URL) == "" {
			continue
		}
		go sendOne(hook, evt)
	}
}

// sendOne 向单个 Webhook 发送；失败仅记录日志，不影响其他通道。
// 客户端禁用代理和重定向，并在实际建立连接前解析、校验并直连允许的 IP，防御 DNS 重绑定 SSRF。
func sendOne(hook *config.WebhookConfig, evt *Event) {
	target, err := validateWebhookURL(hook.URL)
	if err != nil {
		logman.Warn("告警 Webhook 地址被拒绝", "webhook", hook.Name, "error", err)
		return
	}

	body, err := renderBody(hook, evt)
	if err != nil {
		logman.Warn("渲染告警模板失败", "webhook", hook.Name, "error", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target.String(), bytes.NewReader(body))
	if err != nil {
		logman.Warn("创建告警请求失败", "webhook", hook.Name, "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := newWebhookClient().Do(req)
	if err != nil {
		logman.Warn("发送告警失败", "webhook", hook.Name, "error", err)
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= http.StatusMultipleChoices {
		logman.Warn("告警接收方返回异常状态", "webhook", hook.Name, "status", resp.StatusCode)
	}
}

func newWebhookClient() *http.Client {
	return &http.Client{
		Timeout: sendTimeout,
		Transport: &http.Transport{
			Proxy:                 nil,
			DialContext:           secureWebhookDialContext,
			TLSHandshakeTimeout:   sendTimeout,
			ResponseHeaderTimeout: sendTimeout,
			IdleConnTimeout:       30 * time.Second,
		},
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return errors.New("webhook 不允许重定向")
		},
	}
}

// ValidateWebhookURL 校验 Webhook 的静态地址约束；连接时还会重新校验 DNS 解析出的实际地址。
func ValidateWebhookURL(rawURL string) error {
	_, err := validateWebhookURL(rawURL)
	return err
}

func validateWebhookURL(rawURL string) (*url.URL, error) {
	target, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, fmt.Errorf("解析地址失败: %w", err)
	}
	if target.Scheme != "http" && target.Scheme != "https" {
		return nil, fmt.Errorf("仅允许 HTTP 或 HTTPS 地址")
	}
	if target.Hostname() == "" || target.User != nil {
		return nil, fmt.Errorf("地址必须包含主机且不能含用户信息")
	}
	if ip := net.ParseIP(target.Hostname()); ip != nil && isBlockedWebhookIP(ip) {
		return nil, errWebhookTargetBlocked
	}
	return target, nil
}

func secureWebhookDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("解析 Webhook 连接地址失败: %w", err)
	}

	ips, err := lookupAllowedWebhookIPs(ctx, host)
	if err != nil {
		return nil, err
	}

	dialer := &net.Dialer{Timeout: sendTimeout}
	var lastErr error
	for _, ip := range ips {
		conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("连接 Webhook 失败: %w", lastErr)
}

func lookupAllowedWebhookIPs(ctx context.Context, host string) ([]net.IP, error) {
	if ip := net.ParseIP(host); ip != nil {
		if isBlockedWebhookIP(ip) {
			return nil, errWebhookTargetBlocked
		}
		return []net.IP{ip}, nil
	}

	ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, fmt.Errorf("解析 Webhook 主机失败: %w", err)
	}
	allowed := make([]net.IP, 0, len(ips))
	for _, ip := range ips {
		address := net.IP(ip.AsSlice())
		if !isBlockedWebhookIP(address) {
			allowed = append(allowed, address)
		}
	}
	if len(allowed) == 0 {
		return nil, errWebhookTargetBlocked
	}
	return allowed, nil
}

// isBlockedWebhookIP 拒绝本机、私网、链路本地、组播及保留网段；9/8、11/8、21/8、30/8 是部署环境保留的内部网段。
func isBlockedWebhookIP(ip net.IP) bool {
	if ip == nil || ip.IsUnspecified() || ip.IsLoopback() || ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() {
		return true
	}
	if v4 := ip.To4(); v4 != nil {
		switch v4[0] {
		case 0, 9, 10, 11, 21, 30, 127:
			return true
		case 100:
			return v4[1]&0xc0 == 0x40 // 100.64.0.0/10（CGNAT）
		case 198:
			return v4[1] == 18 || v4[1] == 19 // 198.18.0.0/15（基准测试保留）
		}
	}
	return false
}

// renderBody 按模板渲染请求体；模板为空时使用标准 JSON
func renderBody(hook *config.WebhookConfig, evt *Event) ([]byte, error) {
	if strings.TrimSpace(hook.Template) == "" {
		return json.Marshal(evt)
	}

	tpl, err := template.New("webhook").Parse(hook.Template)
	if err != nil {
		return nil, fmt.Errorf("解析模板失败: %w", err)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, evt); err != nil {
		return nil, fmt.Errorf("执行模板失败: %w", err)
	}
	return buf.Bytes(), nil
}
