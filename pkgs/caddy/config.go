package caddy

import (
	"bytes"
	"encoding/json"
)

// 本文件定义 Caddy 顶层配置结构体及内部 unknown fields 透传辅助函数。

// ----- 顶层 -----

// configKnownKeys 已建模的顶层 JSON key，用于分离 unknown fields
var configKnownKeys = map[string]struct{}{
	"admin": {}, "logging": {}, "storage": {}, "apps": {}, "@id": {},
}

// appsKnownKeys 已建模的 apps JSON key
var appsKnownKeys = map[string]struct{}{
	"http": {}, "tls": {}, "pki": {},
}

// Config Caddy 顶层配置
//
// Extras 用于保留未建模的顶层字段，Marshal/Unmarshal 时自动透传。
type Config struct {
	Admin   *AdminConfig   `json:"admin,omitempty"`   // Admin API 配置
	Logging *LoggingConfig `json:"logging,omitempty"` // 日志配置
	Storage map[string]any `json:"storage,omitempty"` // 存储后端配置
	Apps    *AppsConfig    `json:"apps,omitempty"`    // 应用配置（http/tls/pki 等）
	ID      string         `json:"@id,omitempty"`     // 配置 ID（用于引用）

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
	var known alias
	if err := json.Unmarshal(data, &known); err != nil {
		return err
	}
	*c = Config(known)

	extras, err := pickExtras(data, configKnownKeys)
	if err != nil {
		return err
	}
	c.Extras = extras
	return nil
}

// AdminConfig admin 端点配置
type AdminConfig struct {
	Disabled      bool             `json:"disabled,omitempty"`       // 是否禁用 admin API
	Listen        string           `json:"listen,omitempty"`         // 监听地址，如 "localhost:2019"
	EnforceOrigin bool             `json:"enforce_origin,omitempty"` // 是否校验 Origin
	Origins       []string         `json:"origins,omitempty"`        // 允许的 Origin 列表
	Config        *AdminAutoConfig `json:"config,omitempty"`         // 自动配置（如持久化）
}

// AdminAutoConfig admin.config，例如 persist 持久化
type AdminAutoConfig struct {
	Persist *bool `json:"persist,omitempty"` // 是否持久化配置到磁盘
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Sink *LogSink        `json:"sink,omitempty"` // 全局日志输出配置
	Logs map[string]*Log `json:"logs,omitempty"` // 各模块日志配置
	ID   string          `json:"@id,omitempty"`  // 配置 ID
}

// LogSink 全局 sink
type LogSink struct {
	Writer map[string]any `json:"writer,omitempty"` // 日志写入器配置
}

// Log 单个 logger
type Log struct {
	Writer   map[string]any `json:"writer,omitempty"`   // 写入器
	Encoder  map[string]any `json:"encoder,omitempty"`  // 编码器（json/formatted）
	Level    string         `json:"level,omitempty"`    // 日志级别：DEBUG|INFO|WARN|ERROR
	Sampling map[string]any `json:"sampling,omitempty"` // 采样配置
	Include  []string       `json:"include,omitempty"`  // 包含的模块列表
	Exclude  []string       `json:"exclude,omitempty"`  // 排除的模块列表
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
	var known alias
	if err := json.Unmarshal(data, &known); err != nil {
		return err
	}
	*a = AppsConfig(known)

	extras, err := pickExtras(data, appsKnownKeys)
	if err != nil {
		return err
	}
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
	// known 必为对象，否则不合并
	end := bytes.LastIndexByte(knownRaw, '}')
	if end < 0 {
		return knownRaw, nil
	}

	var buf bytes.Buffer
	buf.Grow(len(knownRaw) + 64*len(extras))
	buf.Write(knownRaw[:end])
	hasKnown := end > 1 // {} 时 end == 1
	for k, v := range extras {
		if hasKnown {
			buf.WriteByte(',')
		}
		hasKnown = true
		key, _ := json.Marshal(k)
		buf.Write(key)
		buf.WriteByte(':')
		buf.Write(v)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// pickExtras 从原始 JSON 中挑出不在 known 集合里的字段
func pickExtras(data []byte, known map[string]struct{}) (map[string]json.RawMessage, error) {
	var all map[string]json.RawMessage
	if err := json.Unmarshal(data, &all); err != nil {
		// 非对象（null 等）忽略
		return nil, nil
	}
	var extras map[string]json.RawMessage
	for k, v := range all {
		if _, ok := known[k]; ok {
			continue
		}
		if extras == nil {
			extras = map[string]json.RawMessage{}
		}
		extras[k] = v
	}
	return extras, nil
}
