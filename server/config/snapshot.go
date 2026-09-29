package config

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/goccy/go-yaml"
)

// Snapshot 是一次完整、发布后只读的运行时配置。
type Snapshot struct {
	Server      *ServerConfig            `yaml:"server"`
	Password    *PasswordConfig          `yaml:"password"`
	Passkey     *PasskeyConfig           `yaml:"passkey"`
	OIDC        *OIDCConfig              `yaml:"oidc"`
	THA         *THAConfig               `yaml:"tha"`
	Copilot     *CopilotConfig           `yaml:"copilot"`
	Notify      *NotifyConfig            `yaml:"notify"`
	Apisix      *ApisixConfig            `yaml:"apisix"`
	Caddy       *CaddyConfig             `yaml:"caddy"`
	Docker      *DockerConfig            `yaml:"docker"`
	Monitor     *MonitorConfig           `yaml:"monitor"`
	Marketplace *MarketplaceConfig       `yaml:"marketplace"`
	Links       []*LinkConfig            `yaml:"links"`
	Members     map[string]*MemberConfig `yaml:"members"`
}

var (
	current  atomic.Pointer[Snapshot]
	updateMu sync.Mutex
)

func init() {
	snapshot, err := buildSnapshot(&Config{})
	if err != nil {
		panic(err)
	}
	current.Store(snapshot)
}

// Current 返回当前不可变配置快照。调用方不得修改返回值及其嵌套对象。
func Current() *Snapshot {
	return current.Load()
}

func buildSnapshot(conf *Config) (*Snapshot, error) {
	if conf == nil {
		return nil, fmt.Errorf("配置不能为空")
	}
	var cloned Config
	buf, err := yaml.Marshal(conf)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(buf, &cloned); err != nil {
		return nil, err
	}

	snapshot := &Snapshot{
		Server:      cloned.Server,
		Password:    cloned.Password,
		Passkey:     cloned.Passkey,
		OIDC:        cloned.OIDC,
		THA:         cloned.THA,
		Copilot:     cloned.Copilot,
		Notify:      cloned.Notify,
		Apisix:      cloned.Apisix,
		Caddy:       cloned.Caddy,
		Docker:      cloned.Docker,
		Monitor:     cloned.Monitor,
		Marketplace: cloned.Marketplace,
		Links:       cloned.Links,
		Members:     make(map[string]*MemberConfig, len(cloned.Members)),
	}
	for _, member := range cloned.Members {
		if member == nil {
			continue
		}
		if member.Username == "" {
			return nil, fmt.Errorf("成员用户名不能为空")
		}
		if _, exists := snapshot.Members[member.Username]; exists {
			return nil, fmt.Errorf("成员用户名重复: %s", member.Username)
		}
		snapshot.Members[member.Username] = member
	}
	if err := normalizeSnapshot(snapshot); err != nil {
		return nil, err
	}
	if err := validateSnapshot(snapshot); err != nil {
		return nil, err
	}
	return snapshot, nil
}

func cloneSnapshot(snapshot *Snapshot) (*Snapshot, error) {
	if snapshot == nil {
		return buildSnapshot(&Config{})
	}
	buf, err := yaml.Marshal(snapshot)
	if err != nil {
		return nil, err
	}
	var cloned Snapshot
	if err := yaml.Unmarshal(buf, &cloned); err != nil {
		return nil, err
	}
	return &cloned, nil
}

