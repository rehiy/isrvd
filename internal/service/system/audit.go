// Package system 提供系统级业务服务。
package system

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/libgo/logman"

	"isrvd/config"
	"isrvd/pkgs/jsonl"
)

const (
	maxAuditBufferSize = 100 // 内存缓冲最大条数
	// auditFileSuffix 审计日志文件后缀
	auditFileSuffix = ".jsonl"
)

// sensitiveFields 需要脱敏的请求体字段名
var sensitiveFields = []string{
	// 系统配置
	"jwtSecret", "apiKey", "adminKey", "clientSecret", "client_secret",
	// 账户模块
	"password", "oldPassword", "newPassword", "token", "accessToken", "refreshToken", "idToken",
	// APISIX 插件 + SSL 证书私钥
	"key", "secret", "public_key", "key_id", "secret_key",
}

// AuditLog 操作审计日志条目。
type AuditLog struct {
	Timestamp  time.Time `json:"timestamp"`  // 操作时间
	Username   string    `json:"username"`   // 操作人
	Method     string    `json:"method"`     // HTTP 方法（WebSocket 记为 "WS"）
	URI        string    `json:"uri"`        // 请求 URI
	Body       string    `json:"body"`       // 请求体（文件字段替换为占位符）
	IP         string    `json:"ip"`         // 客户端 IP
	StatusCode int       `json:"statusCode"` // 响应状态码
	Success    bool      `json:"success"`    // 是否成功
	Duration   int64     `json:"duration"`   // 耗时（毫秒）
}

// AuditService 审计日志业务服务
type AuditService struct {
	mu     sync.RWMutex
	buffer []AuditLog
	store  *jsonl.Store
}

// auditNaming 审计日志文件命名规则：YYYY-MM-DD.jsonl
func auditNaming() jsonl.Naming {
	return jsonl.Naming{Prefix: "", Sep: "", Suffix: auditFileSuffix}
}

// auditDir 审计日志根目录
func auditDir() string {
	return filepath.Join(config.Server.RootDirectory, "audit")
}

// NewAuditService 创建审计日志业务服务并自动初始化
func NewAuditService() *AuditService {
	dir := auditDir()
	store, err := jsonl.New(
		dir,
		auditNaming(),
		jsonl.WithBufferSize(4096),
		jsonl.WithFlushInterval(time.Second),
		jsonl.WithAsync(maxAuditBufferSize),
		jsonl.WithErrorHandler(func(err error) {
			logman.Warn("audit log background write failed", "error", err)
		}),
	)
	if err != nil {
		logman.Warn("audit log store init failed", "dir", dir, "error", err)
	}

	s := &AuditService{
		buffer: make([]AuditLog, 0, maxAuditBufferSize),
		store:  store,
	}

	// 启动时加载今日文件最近的 maxAuditBufferSize 条到内存
	s.loadRecent()
	return s
}

// LogAdd 将审计条目写入内存缓冲，并异步追加到当日日志文件。
func (s *AuditService) LogAdd(entry AuditLog) {
	s.mu.Lock()
	if len(s.buffer) >= maxAuditBufferSize {
		// 重新分配以释放底层数组，避免内存泄漏
		newBuf := make([]AuditLog, maxAuditBufferSize-1, maxAuditBufferSize)
		copy(newBuf, s.buffer[1:])
		s.buffer = newBuf
	}
	s.buffer = append(s.buffer, entry)
	s.mu.Unlock()

	if s.store == nil {
		return
	}
	if err := s.store.Append(&entry); err != nil {
		logman.Warn("audit log write failed", "error", err)
	}
}

