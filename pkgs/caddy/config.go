package caddy

import (
	"bytes"
	"encoding/json"
)

// 本文件定义 Caddy 顶层配置结构体及内部 unknown fields 透传辅助函数。

// ----- 顶层 -----

// Config Caddy 顶层配置
//
// Extras 用于保留未建模的顶层字段，Marshal/Unmarshal 时自动透传。
type Config struct {
	Admin   *AdminConfig   `json:"admin,omitempty"`
	Logging *LoggingConfig `json:"logging,omitempty"`
	Storage map[string]any `json:"storage,omitempty"`
	Apps    *AppsConfig    `json:"apps,omitempty"`
	ID      string         `json:"@id,omitempty"`

	// Extras 保存所有未识别的顶层字段；Marshal 时与已知字段合并输出
	Extras map[string]json.RawMessage `json:"-"`
}

// configKnownKeys 已建模的顶层 JSON key，用于分离 unknown fields
var configKnownKeys = map[string]struct{}{
	"admin": {}, "logging": {}, "storage": {}, "apps": {}, "@id": {},
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
	Disabled      bool             `json:"disabled,omitempty"`
	Listen        string           `json:"listen,omitempty"`
	EnforceOrigin bool             `json:"enforce_origin,omitempty"`
	Origins       []string         `json:"origins,omitempty"`
	Config        *AdminAutoConfig `json:"config,omitempty"`
}

// AdminAutoConfig admin.config，例如 persist 持久化
type AdminAutoConfig struct {
	Persist *bool `json:"persist,omitempty"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Sink *LogSink        `json:"sink,omitempty"`
	Logs map[string]*Log `json:"logs,omitempty"`
	ID   string          `json:"@id,omitempty"`
}

// LogSink 全局 sink
type LogSink struct {
	Writer map[string]any `json:"writer,omitempty"`
}

// Log 单个 logger
type Log struct {
	Writer   map[string]any `json:"writer,omitempty"`
	Encoder  map[string]any `json:"encoder,omitempty"`
	Level    string         `json:"level,omitempty"`
	Sampling map[string]any `json:"sampling,omitempty"`
	Include  []string       `json:"include,omitempty"`
	Exclude  []string       `json:"exclude,omitempty"`
}

// AppsConfig 应用集合
//
// Extras 透传其他 app（layer4 / dynamic_dns / 第三方等）。
type AppsConfig struct {
	HTTP *HTTPApp `json:"http,omitempty"`
	TLS  *TLSApp  `json:"tls,omitempty"`
	PKI  *PKIApp  `json:"pki,omitempty"`

	Extras map[string]json.RawMessage `json:"-"`
}

// appsKnownKeys 已建模的 apps JSON key
var appsKnownKeys = map[string]struct{}{
	"http": {}, "tls": {}, "pki": {},
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
