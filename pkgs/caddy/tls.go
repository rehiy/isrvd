package caddy

import "encoding/json"

// 本文件定义 TLS app 及 PKI app 相关的配置结构体。

var (
	tlsAppKnownKeys = jsonKeySet(
		"certificates", "automation", "session_tickets", "cache", "dns",
	)
	tlsCertsKnownKeys      = jsonKeySet("load_files", "load_folders", "load_pem", "automate")
	tlsLoadFileKnownKeys   = jsonKeySet("certificate", "key", "tags", "format")
	tlsLoadPEMKnownKeys    = jsonKeySet("certificate", "key", "tags")
	tlsAutomationKnownKeys = jsonKeySet(
		"policies", "on_demand", "ocsp", "renew_interval", "ocsp_interval",
	)
	tlsPolicyKnownKeys = jsonKeySet(
		"subjects", "issuers", "key_type", "on_demand", "must_staple",
		"renewal_window_ratio", "storage",
	)
	pkiAppKnownKeys = jsonKeySet("certificate_authorities")
)

// ----- TLS App -----

// TLSApp tls 应用配置
type TLSApp struct {
	Certificates   *TLSCerts                  `json:"certificates,omitempty"`    // 证书来源配置
	Automation     *TLSAutomation             `json:"automation,omitempty"`      // 自动证书管理
	SessionTickets map[string]any             `json:"session_tickets,omitempty"` // TLS Session Ticket 配置
	Cache          map[string]any             `json:"cache,omitempty"`           // 会话缓存配置
	DNS            map[string]any             `json:"dns,omitempty"`             // DNS 挑战提供者配置
	Extras         map[string]json.RawMessage `json:"-"`                         // 未建模的 TLS app 配置
}

// MarshalJSON 合并已知字段 + Extras。
func (t TLSApp) MarshalJSON() ([]byte, error) {
	type alias TLSApp
	return mergeKnownAndExtras(alias(t), t.Extras)
}

// UnmarshalJSON 收集未建模的 TLS app 配置。
func (t *TLSApp) UnmarshalJSON(data []byte) error {
	type alias TLSApp
	known, extras, err := unmarshalKnownAndExtras[alias](data, tlsAppKnownKeys)
	if err != nil {
		return err
	}
	*t = TLSApp(known)
	t.Extras = extras
	return nil
}

// TLSCerts certificates 字段
type TLSCerts struct {
	LoadFiles   []TLSLoadFile              `json:"load_files,omitempty"`   // 从文件加载证书
	LoadFolders []string                   `json:"load_folders,omitempty"` // 从目录加载证书
	LoadPEM     []TLSLoadPEM               `json:"load_pem,omitempty"`     // 从内联 PEM 加载证书
	Automate    []string                   `json:"automate,omitempty"`     // 自动管理的域名列表
	Extras      map[string]json.RawMessage `json:"-"`                      // 未建模的证书来源配置
}

// MarshalJSON 合并已知字段 + Extras。
func (t TLSCerts) MarshalJSON() ([]byte, error) {
	type alias TLSCerts
	return mergeKnownAndExtras(alias(t), t.Extras)
}

// UnmarshalJSON 收集未建模的证书来源配置。
func (t *TLSCerts) UnmarshalJSON(data []byte) error {
	type alias TLSCerts
	known, extras, err := unmarshalKnownAndExtras[alias](data, tlsCertsKnownKeys)
	if err != nil {
		return err
	}
	*t = TLSCerts(known)
	t.Extras = extras
	return nil
}

// TLSLoadFile 从磁盘加载证书
type TLSLoadFile struct {
	Certificate string                     `json:"certificate"`      // 证书文件路径
	Key         string                     `json:"key"`              // 私钥文件路径
	Tags        []string                   `json:"tags,omitempty"`   // 标签（用于筛选）
	Format      string                     `json:"format,omitempty"` // 编码格式：pem（默认）| der
	Extras      map[string]json.RawMessage `json:"-"`                // 未建模的文件证书配置
}

// MarshalJSON 合并已知字段 + Extras。
func (t TLSLoadFile) MarshalJSON() ([]byte, error) {
	type alias TLSLoadFile
	return mergeKnownAndExtras(alias(t), t.Extras)
}

// UnmarshalJSON 收集未建模的文件证书配置。
func (t *TLSLoadFile) UnmarshalJSON(data []byte) error {
	type alias TLSLoadFile
	known, extras, err := unmarshalKnownAndExtras[alias](data, tlsLoadFileKnownKeys)
	if err != nil {
		return err
	}
	*t = TLSLoadFile(known)
	t.Extras = extras
	return nil
}

