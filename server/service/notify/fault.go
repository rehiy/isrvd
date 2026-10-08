package notify

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/rehiy/libgo/certify"
	"github.com/rehiy/libgo/logman"

	"isrvd/pkgs/apisix"
	"isrvd/server/config"
)

const (
	faultPollInterval           = 30 * time.Second
	faultProbeTimeout           = 20 * time.Second
	containerInspectTimeout     = 5 * time.Second
	certificatePollInterval     = time.Hour
	certificateReminderInterval = 24 * time.Hour
)

// Caddy 证书来源类型。
//
// 注意：这三个取值是 server/service/caddy 的 CertSourceFile / CertSourcePEM / CertSourceAutomate
// 的副本，用来避免 notify 依赖 service/caddy。两侧没有编译期约束，修改任一侧的取值
// 都必须同步修改另一侧，否则证书到期告警会静默漏报（automate 会被当作普通证书、
// file/pem 的去重标识会变成空）。
const (
	caddyCertSourceFile     = "file"
	caddyCertSourcePEM      = "pem"
	caddyCertSourceAutomate = "automate"
)

// CaddyCert 证书到期检测所需的 Caddy 证书字段，由注入方从具体服务转换而来，
// 避免告警层依赖 server/service/caddy。
type CaddyCert struct {
	Key         string     // 列表复合主键，缓存证书用作稳定标识
	Source      string     // file / pem / automate / cached
	Subject     string     // 域名或证书 CN
	Certificate string     // file：证书文件路径；pem：证书 PEM 文本
	NotAfter    *time.Time // 证书过期时间，无法解析时为 nil
}

// FaultSources 由服务生命周期注入已初始化的客户端，不在告警层创建连接。
type FaultSources struct {
	DockerKey string `json:"-"` // 数据源地址，重载切换后不复用旧来源的去重状态
	CaddyKey  string `json:"-"`
	ApisixKey string `json:"-"`
	Docker    interface {
		ContainerList(context.Context, bool, ...string) ([]container.Summary, error)
		ContainerInspect(context.Context, string) (container.InspectResponse, error)
	} `json:"-"`
	Caddy interface {
		CertList(context.Context) ([]CaddyCert, error)
	} `json:"-"`
	Apisix interface {
		SSLList(context.Context) ([]apisix.SSL, error)
	} `json:"-"`
}

// FaultWatcher 独立于监控日志的应用故障检测器。
// 去重状态仅由当前检测协程访问，重载时由新实例继承。
type FaultWatcher struct {
	options      config.FaultAlertConfig
	sources      FaultSources
	containers   map[string]*containerFaultState
	certificates map[string]*certificateFaultState
	webhooks     []*config.WebhookConfig
	cancel       context.CancelFunc
	done         chan struct{}
}

// NewFaultWatcher 在前一实例已停止后继承其去重状态。
func NewFaultWatcher(cfg *config.NotifyConfig, sources FaultSources, previous *FaultWatcher) *FaultWatcher {
	var events *config.FaultAlertConfig
	if cfg != nil {
		events = cfg.Events
	}
	w := &FaultWatcher{
		options:  config.FaultAlertNormalize(events),
		sources:  sources,
		webhooks: snapshotWebhooks(cfg),
	}
	if previous == nil {
		w.containers = make(map[string]*containerFaultState)
		w.certificates = make(map[string]*certificateFaultState)
		return w
	}
	w.containers, w.certificates = previous.containers, previous.certificates
	if !w.options.ContainerEnabled || len(w.webhooks) == 0 || previous.sources.DockerKey != sources.DockerKey {
		clear(w.containers)
	}
	for key := range w.certificates {
		if !w.options.CertificateEnabled || len(w.webhooks) == 0 ||
			(strings.HasPrefix(key, "caddy:") && previous.sources.CaddyKey != sources.CaddyKey) ||
			(strings.HasPrefix(key, "apisix:") && previous.sources.ApisixKey != sources.ApisixKey) {
			delete(w.certificates, key)
		}
	}
	return w
}

