// Package system 提供系统级业务服务。
package system

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"isrvd/server/i18n"

	"github.com/rehiy/libgo/jsonl"
	"github.com/rehiy/libgo/logman"

	"isrvd/pkgs/helper"
	"isrvd/server/config"
)

const (
	maxAuditBufferSize = 100      // 内存缓冲最大条数
	maxAuditBodySize   = 64 << 10 // 审计日志中请求体最多保留 64 KiB（脱敏后截断）
	maxAuditParseSize  = 4 << 20  // JSON/表单请求体脱敏解析的最大读取量 4 MiB，超出部分丢弃
)

// sensitiveFields 审计请求体与 URI 中需要脱敏的字段名
var sensitiveFields = []string{
	// 系统配置密钥：JWT、Copilot、APISIX、OIDC
	"jwtSecret", "apiKey", "adminKey", "clientSecret",
	// 跨模块密码与共享密钥：账户、SSH、镜像仓库、APISIX 认证插件
	"password", "secret",
	// 账户密码变更、认证码与双因素验证码
	"oldPassword", "newPassword", "code", "totpCode",
	// 访问令牌
	"token", "nodeToken",
	// 文件、脚本、环境变量与 SSH 私钥内容不进入审计日志
	"content", "envContent", "privateKey",
	// APISIX key-auth 密钥与 SSL 证书私钥
	"key",
	// APISIX 插件专用字段
	"client_secret",        // openid-connect
	"public_key",           // jwt-auth
	"key_id", "secret_key", // hmac-auth
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
	buffer []AuditLog
	store  *jsonl.Store
	closed bool
	mu     sync.RWMutex
}