// TLSLoadPEM 从内联 PEM 加载证书
type TLSLoadPEM struct {
	Certificate string                     `json:"certificate"`    // PEM 格式证书内容
	Key         string                     `json:"key"`            // PEM 格式私钥内容
	Tags        []string                   `json:"tags,omitempty"` // 标签
	Extras      map[string]json.RawMessage `json:"-"`              // 未建模的内联证书配置
}

// MarshalJSON 合并已知字段 + Extras。
func (t TLSLoadPEM) MarshalJSON() ([]byte, error) {
	type alias TLSLoadPEM
	return mergeKnownAndExtras(alias(t), t.Extras)
}

// UnmarshalJSON 收集未建模的内联证书配置。
func (t *TLSLoadPEM) UnmarshalJSON(data []byte) error {
	type alias TLSLoadPEM
	known, extras, err := unmarshalKnownAndExtras[alias](data, tlsLoadPEMKnownKeys)
	if err != nil {
		return err
	}
	*t = TLSLoadPEM(known)
	t.Extras = extras
	return nil
}

// TLSAutomation 自动化配置
type TLSAutomation struct {
	Policies      []TLSPolicy                `json:"policies,omitempty"`       // 自动化策略列表
	OnDemand      map[string]any             `json:"on_demand,omitempty"`      // 按需证书配置
	OCSP          map[string]any             `json:"ocsp,omitempty"`           // OCSP 装订配置
	RenewInterval Duration                   `json:"renew_interval,omitempty"` // 续期检查间隔（如 "24h"）
	OCSPInterval  Duration                   `json:"ocsp_interval,omitempty"`  // OCSP 更新间隔
	Extras        map[string]json.RawMessage `json:"-"`                        // 未建模的 TLS 自动化配置
}

// MarshalJSON 合并已知字段 + Extras。
func (t TLSAutomation) MarshalJSON() ([]byte, error) {
	type alias TLSAutomation
	return mergeKnownAndExtras(alias(t), t.Extras)
}

// UnmarshalJSON 收集未建模的 TLS 自动化配置。
func (t *TLSAutomation) UnmarshalJSON(data []byte) error {
	type alias TLSAutomation
	known, extras, err := unmarshalKnownAndExtras[alias](data, tlsAutomationKnownKeys)
	if err != nil {
		return err
	}
	*t = TLSAutomation(known)
	t.Extras = extras
	return nil
}

// TLSPolicy 自动化策略
type TLSPolicy struct {
	Subjects        []string                   `json:"subjects,omitempty"`             // 适用的域名列表
	Issuers         []map[string]any           `json:"issuers,omitempty"`              // 证书颁发者配置（acme/internal）
	KeyType         string                     `json:"key_type,omitempty"`             // 密钥类型：rsa2048|p256|ed25519 等
	OnDemand        bool                       `json:"on_demand,omitempty"`            // 是否按需申请证书
	MustStaple      bool                       `json:"must_staple,omitempty"`          // 是否启用 OCSP Must-Staple
	RenewalWindow   float64                    `json:"renewal_window_ratio,omitempty"` // 续期窗口比例（0-1）
	StorageOverride map[string]any             `json:"storage,omitempty"`              // 覆盖默认存储配置
	Extras          map[string]json.RawMessage `json:"-"`                              // 未建模的 TLS 策略配置
}

// MarshalJSON 合并已知字段 + Extras。
func (t TLSPolicy) MarshalJSON() ([]byte, error) {
	type alias TLSPolicy
	return mergeKnownAndExtras(alias(t), t.Extras)
}

// UnmarshalJSON 收集未建模的 TLS 策略配置。
func (t *TLSPolicy) UnmarshalJSON(data []byte) error {
	type alias TLSPolicy
	known, extras, err := unmarshalKnownAndExtras[alias](data, tlsPolicyKnownKeys)
	if err != nil {
		return err
	}
	*t = TLSPolicy(known)
	t.Extras = extras
	return nil
}

// ----- PKI App -----

// PKIApp pki 应用配置（内部 CA）
type PKIApp struct {
	CAs    map[string]map[string]any  `json:"certificate_authorities,omitempty"` // CA 名称 → 配置
	Extras map[string]json.RawMessage `json:"-"`                                 // 未建模的 PKI app 配置
}

// MarshalJSON 合并已知字段 + Extras。
func (p PKIApp) MarshalJSON() ([]byte, error) {
	type alias PKIApp
	return mergeKnownAndExtras(alias(p), p.Extras)
}

// UnmarshalJSON 收集未建模的 PKI app 配置。
func (p *PKIApp) UnmarshalJSON(data []byte) error {
	type alias PKIApp
	known, extras, err := unmarshalKnownAndExtras[alias](data, pkiAppKnownKeys)
	if err != nil {
		return err
	}
	*p = PKIApp(known)
	p.Extras = extras
	return nil
}
