package caddy

// 本文件定义 TLS app 及 PKI app 相关的配置结构体。

// ----- TLS App -----

// TLSApp tls 应用配置
type TLSApp struct {
	Certificates   *TLSCerts      `json:"certificates,omitempty"`
	Automation     *TLSAutomation `json:"automation,omitempty"`
	SessionTickets map[string]any `json:"session_tickets,omitempty"`
	Cache          map[string]any `json:"cache,omitempty"`
	DNS            map[string]any `json:"dns,omitempty"`
}

// TLSCerts certificates 字段
type TLSCerts struct {
	LoadFiles   []TLSLoadFile `json:"load_files,omitempty"`
	LoadFolders []string      `json:"load_folders,omitempty"`
	LoadPEM     []TLSLoadPEM  `json:"load_pem,omitempty"`
	Automate    []string      `json:"automate,omitempty"`
}

// TLSLoadFile 从磁盘加载证书
type TLSLoadFile struct {
	Certificate string   `json:"certificate"`
	Key         string   `json:"key"`
	Tags        []string `json:"tags,omitempty"`
	Format      string   `json:"format,omitempty"`
}

// TLSLoadPEM 从内联 PEM 加载证书
type TLSLoadPEM struct {
	Certificate string   `json:"certificate"`
	Key         string   `json:"key"`
	Tags        []string `json:"tags,omitempty"`
}

// TLSAutomation 自动化配置
type TLSAutomation struct {
	Policies      []TLSPolicy    `json:"policies,omitempty"`
	OnDemand      map[string]any `json:"on_demand,omitempty"`
	OCSP          map[string]any `json:"ocsp,omitempty"`
	RenewInterval string         `json:"renew_interval,omitempty"`
	OCSPInterval  string         `json:"ocsp_interval,omitempty"`
}

// TLSPolicy 自动化策略
type TLSPolicy struct {
	Subjects        []string         `json:"subjects,omitempty"`
	Issuers         []map[string]any `json:"issuers,omitempty"`
	KeyType         string           `json:"key_type,omitempty"`
	OnDemand        bool             `json:"on_demand,omitempty"`
	MustStaple      bool             `json:"must_staple,omitempty"`
	RenewalWindow   float64          `json:"renewal_window_ratio,omitempty"`
	StorageOverride map[string]any   `json:"storage,omitempty"`
}

// ----- PKI App -----

// PKIApp pki 应用配置
type PKIApp struct {
	CAs map[string]map[string]any `json:"certificate_authorities,omitempty"`
}