// NewAuditService 创建审计日志业务服务并自动初始化
func NewAuditService() *AuditService {
	dataDir := filepath.Join(config.Current().Server.RootDirectory, "audit")
	store, err := jsonl.New(
		dataDir,
		jsonl.Naming{Suffix: ".jsonl"},
		jsonl.WithBufferSize(4096),
		jsonl.WithFlushInterval(time.Second),
		jsonl.WithAsync(maxAuditBufferSize),
	)
	if err != nil {
		logman.Warn("audit log store init failed", "dir", dataDir, "error", err)
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
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	if len(s.buffer) >= maxAuditBufferSize {
		// 复用固定容量缓冲区，移除最旧记录后由 append 覆盖尾部引用。
		s.buffer = slices.Delete(s.buffer, 0, 1)
	}
	s.buffer = append(s.buffer, entry)

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
	for _, entry := range slices.Backward(s.buffer) {
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

// AuditRecord 根据请求类型记录审计日志，供中间件在请求处理完成后调用。
// WebSocket 升级请求记录 "WS" 方法；其余记录方法、URI、请求体、状态码。
func (s *AuditService) AuditRecord(r *http.Request, username, ip string, statusCode int, startTime time.Time, body string) {
	username = auditUsername(username, body, r)

	// WebSocket
	if helper.IsWebSocket(r) {
		if statusCode == 0 || statusCode == http.StatusOK {
			statusCode = http.StatusSwitchingProtocols
		}
		s.LogAdd(AuditLog{
			Timestamp:  startTime,
			Username:   username,
			Method:     "WS",
			URI:        maskSensitiveURI(r.RequestURI),
			IP:         ip,
			StatusCode: statusCode,
			Success:    statusCode == http.StatusSwitchingProtocols,
			Duration:   time.Since(startTime).Milliseconds(),
		})
		return
	}

	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	s.LogAdd(AuditLog{
		Timestamp:  startTime,
		Username:   username,
		Method:     r.Method,
		URI:        maskSensitiveURI(r.RequestURI),
		Body:       body,
		IP:         ip,
		StatusCode: statusCode,
		Success:    statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices,
		Duration:   time.Since(startTime).Milliseconds(),
	})
}

// BodyRead 读取有限长度的请求体用于审计，并确保后续 handler 仍可读取完整内容。
// 文件上传只记录占位符，避免审计层提前解析 multipart 并占用大量内存或临时磁盘。
func (s *AuditService) BodyRead(r *http.Request) string {
	contentType := r.Header.Get("Content-Type")
	switch {
	case strings.HasPrefix(contentType, "application/octet-stream"):
		return "[Binary Omitted]"
	case strings.HasPrefix(contentType, "multipart/form-data"):
		return "[Multipart Omitted]"
	}

	isJSON := strings.Contains(contentType, "json")
	isForm := strings.HasPrefix(contentType, "application/x-www-form-urlencoded")
	if !isJSON && !isForm {
		return "[Body Omitted]"
	}

	original := r.Body
	if original == nil {
		return ""
	}
	raw, err := io.ReadAll(io.LimitReader(original, maxAuditParseSize+1))
	r.Body = struct {
		io.Reader
		io.Closer
	}{
		Reader: io.MultiReader(bytes.NewReader(raw), original),
		Closer: original,
	}
	if err != nil {
		return "[Body Read Failed]"
	}

	truncated := len(raw) > maxAuditParseSize
	if truncated {
		raw = raw[:maxAuditParseSize]
	}

	var body string
	if isJSON {
		body = maskSensitiveJSON(raw)
	} else {
		if truncated {
			// 丢弃末尾可能不完整的键值对
			raw = raw[:max(bytes.LastIndexByte(raw, '&'), 0)]
		}
		body = maskSensitiveForm(string(raw))
	}

	// 脱敏后的结果按审计上限截断，避免切断多字节字符
	if len(body) > maxAuditBodySize {
		n := maxAuditBodySize
		for n > 0 && !utf8.RuneStart(body[n]) {
			n--
		}
		body = body[:n]
		truncated = true
	}
	if truncated {
		body += " [Truncated]"
	}
	return body
}

// Close 关闭底层文件句柄，刷盘缓冲数据
func (s *AuditService) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	if s.store == nil {
		return nil
	}
	return s.store.Close()
}

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
	for _, entry := range slices.Backward(entries) {
		s.buffer = append(s.buffer, entry)
	}
}

// ─── 辅助函数 ───

// auditUsername 获取审计日志中的操作人。
// 登录等匿名路由没有认证上下文时，尝试从 JSON 请求体读取 username，仍为空则按请求语言标记为匿名。
func auditUsername(authUsername, body string, r *http.Request) string {
	if username := strings.TrimSpace(authUsername); username != "" {
		return username
	}

	var data map[string]any
	if json.Unmarshal([]byte(body), &data) == nil {
		if username, ok := data["username"].(string); ok && strings.TrimSpace(username) != "" {
			return strings.TrimSpace(username)
		}
	}

	return i18n.TR(r, "匿名")
}

// isSensitiveField 判断字段名是否需要脱敏（忽略大小写）。
func isSensitiveField(key string) bool {
	for _, field := range sensitiveFields {
		if strings.EqualFold(key, field) {
			return true
		}
	}
	return false
}

// maskSensitiveURI 隐藏查询参数中的密钥，保留其他参数的顺序和原始编码。
func maskSensitiveURI(uri string) string {
	path, query, ok := strings.Cut(uri, "?")
	if !ok {
		return uri
	}
	parts := strings.Split(query, "&")
	for i, part := range parts {
		key, _, _ := strings.Cut(part, "=")
		name, err := url.QueryUnescape(key)
		if err != nil {
			continue
		}
		if isSensitiveField(name) {
			parts[i] = key + "=******"
		}
	}
	return path + "?" + strings.Join(parts, "&")
}

// maskSensitiveJSON 流式脱敏 JSON 中的敏感字段。
// 输入被截断或中途非法时，保留已解析出的前部内容（输出可能不是合法 JSON）；
// 敏感字段无论值类型都只输出 [REDACTED]，被截断的敏感值同样不会泄露。
func maskSensitiveJSON(raw []byte) string {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	tok, err := dec.Token()
	if err != nil {
		return "[Invalid JSON Omitted]"
	}
	var sb strings.Builder
	_ = writeMaskedJSON(dec, &sb, tok)
	return sb.String()
}

// writeMaskedJSON 递归输出以 tok 开头的 JSON 值，遇到解析错误即返回。
func writeMaskedJSON(dec *json.Decoder, sb *strings.Builder, tok json.Token) error {
	delim, ok := tok.(json.Delim)
	if !ok {
		value, _ := json.Marshal(tok)
		sb.Write(value)
		return nil
	}

	isObject := delim == '{'
	sb.WriteRune(rune(delim))
	for i := 0; dec.More(); i++ {
		if i > 0 {
			sb.WriteByte(',')
		}
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		if isObject {
			key, _ := tok.(string)
			quoted, _ := json.Marshal(key)
			sb.Write(quoted)
			sb.WriteByte(':')
			if isSensitiveField(key) {
				sb.WriteString(`"[REDACTED]"`)
				var skip json.RawMessage
				if err := dec.Decode(&skip); err != nil {
					return err
				}
				continue
			}
			if tok, err = dec.Token(); err != nil {
				return err
			}
		}
		if err := writeMaskedJSON(dec, sb, tok); err != nil {
			return err
		}
	}
	if _, err := dec.Token(); err != nil {
		return err
	}
	if isObject {
		sb.WriteByte('}')
	} else {
		sb.WriteByte(']')
	}
	return nil
}

// maskSensitiveForm 对 URL 编码表单中的敏感字段进行脱敏。
func maskSensitiveForm(raw string) string {
	values, err := url.ParseQuery(raw)
	if err != nil {
		return "[Invalid Form Omitted]"
	}
	for key, items := range values {
		if isSensitiveField(key) {
			for i := range items {
				items[i] = "[REDACTED]"
			}
		}
	}
	return values.Encode()
}