// Start 启动检测；未开启的类型不调用对应外部服务。
// Start/Stop 由 App 的服务生命周期串行调用。
func (w *FaultWatcher) Start(ctx context.Context) {
	if w.cancel != nil || len(w.webhooks) == 0 || (!w.options.ContainerEnabled && !w.options.CertificateEnabled) {
		return
	}
	ctx, w.cancel = context.WithCancel(ctx)
	w.done = make(chan struct{})
	go func() {
		defer close(w.done)
		ticker := time.NewTicker(faultPollInterval)
		defer ticker.Stop()
		nextCertificates := time.Time{}
		for {
			if ctx.Err() != nil {
				return
			}
			now := time.Now()
			if w.options.ContainerEnabled && w.sources.Docker != nil {
				w.pollContainers(ctx, now)
			}
			if w.options.CertificateEnabled && !now.Before(nextCertificates) {
				w.pollCertificates(ctx, now)
				nextCertificates = now.Add(certificatePollInterval)
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

// Stop 取消检测并等待当前探测退出，重载时不留下重复检测协程。
func (w *FaultWatcher) Stop() {
	if w.cancel != nil {
		w.cancel()
		<-w.done
		w.cancel = nil
	}
}

func (w *FaultWatcher) pollContainers(parent context.Context, now time.Time) {
	ctx, cancel := context.WithTimeout(parent, faultProbeTimeout)
	list, err := w.sources.Docker.ContainerList(ctx, true)
	cancel()
	if err != nil {
		if parent.Err() == nil {
			logman.Warn("fault watcher: list containers failed", "error", err)
		}
		return // 查询失败不视为故障解除或容器消失
	}
	seen := make(map[string]bool, len(list))
	for _, item := range list {
		if parent.Err() != nil {
			return
		}
		if item.Labels["isrvd.notify.ignore"] == "true" {
			continue
		}
		seen[item.ID] = true // 详情查询失败仍保留旧告警状态。
		ctx, cancel := context.WithTimeout(parent, containerInspectTimeout)
		info, err := w.sources.Docker.ContainerInspect(ctx, item.ID)
		cancel()
		if err != nil || info.ContainerJSONBase == nil || info.State == nil {
			continue
		}
		if parent.Err() != nil {
			return
		}
		w.checkContainer(item.ID, strings.TrimPrefix(info.Name, "/"), info, now)
	}
	for id := range w.containers {
		if !seen[id] {
			delete(w.containers, id) // 删除/排除容器不发送恢复通知
		}
	}
}

type restartSample struct {
	at    time.Time
	count int
}

type containerFaultState struct {
	lastRestarts int
	restarts     []restartSample
	firing       bool
}

func (w *FaultWatcher) checkContainer(id, name string, info container.InspectResponse, now time.Time) {
	previous := w.containers[id]
	if previous == nil {
		previous = &containerFaultState{lastRestarts: info.RestartCount}
		w.containers[id] = previous
	}
	if info.RestartCount < previous.lastRestarts {
		previous.restarts = nil
	} else if delta := info.RestartCount - previous.lastRestarts; delta > 0 {
		previous.restarts = append(previous.restarts, restartSample{at: now, count: delta})
	}
	previous.lastRestarts = info.RestartCount
	windowStart := now.Add(-time.Duration(w.options.RestartWindow) * time.Second)
	count := 0
	kept := previous.restarts[:0]
	for _, sample := range previous.restarts {
		if sample.at.After(windowStart) {
			kept = append(kept, sample)
			count += sample.count
		}
	}
	previous.restarts = kept
	fault := ""
	state := info.State
	switch {
	case state.Restarting || state.Status == "restarting":
		fault = "restarting"
	case state.Status == "dead":
		fault = "dead"
	case !state.Running && state.OOMKilled:
		fault = "oom"
	case state.Status == "exited" && state.ExitCode != 0 && state.ExitCode != 137 && state.ExitCode != 143:
		fault = "exit"
	case state.Running && !state.Paused && state.Health != nil && state.Health.Status == "unhealthy":
		fault = "unhealthy"
	case state.Running && !state.Paused && count >= w.options.RestartThreshold:
		fault = "restarts"
	}
	if fault == "" && (!state.Running || state.Paused || (state.Health != nil && state.Health.Status != "healthy")) {
		return
	}
	if previous.firing == (fault != "") {
		return
	}
	previous.firing = fault != ""
	evt := &Event{Data: map[string]any{"containerId": id, "containerName": name, "reason": fault,
		"exitCode": state.ExitCode, "restartCount": count, "restartWindow": w.options.RestartWindow}}
	if previous.firing {
		label := map[string]string{"restarting": "正在反复重启", "dead": "无法运行", "oom": "因内存不足退出",
			"exit": "异常退出", "unhealthy": "健康检查失败", "restarts": "频繁重启"}[fault]
		evt.Event, evt.Level = "container.alert", "warning"
		evt.Title = fmt.Sprintf("容器 %s%s", name, label)
		evt.Message = fmt.Sprintf("容器 %s%s；退出码 %d，最近 %d 秒检测到 %d 次重启", name, label, state.ExitCode, w.options.RestartWindow, count)
	} else {
		evt.Event, evt.Level = "container.recover", "info"
		evt.Title = fmt.Sprintf("容器 %s已恢复", name)
		evt.Message = "容器已恢复运行，未检测到异常或频繁重启"
	}
	sendTo(w.webhooks, evt)
}

func (w *FaultWatcher) pollCertificates(parent context.Context, now time.Time) {
	if w.sources.Caddy != nil {
		ctx, cancel := context.WithTimeout(parent, faultProbeTimeout)
		list, err := w.sources.Caddy.CertList(ctx)
		cancel()
		if err == nil && parent.Err() == nil {
			certs := make([]certificateExpiry, 0, len(list))
			for _, cert := range list {
				if cert.Source == caddyCertSourceAutomate {
					continue // 自动签发策略本身没有证书有效期
				}
				key := caddyCertificateKey(cert)
				id := cert.Key
				if id == "" {
					id = key
				}
				expiry := certificateExpiry{id: id, key: key, subject: cert.Subject}
				if cert.NotAfter != nil {
					expiry.notAfter = *cert.NotAfter
				}
				certs = append(certs, expiry)
			}
			w.checkCertificates("caddy", certs, now)
		} else if err != nil && parent.Err() == nil {
			logman.Warn("fault watcher: list Caddy certificates failed", "error", err)
		}
	}
	if w.sources.Apisix != nil && parent.Err() == nil {
		ctx, cancel := context.WithTimeout(parent, faultProbeTimeout)
		list, err := w.sources.Apisix.SSLList(ctx)
		cancel()
		if err == nil && parent.Err() == nil {
			certs := make([]certificateExpiry, 0, len(list))
			for _, ssl := range list {
				if ssl.Status != nil && *ssl.Status == 0 {
					continue
				}
				if cert := certify.PEMParse([]byte(ssl.Cert)); cert != nil {
					certs = append(certs, certificateExpiry{id: ssl.ID, subject: strings.Join(ssl.Snis, ", "), notAfter: cert.NotAfter})
				} else {
					certs = append(certs, certificateExpiry{id: ssl.ID})
				}
			}
			w.checkCertificates("apisix", certs, now)
		} else if err != nil && parent.Err() == nil {
			logman.Warn("fault watcher: list APISIX certificates failed", "error", err)
		}
	}
}

type certificateExpiry struct {
	id       string
	key      string // Caddy 去重标识独立于供 API 使用的列表下标
	subject  string
	notAfter time.Time
}

type certificateFaultState struct {
	notAfter time.Time
	lastSent time.Time
	level    string
}

func (w *FaultWatcher) checkCertificates(provider string, certs []certificateExpiry, now time.Time) {
	// 同身份的多张证书按最早到期的一张判断，避免互相覆盖告警状态。
	seen := make(map[string]certificateExpiry, len(certs))
	unreadablePEM := false
	for _, cert := range certs {
		if provider == "caddy" && cert.key == "" && cert.notAfter.IsZero() {
			// 无法解析的 PEM 没有稳定身份，不能通过列表下标关联旧证书。
			unreadablePEM = true
			continue
		}
		identity := cert.key
		if identity == "" {
			identity = cert.id
		}
		key := provider + ":" + identity
		if previous, ok := seen[key]; !ok || (!cert.notAfter.IsZero() && (previous.notAfter.IsZero() ||
			cert.notAfter.Before(previous.notAfter) || (cert.notAfter.Equal(previous.notAfter) && cert.id < previous.id))) {
			seen[key] = cert
		}
	}
	for key, cert := range seen {
		if cert.notAfter.IsZero() {
			continue
		}
		old := w.certificates[key]
		days := int(math.Ceil(cert.notAfter.Sub(now).Hours() / 24))
		data := map[string]any{"provider": provider, "certificateId": cert.id, "subject": cert.subject,
			"notAfter": cert.notAfter.UTC().Format(time.RFC3339), "daysRemaining": days}
		if cert.notAfter.After(now.Add(time.Duration(w.options.CertificateDays) * 24 * time.Hour)) {
			// 身份不明的 PEM 可能是同身份的旧证书，暂缓恢复以免掩盖其告警。
			if old != nil && !(unreadablePEM && strings.HasPrefix(key, "caddy:pem:")) {
				delete(w.certificates, key)
				sendTo(w.webhooks, &Event{Event: "certificate.recover", Level: "info", Title: fmt.Sprintf("%s 证书 %s有效期已恢复", provider, cert.subject),
					Message: "证书有效期已超过提前告警阈值", Data: data})
			}
			continue
		}
		level, description := "warning", fmt.Sprintf("将在 %d 天内到期", days)
		if !cert.notAfter.After(now) {
			level, description = "critical", "已过期"
		}
		if old != nil && old.notAfter.Equal(cert.notAfter) && old.level == level && now.Sub(old.lastSent) < certificateReminderInterval {
			continue
		}
		w.certificates[key] = &certificateFaultState{notAfter: cert.notAfter, lastSent: now, level: level}
		sendTo(w.webhooks, &Event{Event: "certificate.alert", Level: level, Title: fmt.Sprintf("%s 证书 %s%s", provider, cert.subject, description),
			Message: fmt.Sprintf("证书%s，到期时间 %s", description, cert.notAfter.UTC().Format(time.RFC3339)), Data: data})
	}
	for key := range w.certificates {
		if unreadablePEM && strings.HasPrefix(key, "caddy:pem:") {
			continue // 保留无法确认是否仍存在的旧 PEM 状态。
		}
		if _, ok := seen[key]; strings.HasPrefix(key, provider+":") && !ok {
			delete(w.certificates, key)
		}
	}
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
		sendTo(hooks, &Event{Event: "cron.failed", Level: "warning", Title: fmt.Sprintf("计划任务 %s执行失败", jobName),
			Message: "请在计划任务执行历史中查看失败原因", Data: map[string]any{
				"jobId": jobID, "jobName": jobName, "runId": runID, "duration": duration,
			}})
	}
}

// ─── 辅助函数 ───

func caddyCertificateKey(cert CaddyCert) string {
	switch cert.Source {
	case caddyCertSourceFile:
		return "file:" + cert.Certificate
	case caddyCertSourcePEM:
		// 使用证书身份而非 PEM 内容，续期后仍可关联原告警。
		parsed := certify.PEMParse([]byte(cert.Certificate))
		if parsed == nil {
			return ""
		}
		names := make([]string, 0, len(parsed.DNSNames))
		for _, name := range parsed.DNSNames {
			names = append(names, "dns:"+name)
		}
		for _, ip := range parsed.IPAddresses {
			names = append(names, "ip:"+ip.String())
		}
		for _, email := range parsed.EmailAddresses {
			names = append(names, "email:"+email)
		}
		for _, uri := range parsed.URIs {
			names = append(names, "uri:"+uri.String())
		}
		sort.Strings(names)
		identity := parsed.Subject.String() + "\x00" + parsed.PublicKeyAlgorithm.String() + "\x00" + strings.Join(names, "\x00")
		return fmt.Sprintf("pem:%x", sha256.Sum256([]byte(identity)))
	default:
		return cert.Key // 缓存证书的 Key 包含文件路径，已是稳定标识
	}
}
