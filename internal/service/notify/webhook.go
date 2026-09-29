// Package notify 提供告警通知能力（通用 Webhook 发送）。
//
// 设计：后端只负责发出结构统一的告警事件，各消息渠道（钉钉、飞书、企业微信等）
// 的格式差异由 WebhookConfig.Template 承载，模板内容由前端预置填充。
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/rehiy/libgo/logman"
	"github.com/rehiy/libgo/request"

	"isrvd/config"
)

// 单个请求的超时时间，避免接收方不可达时长时间占用
const sendTimeout = 10 * time.Second

var (
	sendsMu     sync.Mutex
	sendsWG     sync.WaitGroup
	sendsClosed bool
)

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

// snapshotWebhooks 为异步通知保留通道快照，避免重载后引用可变配置。
func snapshotWebhooks(cfg *config.NotifyConfig) []*config.WebhookConfig {
	if cfg == nil {
		return nil
	}
	hooks := make([]*config.WebhookConfig, 0, len(cfg.Webhooks))
	for _, hook := range cfg.Webhooks {
		if hook != nil && strings.TrimSpace(hook.URL) != "" {
			copy := *hook
			hooks = append(hooks, &copy)
		}
	}
	return hooks
}

func sendTo(hooks []*config.WebhookConfig, evt *Event) {
	if evt == nil || len(hooks) == 0 {
		return
	}
	sendsMu.Lock()
	if sendsClosed {
		sendsMu.Unlock()
		return
	}
	sendsWG.Add(len(hooks))
	sendsMu.Unlock()
	copy := *evt
	if copy.Timestamp == 0 {
		copy.Timestamp = time.Now().Unix()
	}
	if copy.Source == "" {
		copy.Source = "isrvd"
	}
	for _, hook := range hooks {
		go func() {
			defer sendsWG.Done()
			sendOne(hook, &copy)
		}()
	}
}

// Shutdown 停止接收新通知，并在退出宽限期内等待已经发出的请求。
// 仅用于进程退出；配置重载不关闭通知发送。
func Shutdown(ctx context.Context) bool {
	sendsMu.Lock()
	sendsClosed = true
	sendsMu.Unlock()
	done := make(chan struct{})
	go func() {
		sendsWG.Wait()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-ctx.Done():
		return false
	}
}

// sendOne 向单个 Webhook 发送；失败仅记录日志，不影响其他通道。
// Webhook 地址由管理员在后台配置，属于可信输入，不做内网地址等 SSRF 校验。
func sendOne(hook *config.WebhookConfig, evt *Event) {
	target, err := validateWebhookURL(hook.URL)
	if err != nil {
		logman.Warn("告警 Webhook 地址无效", "webhook", hook.Name, "error", err)
		return
	}

	body, err := renderBody(hook, evt)
	if err != nil {
		logman.Warn("渲染告警模板失败", "webhook", hook.Name, "error", err)
		return
	}

	client := request.Client{
		Method:  "POST",
		Url:     target.String(),
		Data:    string(body),
		Headers: request.Header{"Content-Type": "application/json"},
		Timeout: sendTimeout,
	}
	if _, err := client.Request(); err != nil {
		logman.Warn("发送告警失败", "webhook", hook.Name, "error", err)
	}
}

// validateWebhookURL 仅做基本格式校验，不限制目标网段（Webhook 地址由管理员配置，可信）。
func validateWebhookURL(rawURL string) (*url.URL, error) {
	target, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, fmt.Errorf("解析地址失败: %w", err)
	}
	if target.Scheme != "http" && target.Scheme != "https" {
		return nil, fmt.Errorf("仅允许 HTTP 或 HTTPS 地址")
	}
	if target.Hostname() == "" {
		return nil, fmt.Errorf("地址必须包含主机")
	}
	return target, nil
}

// renderBody 按模板渲染请求体；模板为空时使用标准 JSON
func renderBody(hook *config.WebhookConfig, evt *Event) ([]byte, error) {
	if strings.TrimSpace(hook.Template) == "" {
		return json.Marshal(evt)
	}

	tpl, err := template.New("webhook").Funcs(template.FuncMap{
		"json": templateJSON,
	}).Parse(hook.Template)
	if err != nil {
		return nil, fmt.Errorf("解析模板失败: %w", err)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, evt); err != nil {
		return nil, fmt.Errorf("执行模板失败: %w", err)
	}
	return buf.Bytes(), nil
}

func templateJSON(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("JSON 编码模板变量失败: %w", err)
	}
	return string(data), nil
}
