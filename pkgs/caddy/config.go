package caddy

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// 本文件定义 Caddy 顶层配置结构体及内部 unknown fields 透传辅助函数。

// ----- 顶层 -----

var (
	configKnownKeys    = jsonKeySet("admin", "logging", "storage", "apps", "@id")
	adminKnownKeys     = jsonKeySet("disabled", "listen", "enforce_origin", "origins", "config")
	adminAutoKnownKeys = jsonKeySet("persist")
	loggingKnownKeys   = jsonKeySet("sink", "logs", "@id")
	logSinkKnownKeys   = jsonKeySet("writer")
	logKnownKeys       = jsonKeySet("writer", "encoder", "level", "sampling", "include", "exclude")
	appsKnownKeys      = jsonKeySet("http", "tls", "pki")
)

// ID 是 Caddy 配置对象的 @id，原样保留字符串或数字表示。
//
// Caddy 同时接受字符串和数字 ID；使用 RawMessage 的命名类型避免数字经过
// float64 后丢失精度，也避免把数字 ID 强制改写成字符串。
type ID json.RawMessage

// MarshalJSON 原样输出 ID。
func (id ID) MarshalJSON() ([]byte, error) {
	if len(id) == 0 {
		return []byte("null"), nil
	}
	if !json.Valid(id) {
		return nil, fmt.Errorf("无效的 caddy @id JSON: %q", string(id))
	}
	return id, nil
}

// UnmarshalJSON 接受字符串、数字或 null，保留原始 JSON 表示。
func (id *ID) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if bytes.Equal(trimmed, []byte("null")) {
		*id = nil
		return nil
	}
	if len(trimmed) == 0 {
		return fmt.Errorf("caddy @id 不能为空")
	}
	if trimmed[0] == '"' {
		var value string
		if err := json.Unmarshal(trimmed, &value); err != nil {
			return fmt.Errorf("caddy @id 必须是字符串或数字: %w", err)
		}
	} else {
		var value json.Number
		if err := json.Unmarshal(trimmed, &value); err != nil {
			return fmt.Errorf("caddy @id 必须是字符串或数字: %w", err)
		}
	}
	*id = append((*id)[:0], trimmed...)
	return nil
}

// String 返回 ID 的展示文本；字符串 ID 去除 JSON 引号，数字保持原样。
func (id ID) String() string {
	if len(id) == 0 {
		return ""
	}
	var value string
	if json.Unmarshal(id, &value) == nil {
		return value
	}
	return string(id)
}

// Config Caddy 顶层配置
//
// Extras 用于保留未建模的顶层字段，Marshal/Unmarshal 时自动透传。
type Config struct {
	Admin   *AdminConfig   `json:"admin,omitempty"`   // Admin API 配置
	Logging *LoggingConfig `json:"logging,omitempty"` // 日志配置
	Storage map[string]any `json:"storage,omitempty"` // 存储后端配置
	Apps    *AppsConfig    `json:"apps,omitempty"`    // 应用配置（http/tls/pki 等）
	ID      ID             `json:"@id,omitempty"`     // 配置 ID（用于引用）

	// Extras 保存所有未识别的顶层字段；Marshal 时与已知字段合并输出
	Extras map[string]json.RawMessage `json:"-"`
}

// MarshalJSON 合并已知字段 + Extras 输出
func (c Config) MarshalJSON() ([]byte, error) {
	type alias Config // 借助类型别名避免递归
	return mergeKnownAndExtras(alias(c), c.Extras)
}

// UnmarshalJSON 解析时把未知字段收集到 Extras
func (c *Config) UnmarshalJSON(data []byte) error {
	type alias Config
	known, extras, err := unmarshalKnownAndExtras[alias](data, configKnownKeys)
	if err != nil {
		return err
	}
	*c = Config(known)
	c.Extras = extras
	return nil
}

// AdminConfig admin 端点配置
type AdminConfig struct {
	Disabled      bool                       `json:"disabled,omitempty"`       // 是否禁用 admin API
	Listen        string                     `json:"listen,omitempty"`         // 监听地址，如 "localhost:2019"
	EnforceOrigin bool                       `json:"enforce_origin,omitempty"` // 是否校验 Origin
	Origins       []string                   `json:"origins,omitempty"`        // 允许的 Origin 列表
	Config        *AdminAutoConfig           `json:"config,omitempty"`         // 自动配置（如持久化）
	Extras        map[string]json.RawMessage `json:"-"`                        // 未建模的 admin 配置
}