// LogList 返回内存缓冲中的审计日志，按时间倒序排列。
// username 非空时仅返回该用户的记录；limit <= 0 时返回全部。
func (s *AuditService) LogList(username string, limit int) []AuditLog {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []AuditLog
	for i := len(s.buffer) - 1; i >= 0; i-- {
		entry := s.buffer[i]
		if username != "" && entry.Username != username {
			continue
		}
		result = append(result, entry)
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result
}

// AuditRecord 根据请求类型记录审计日志，供中间件在 c.Next() 后调用。
// WebSocket 升级请求记录 "WS" 方法；其余记录方法、URI、请求体、状态码。
func (s *AuditService) AuditRecord(c *gin.Context, startTime time.Time, body string) {
	// WebSocket
	if strings.EqualFold(c.GetHeader("Upgrade"), "websocket") {
		statusCode := c.Writer.Status()
		if statusCode == 0 || statusCode == http.StatusOK {
			statusCode = http.StatusSwitchingProtocols
		}
		s.LogAdd(AuditLog{
			Timestamp:  startTime,
			Username:   c.GetString("username"),
			Method:     "WS",
			URI:        c.Request.RequestURI,
			IP:         c.ClientIP(),
			StatusCode: statusCode,
			Success:    statusCode == http.StatusSwitchingProtocols,
			Duration:   time.Since(startTime).Milliseconds(),
		})
		return
	}

	statusCode := c.Writer.Status()
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	s.LogAdd(AuditLog{
		Timestamp:  startTime,
		Username:   c.GetString("username"),
		Method:     c.Request.Method,
		URI:        c.Request.RequestURI,
		Body:       body,
		IP:         c.ClientIP(),
		StatusCode: statusCode,
		Success:    statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices,
		Duration:   time.Since(startTime).Milliseconds(),
	})
}

// BodyRead 读取请求体，按 Content-Type 差异化处理：
//   - application/octet-stream：返回占位符
//   - multipart/form-data：保留文本字段，文件字段替换为占位符，敏感字段脱敏
//   - 其他：读取全部内容并回填 Body，敏感字段脱敏
func (s *AuditService) BodyRead(c *gin.Context) string {
	switch {
	case strings.HasPrefix(c.ContentType(), "application/octet-stream"):
		return "[Binary Omitted]"

	case strings.HasPrefix(c.ContentType(), "multipart/form-data"):
		form, err := c.MultipartForm()
		if err != nil || form == nil {
			return ""
		}
		fields := make(map[string]any)
		for k, vs := range form.Value {
			if len(vs) == 1 {
				fields[k] = maskSensitiveValue(k, vs[0])
			} else {
				masked := make([]string, len(vs))
				for i, v := range vs {
					masked[i] = maskSensitiveValue(k, v)
				}
				fields[k] = masked
			}
		}
		for k := range form.File {
			fields[k] = "[File Omitted]"
		}
		data, _ := json.Marshal(fields)
		return string(data)

	default:
		// 读取并回填 body，确保后续 handler 可正常读取
		raw, _ := io.ReadAll(c.Request.Body)
		c.Request.Body = io.NopCloser(bytes.NewReader(raw))
		return maskSensitiveJSON(string(raw))
	}
}

// Close 关闭底层文件句柄，刷盘缓冲数据
func (s *AuditService) Close() error {
	if s.store == nil {
		return nil
	}
	return s.store.Close()
}

// ---- 内部方法 ----------------------------------------------------------------

// loadRecent 启动时从今日文件尾部读取最近 maxAuditBufferSize 条到内存。
// 使用 TailLines 反向读取，避免加载整个日志文件。
func (s *AuditService) loadRecent() {
	if s.store == nil {
		return
	}
	path := s.store.FilePath(s.store.Today())
	entries, err := jsonl.DecodeTail[AuditLog](path, maxAuditBufferSize, nil)
	if err != nil {
		logman.Warn("audit log load recent failed", "path", path, "error", err)
		return
	}
	// DecodeTail 返回顺序为"由新到旧"，buffer 期望"由旧到新"
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := len(entries) - 1; i >= 0; i-- {
		s.buffer = append(s.buffer, entries[i])
	}
}

// maskSensitiveValue 对敏感字段的值进行脱敏
// - 长度 > 10：保留前 5 位 + ****** + 后 3 位
// - 长度 <= 10：保留首尾各 1 位，中间替换为 ******
func maskSensitiveValue(key, value string) string {
	for _, field := range sensitiveFields {
		if strings.EqualFold(key, field) {
			n := len(value)
			if n > 10 {
				return value[:5] + "******" + value[n-3:]
			} else if n > 2 {
				return value[:1] + "******" + value[n-1:]
			}
			return "******"
		}
	}
	return value
}

// maskSensitiveJSON 对 JSON 字符串中的敏感字段进行脱敏
func maskSensitiveJSON(jsonStr string) string {
	var data map[string]any
	if json.Unmarshal([]byte(jsonStr), &data) != nil {
		return jsonStr
	}
	maskMap(data)
	result, _ := json.Marshal(data)
	return string(result)
}

// maskMap 递归脱敏 map 中的敏感字段
func maskMap(m map[string]any) {
	for key, val := range m {
		switch v := val.(type) {
		case string:
			m[key] = maskSensitiveValue(key, v)
		case map[string]any:
			maskMap(v)
		case []any:
			for i, item := range v {
				if itemMap, ok := item.(map[string]any); ok {
					maskMap(itemMap)
					v[i] = itemMap
				}
			}
		}
	}
}
