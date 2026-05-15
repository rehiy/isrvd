// Package system 提供系统级业务服务。
package system

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/libgo/logman"

	"isrvd/config"
)

const maxAuditBufferSize = 100 // 内存缓冲最大条数

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
	mu      sync.RWMutex
	buffer  []AuditLog
	ch      chan AuditLog
	file    *os.File
	dateKey string // 当前日志文件对应的日期（YYYY-MM-DD）
}

// NewAuditService 创建审计日志业务服务并自动初始化
func NewAuditService() *AuditService {
	s := &AuditService{
		buffer: make([]AuditLog, 0, maxAuditBufferSize),
		ch:     make(chan AuditLog, maxAuditBufferSize),
	}

	today := time.Now().Format("2006-01-02")
	s.loadFile(auditFilePath(today))
	s.openFile(today)

	go s.process()

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

	select {
	case s.ch <- entry:
	default:
		logman.Warn("audit log channel full, discard write", "max_size", maxAuditBufferSize)
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

// ---- 内部方法 ----------------------------------------------------------------

// process 从通道消费审计条目，按日期轮转文件并写入。
func (s *AuditService) process() {
	for entry := range s.ch {
		today := entry.Timestamp.Format("2006-01-02")
		if today != s.dateKey {
			s.openFile(today)
		}
		if s.file == nil {
			continue
		}
		data, err := json.Marshal(entry)
		if err != nil {
			continue
		}
		data = append(data, '\n')
		if _, err = s.file.Write(data); err != nil {
			logman.Warn("audit log write failed", "error", err)
		}
	}
}

// loadFile 从指定路径读取日志文件，将最后 maxAuditBufferSize 条记录追加到内存缓冲。
func (s *AuditService) loadFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	var buf []AuditLog
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var entry AuditLog
		if json.Unmarshal(scanner.Bytes(), &entry) == nil {
			buf = append(buf, entry)
		}
	}

	if len(buf) > maxAuditBufferSize {
		buf = buf[len(buf)-maxAuditBufferSize:]
	}
	s.buffer = append(s.buffer, buf...)
}

// openFile 打开指定日期的日志文件（追加模式），关闭旧文件。
func (s *AuditService) openFile(date string) {
	path := auditFilePath(date)

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		logman.Warn("audit log mkdir failed", "path", path, "error", err)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		logman.Warn("audit log open failed", "path", path, "error", err)
		return
	}

	if s.file != nil {
		s.file.Close()
	}
	s.file = f
	s.dateKey = date
}

// auditFilePath 返回指定日期的日志文件绝对路径。
func auditFilePath(date string) string {
	return filepath.Join(config.Server.RootDirectory, "audit", date+".log")
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
