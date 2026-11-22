package notify

import (
	"fmt"
	"time"

	"isrvd/server/config"
)

// NodeAlertsEnabled 仅在开关启用且存在至少一个格式有效的 Webhook 地址时开始观察。
// 与发送逻辑共用 URL 过滤和校验，避免空白或无效地址造成未发送却已去重。
func NodeAlertsEnabled() bool {
	cfg := config.Current().Notify
	if cfg == nil || cfg.Events == nil || !cfg.Events.NodeEnabled {
		return false
	}
	for _, hook := range snapshotWebhooks(cfg) {
		if _, err := validateWebhookURL(hook.URL); err == nil {
			return true
		}
	}
	return false
}

// NodeStatusNotify 复用已配置 Webhook 通道发送中控节点离线与恢复事件。
// 节点观察与去重在 node 服务中完成，通知层不依赖节点模块。
func NodeStatusNotify(id, name string, recovered bool, offline time.Duration) {
	cfg := config.Current().Notify
	if cfg == nil || cfg.Events == nil || !cfg.Events.NodeEnabled {
		return
	}
	event, level := "node.alert", "warning"
	title := fmt.Sprintf("节点 %s 已离线", name)
	message := fmt.Sprintf("受管节点 %s 已持续离线约 %d 秒", name, int64(offline.Seconds()))
	if recovered {
		event, level = "node.recover", "info"
		title = fmt.Sprintf("节点 %s 已恢复上线", name)
		message = fmt.Sprintf("受管节点 %s 已重新上线，离线约 %d 秒", name, int64(offline.Seconds()))
	}
	sendTo(snapshotWebhooks(cfg), "node:"+id, &Event{
		Event: event, Level: level, Title: title, Message: message,
		Data: map[string]any{"nodeId": id, "nodeName": name, "offlineSeconds": int64(offline.Seconds())},
	})
}

// JobFailureNotifier 为当前调度器捕获配置快照，每次失败执行仅发送一次通知。
// 不发送脚本正文、输出或错误详情，避免通过通知渠道泄露凭据。
func JobFailureNotifier(cfg *config.NotifyConfig) func(string, string, string, int64) {
	enabled := cfg != nil && cfg.Events != nil && cfg.Events.CronEnabled
	hooks := snapshotWebhooks(cfg)
	return func(jobID, jobName, runID string, duration int64) {
		if !enabled {
			return
		}
		sendTo(hooks, "", &Event{Event: "cron.failed", Level: "warning", Title: fmt.Sprintf("计划任务 %s执行失败", jobName),
			Message: "请在计划任务执行历史中查看失败原因", Data: map[string]any{
				"jobId": jobID, "jobName": jobName, "runId": runID, "duration": duration,
			}})
	}
}