// MarshalJSON 合并已知字段 + Extras。
func (a AdminConfig) MarshalJSON() ([]byte, error) {
	type alias AdminConfig
	return mergeKnownAndExtras(alias(a), a.Extras)
}

// UnmarshalJSON 收集未建模的 admin 配置。
func (a *AdminConfig) UnmarshalJSON(data []byte) error {
	type alias AdminConfig
	known, extras, err := unmarshalKnownAndExtras[alias](data, adminKnownKeys)
	if err != nil {
		return err
	}
	*a = AdminConfig(known)
	a.Extras = extras
	return nil
}

// AdminAutoConfig admin.config，例如 persist 持久化
type AdminAutoConfig struct {
	Persist *bool                      `json:"persist,omitempty"` // 是否持久化配置到磁盘
	Extras  map[string]json.RawMessage `json:"-"`                 // 未建模的自动配置
}

// MarshalJSON 合并已知字段 + Extras。
func (a AdminAutoConfig) MarshalJSON() ([]byte, error) {
	type alias AdminAutoConfig
	return mergeKnownAndExtras(alias(a), a.Extras)
}

// UnmarshalJSON 收集未建模的自动配置。
func (a *AdminAutoConfig) UnmarshalJSON(data []byte) error {
	type alias AdminAutoConfig
	known, extras, err := unmarshalKnownAndExtras[alias](data, adminAutoKnownKeys)
	if err != nil {
		return err
	}
	*a = AdminAutoConfig(known)
	a.Extras = extras
	return nil
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Sink   *LogSink                   `json:"sink,omitempty"` // 全局日志输出配置
	Logs   map[string]*Log            `json:"logs,omitempty"` // 各模块日志配置
	ID     ID                         `json:"@id,omitempty"`  // 配置 ID
	Extras map[string]json.RawMessage `json:"-"`              // 未建模的日志配置
}

// MarshalJSON 合并已知字段 + Extras。
func (l LoggingConfig) MarshalJSON() ([]byte, error) {
	type alias LoggingConfig
	return mergeKnownAndExtras(alias(l), l.Extras)
}

// UnmarshalJSON 收集未建模的日志配置。
func (l *LoggingConfig) UnmarshalJSON(data []byte) error {
	type alias LoggingConfig
	known, extras, err := unmarshalKnownAndExtras[alias](data, loggingKnownKeys)
	if err != nil {
		return err
	}
	*l = LoggingConfig(known)
	l.Extras = extras
	return nil
}

// LogSink 全局 sink
type LogSink struct {
	Writer map[string]any             `json:"writer,omitempty"` // 日志写入器配置
	Extras map[string]json.RawMessage `json:"-"`                // 未建模的 sink 配置
}

// MarshalJSON 合并已知字段 + Extras。
func (l LogSink) MarshalJSON() ([]byte, error) {
	type alias LogSink
	return mergeKnownAndExtras(alias(l), l.Extras)
}

// UnmarshalJSON 收集未建模的 sink 配置。
func (l *LogSink) UnmarshalJSON(data []byte) error {
	type alias LogSink
	known, extras, err := unmarshalKnownAndExtras[alias](data, logSinkKnownKeys)
	if err != nil {
		return err
	}
	*l = LogSink(known)
	l.Extras = extras
	return nil
}

// Log 单个 logger
type Log struct {
	Writer   map[string]any             `json:"writer,omitempty"`   // 写入器
	Encoder  map[string]any             `json:"encoder,omitempty"`  // 编码器（json/formatted）
	Level    string                     `json:"level,omitempty"`    // 日志级别：DEBUG|INFO|WARN|ERROR
	Sampling map[string]any             `json:"sampling,omitempty"` // 采样配置
	Include  []string                   `json:"include,omitempty"`  // 包含的模块列表
	Exclude  []string                   `json:"exclude,omitempty"`  // 排除的模块列表
	Extras   map[string]json.RawMessage `json:"-"`                  // 未建模的 logger 配置
}

// MarshalJSON 合并已知字段 + Extras。
func (l Log) MarshalJSON() ([]byte, error) {
	type alias Log
	return mergeKnownAndExtras(alias(l), l.Extras)
}