func validateSnapshot(snapshot *Snapshot) error {
	if len(strings.TrimSpace(snapshot.Server.JWTSecret)) < 32 {
		return fmt.Errorf("JWT 密钥长度不能少于 32 个字符")
	}
	if snapshot.Passkey.Enabled {
		if strings.TrimSpace(snapshot.Passkey.RPName) == "" {
			return fmt.Errorf("Passkey rpName 不能为空")
		}
		rpID := strings.TrimSpace(snapshot.Passkey.RPID)
		if rpID == "" || strings.Contains(rpID, "://") || strings.ContainsAny(rpID, "/?#") {
			return fmt.Errorf("Passkey rpId 必须是合法的域名或 IP 地址")
		}
		if len(snapshot.Passkey.RPOrigins) == 0 {
			return fmt.Errorf("Passkey rpOrigins 不能为空")
		}
		for _, origin := range snapshot.Passkey.RPOrigins {
			if err := validateHTTPURL("Passkey rpOrigin", origin); err != nil {
				return err
			}
		}
	}
	if snapshot.OIDC.Enabled {
		if snapshot.OIDC.ClientID == "" {
			return fmt.Errorf("OIDC clientId 不能为空")
		}
		if err := validateHTTPURL("OIDC issuerUrl", snapshot.OIDC.IssuerURL); err != nil {
			return err
		}
		if err := validateHTTPURL("OIDC redirectUrl", snapshot.OIDC.RedirectURL); err != nil {
			return err
		}
	}
	if snapshot.Password.Disabled && !snapshot.Passkey.Enabled && !snapshot.OIDC.Enabled && !snapshot.THA.Enabled {
		return fmt.Errorf("至少需要启用一种登录方式")
	}

	registries := snapshot.Docker.Registries[:0]
	for _, registry := range snapshot.Docker.Registries {
		if registry != nil {
			registries = append(registries, registry)
		}
	}
	snapshot.Docker.Registries = registries
	for username, member := range snapshot.Members {
		if member == nil {
			delete(snapshot.Members, username)
			continue
		}
		if username == "" || member.Username == "" || username != member.Username {
			return fmt.Errorf("成员用户名无效: %q", username)
		}
		passkeys := member.Passkeys[:0]
		for _, passkey := range member.Passkeys {
			if passkey != nil {
				passkeys = append(passkeys, passkey)
			}
		}
		member.Passkeys = passkeys
	}
	return nil
}

func validateHTTPURL(name, value string) error {
	parsed, err := url.ParseRequestURI(strings.TrimSpace(value))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.User != nil {
		return fmt.Errorf("%s 必须是合法的 HTTP(S) 绝对地址", name)
	}
	return nil
}

func generateJWTSecret() (string, error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return "", fmt.Errorf("生成 JWT 密钥失败: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(secret), nil
}

func normalizeSnapshot(snapshot *Snapshot) error {
	snapshot.Server = ServerNormalize(snapshot.Server)
	if strings.TrimSpace(snapshot.Server.JWTSecret) == "" || snapshot.Server.JWTSecret == "your-jwt-secret" {
		secret, err := generateJWTSecret()
		if err != nil {
			return err
		}
		snapshot.Server.JWTSecret = secret
	}
	snapshot.Password = PasswordNormalize(snapshot.Password)
	snapshot.Passkey = PasskeyNormalize(snapshot.Passkey)
	snapshot.OIDC = OIDCNormalize(snapshot.OIDC)
	snapshot.THA = THANormalize(snapshot.THA)
	if snapshot.Copilot == nil {
		snapshot.Copilot = &CopilotConfig{}
	}
	snapshot.Notify = NotifyNormalize(snapshot.Notify)
	if snapshot.Apisix == nil {
		snapshot.Apisix = &ApisixConfig{}
	}
	if snapshot.Caddy == nil {
		snapshot.Caddy = &CaddyConfig{}
	}
	if snapshot.Docker == nil {
		snapshot.Docker = &DockerConfig{}
	}
	snapshot.Docker.ContainerRoot = PathToAbs(snapshot.Docker.ContainerRoot, snapshot.Server.RootDirectory)
	snapshot.Monitor = MonitorNormalize(snapshot.Monitor)
	if snapshot.Marketplace == nil {
		snapshot.Marketplace = &MarketplaceConfig{}
	}
	if snapshot.Members == nil {
		snapshot.Members = map[string]*MemberConfig{}
	}
	for _, member := range snapshot.Members {
		if member != nil {
			member.HomeDirectory = PathToAbs(member.HomeDirectory, snapshot.Server.RootDirectory)
		}
	}
	return nil
}
