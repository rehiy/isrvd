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
	"net/http"
	"net/url"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/rehiy/libgo/logman"

	"isrvd/server/config"
)

const (
	maxWebhookSends = 128
	webhookMaxAge   = 30 * time.Second
)

var (
	sendsMu               sync.Mutex
	sendsWG               = new(sync.WaitGroup)
	sendsClosed           bool
	sendsActive           int
	sendQueues            = map[[2]string]chan webhookDelivery{}
	sendsCtx, sendsCancel = context.WithCancel(context.Background())
)

type webhookDelivery struct {
	hook     *config.WebhookConfig
	event    Event
	deadline time.Time
}

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

// Shutdown 停止接收新通知，并在退出宽限期内等待已经发出的请求。
// 仅用于进程退出；配置重载不关闭通知发送。
func Shutdown(ctx context.Context) bool {
	sendsMu.Lock()
	sendsClosed = true
	wg := sendsWG
	sendsMu.Unlock()
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		sendsCancel()
		return true
	case <-ctx.Done():
		sendsCancel()
		return false
	}
}

func sendTo(hooks []*config.WebhookConfig, key string, evt *Event) {
	if evt == nil || len(hooks) == 0 {
		return
	}
	event := *evt
	if event.Timestamp == 0 {
		event.Timestamp = time.Now().Unix()
	}
	if event.Source == "" {
		event.Source = "isrvd"
	}
	sendsMu.Lock()
	defer sendsMu.Unlock()
	if sendsClosed {
		return
	}
	for _, hook := range hooks {
		if hook == nil {
			continue
		}
		queue := [2]string{key, strings.TrimSpace(hook.URL)}
		delivery := webhookDelivery{hook: hook, event: event, deadline: time.Now().Add(webhookMaxAge)}
		if pending := sendQueues[queue]; pending != nil {
			select {
			case <-pending:
				logman.Warn("webhook pending state replaced", "webhook", hook.Name, "key", key)
			default:
			}
			pending <- delivery
			continue
		}
		if sendsActive >= maxWebhookSends {
			logman.Warn("webhook send capacity reached", "webhook", hook.Name, "key", key)
			continue
		}
		var pending chan webhookDelivery
		if key != "" {
			pending = make(chan webhookDelivery, 1)
			sendQueues[queue] = pending
		}
		sendsActive++
		sendsWG.Go(func() { sendQueued(queue, pending, delivery) })
	}
}

func sendQueued(queue [2]string, pending chan webhookDelivery, delivery webhookDelivery) {
	for {
		ctx, cancel := context.WithDeadline(sendsCtx, delivery.deadline)
		if ctx.Err() == nil {
			sendOne(ctx, delivery.hook, &delivery.event)
		}
		cancel()
		sendsMu.Lock()
		if sendsCtx.Err() == nil {
			select {
			case delivery = <-pending:
				sendsMu.Unlock()
				continue
			default:
			}
		}
		delete(sendQueues, queue)
		sendsActive--
		sendsMu.Unlock()
		return
	}
}

// 单个请求的超时时间，避免接收方不可达时长时间占用
var webhookClient = &http.Client{
	Timeout:       10 * time.Second,
	CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
}

// sendOne 向单个 Webhook 发送；失败仅记录日志，不影响其他通道。
// Webhook 地址由管理员在后台配置，属于可信输入，不做内网地址等 SSRF 校验。
func sendOne(ctx context.Context, hook *config.WebhookConfig, evt *Event) {
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target.String(), bytes.NewReader(body))
	if err != nil {
		logman.Warn("发送告警失败", "webhook", hook.Name, "error", "invalid request")
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := webhookClient.Do(req)
	if err != nil {
		logman.Warn("发送告警失败", "webhook", hook.Name, "error", err.(*url.Error).Err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		logman.Warn("发送告警失败", "webhook", hook.Name, "status", resp.StatusCode)
	}
}

// ─── 辅助函数 ───

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

// snapshotWebhooks 过滤出可用的 Webhook 通道。
// config.Current() 返回的快照本身不可变（每次更新都会整体替换为新对象），
// 因此这里无需再对 WebhookConfig 做防御性拷贝。
func snapshotWebhooks(cfg *config.NotifyConfig) []*config.WebhookConfig {
	if cfg == nil {
		return nil
	}
	hooks := make([]*config.WebhookConfig, 0, len(cfg.Webhooks))
	for _, hook := range cfg.Webhooks {
		if hook != nil && strings.TrimSpace(hook.URL) != "" {
			hooks = append(hooks, hook)
		}
	}
	return hooks
}