// UnmarshalJSON 收集未建模的 logger 配置。
func (l *Log) UnmarshalJSON(data []byte) error {
	type alias Log
	known, extras, err := unmarshalKnownAndExtras[alias](data, logKnownKeys)
	if err != nil {
		return err
	}
	*l = Log(known)
	l.Extras = extras
	return nil
}

// AppsConfig 应用集合
//
// Extras 透传其他 app（layer4 / dynamic_dns / 第三方等）。
type AppsConfig struct {
	HTTP *HTTPApp `json:"http,omitempty"` // HTTP 应用配置
	TLS  *TLSApp  `json:"tls,omitempty"`  // TLS 应用配置
	PKI  *PKIApp  `json:"pki,omitempty"`  // PKI（CA）应用配置

	Extras map[string]json.RawMessage `json:"-"` // 其他未建模的 app 配置
}

// MarshalJSON 合并已知字段 + Extras
func (a AppsConfig) MarshalJSON() ([]byte, error) {
	type alias AppsConfig
	return mergeKnownAndExtras(alias(a), a.Extras)
}

// UnmarshalJSON 收集未知 app 到 Extras
func (a *AppsConfig) UnmarshalJSON(data []byte) error {
	type alias AppsConfig
	known, extras, err := unmarshalKnownAndExtras[alias](data, appsKnownKeys)
	if err != nil {
		return err
	}
	*a = AppsConfig(known)
	a.Extras = extras
	return nil
}

// ----- 内部辅助：unknown fields 透传 -----

// mergeKnownAndExtras 合并已知字段与 extras 输出 JSON 对象
//
// 实现技巧：直接在已知 JSON 的末尾 `}` 之前插入 extras 字段，
// 避免再次 unmarshal/marshal 整个对象。
func mergeKnownAndExtras(known any, extras map[string]json.RawMessage) ([]byte, error) {
	knownRaw, err := json.Marshal(known)
	if err != nil {
		return nil, err
	}
	if len(extras) == 0 {
		return knownRaw, nil
	}
	var emitted map[string]json.RawMessage
	if err := json.Unmarshal(knownRaw, &emitted); err != nil {
		return nil, err
	}
	// known 必为 JSON 对象，否则不合并
	end := bytes.LastIndexByte(knownRaw, '}')
	if end < 0 {
		return knownRaw, nil
	}

	var buf bytes.Buffer
	buf.Grow(len(knownRaw) + 64*len(extras))
	prefix := knownRaw[:end]
	buf.Write(prefix)
	// 如果 {} 之间有内容（即有已知字段），首个 extra 前需要加逗号
	needComma := len(bytes.TrimSpace(prefix)) > 1 // `{` 之后还有内容
	for k, v := range extras {
		// 已知字段的当前非空值优先；Extras 中的同名值只用于
		// 记住 omitempty 会丢失的空对象/空数组存在性。
		if _, ok := emitted[k]; ok {
			continue
		}
		if needComma {
			buf.WriteByte(',')
		}
		needComma = true
		key, _ := json.Marshal(k)
		buf.Write(key)
		buf.WriteByte(':')
		buf.Write(v)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// pickExtras 从原始 JSON 中挑出未建模字段，并记住已知字段的
// 空对象/空数组存在性，避免 omitempty 在结构化读改写中删除它们。
func pickExtras(data []byte, known map[string]struct{}) (map[string]json.RawMessage, error) {
	var all map[string]json.RawMessage
	if err := json.Unmarshal(data, &all); err != nil {
		// 非对象（null 等）忽略
		return nil, nil
	}
	var extras map[string]json.RawMessage
	for k, v := range all {
		if _, ok := known[k]; ok {
			trimmed := bytes.TrimSpace(v)
			if !bytes.Equal(trimmed, []byte("{}")) && !bytes.Equal(trimmed, []byte("[]")) {
				continue
			}
		}
		if extras == nil {
			extras = map[string]json.RawMessage{}
		}
		extras[k] = v
	}
	return extras, nil
}

func unmarshalKnownAndExtras[T any](data []byte, knownKeys map[string]struct{}) (T, map[string]json.RawMessage, error) {
	var known T
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&known); err != nil {
		return known, nil, err
	}
	extras, err := pickExtras(data, knownKeys)
	return known, extras, err
}

func jsonKeySet(keys ...string) map[string]struct{} {
	set := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		set[key] = struct{}{}
	}
	return set
}
