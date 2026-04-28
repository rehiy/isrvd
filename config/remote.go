package config

import (
	"context"
	"encoding/json"
	"fmt"
)

// RemoteStore 抽象远程配置存储（etcd / 未来可扩展）
type RemoteStore interface {
	Load(ctx context.Context) (*RemoteConfig, int64, error)
	Save(ctx context.Context, rc *RemoteConfig) error
	Watch(ctx context.Context, rev int64, onChange func(key string, value []byte)) error
	Close() error
}

// RemoteConfig 需要从远程存储加载/保存的全局配置段
type RemoteConfig struct {
	Agent       *AgentConfig       `json:"agent"`
	Apisix      *ApisixRemote      `json:"apisix"`
	Marketplace *MarketplaceConfig `json:"marketplace"`
	Links       []*LinkConfig      `json:"links"`
	Members     []*MemberConfig    `json:"members"`
	Docker      *DockerRemote      `json:"docker"`
	Server      *ServerRemote      `json:"server"`
}

// ApisixRemote apisix 全局段（仅 adminKey）
type ApisixRemote struct {
	AdminKey string `json:"adminKey"`
}

// ServerRemote server 全局段（仅 jwtSecret + proxyHeaderName）
type ServerRemote struct {
	JWTSecret       string `json:"jwtSecret"`
	ProxyHeaderName string `json:"proxyHeaderName"`
}

// DockerRemote docker 全局段（仅 registries）
type DockerRemote struct {
	Registries []*DockerRegistry `json:"registries"`
}

// extractRemote 从完整 Config 中提取全局段
func extractRemote(conf *Config) *RemoteConfig {
	rc := &RemoteConfig{}
	if conf.Agent != nil {
		rc.Agent = &AgentConfig{
			Model:   conf.Agent.Model,
			BaseURL: conf.Agent.BaseURL,
			APIKey:  conf.Agent.APIKey,
		}
	}
	if conf.Apisix != nil {
		rc.Apisix = &ApisixRemote{AdminKey: conf.Apisix.AdminKey}
	}
	if conf.Marketplace != nil {
		rc.Marketplace = &MarketplaceConfig{URL: conf.Marketplace.URL}
	}
	if conf.Links != nil {
		rc.Links = make([]*LinkConfig, len(conf.Links))
		for i, l := range conf.Links {
			rc.Links[i] = &LinkConfig{Label: l.Label, URL: l.URL, Icon: l.Icon}
		}
	}
	if conf.Members != nil {
		rc.Members = make([]*MemberConfig, len(conf.Members))
		for i, m := range conf.Members {
			rc.Members[i] = &MemberConfig{
				Username:      m.Username,
				Password:      m.Password,
				HomeDirectory: m.HomeDirectory,
				Permissions:   clonePermissions(m.Permissions),
			}
		}
	}
	if conf.Docker != nil {
		rc.Docker = &DockerRemote{
			Registries: make([]*DockerRegistry, len(conf.Docker.Registries)),
		}
		for i, r := range conf.Docker.Registries {
			rc.Docker.Registries[i] = &DockerRegistry{
				Name:        r.Name,
				Description: r.Description,
				URL:         r.URL,
				Username:    r.Username,
				Password:    r.Password,
			}
		}
	}
	if conf.Server != nil {
		rc.Server = &ServerRemote{
			JWTSecret:       conf.Server.JWTSecret,
			ProxyHeaderName: conf.Server.ProxyHeaderName,
		}
	}
	return rc
}

// mergeRemote 将 RemoteConfig 合并到 Config（etcd 覆盖 YAML）
func mergeRemote(conf *Config, rc *RemoteConfig) {
	if rc.Agent != nil {
		conf.Agent = rc.Agent
	}
	if rc.Apisix != nil {
		if conf.Apisix == nil {
			conf.Apisix = &ApisixConfig{}
		}
		conf.Apisix.AdminKey = rc.Apisix.AdminKey
	}
	if rc.Marketplace != nil {
		conf.Marketplace = rc.Marketplace
	}
	if rc.Links != nil {
		conf.Links = rc.Links
	}
	if rc.Members != nil {
		conf.Members = rc.Members
	}
	if rc.Docker != nil && rc.Docker.Registries != nil {
		if conf.Docker == nil {
			conf.Docker = &DockerConfig{}
		}
		conf.Docker.Registries = rc.Docker.Registries
	}
	if rc.Server != nil {
		if conf.Server == nil {
			conf.Server = &Server{}
		}
		conf.Server.JWTSecret = rc.Server.JWTSecret
		conf.Server.ProxyHeaderName = rc.Server.ProxyHeaderName
	}
}

// validateRemote 基础校验
func validateRemote(rc *RemoteConfig) error {
	for _, m := range rc.Members {
		if m.Username == "" {
			return fmt.Errorf("invalid member: empty username")
		}
	}
	return nil
}

// remoteToBytes 将 RemoteConfig 分段序列化为 map[key]bytes
func remoteToBytes(rc *RemoteConfig) (map[string][]byte, error) {
	out := make(map[string][]byte)
	if rc.Agent != nil {
		b, err := json.Marshal(rc.Agent)
		if err != nil {
			return nil, err
		}
		out["agent"] = b
	}
	if rc.Apisix != nil {
		b, err := json.Marshal(rc.Apisix)
		if err != nil {
			return nil, err
		}
		out["apisix"] = b
	}
	if rc.Marketplace != nil {
		b, err := json.Marshal(rc.Marketplace)
		if err != nil {
			return nil, err
		}
		out["marketplace"] = b
	}
	if rc.Links != nil {
		b, err := json.Marshal(rc.Links)
		if err != nil {
			return nil, err
		}
		out["links"] = b
	}
	if rc.Members != nil {
		b, err := json.Marshal(rc.Members)
		if err != nil {
			return nil, err
		}
		out["members"] = b
	}
	if rc.Docker != nil && rc.Docker.Registries != nil {
		b, err := json.Marshal(rc.Docker)
		if err != nil {
			return nil, err
		}
		out["docker"] = b
	}
	if rc.Server != nil {
		b, err := json.Marshal(rc.Server)
		if err != nil {
			return nil, err
		}
		out["server"] = b
	}
	return out, nil
}

// bytesToRemote 从 map[key]bytes 反序列化为 RemoteConfig
func bytesToRemote(data map[string][]byte) (*RemoteConfig, error) {
	rc := &RemoteConfig{}
	if b, ok := data["agent"]; ok && len(b) > 0 {
		if err := json.Unmarshal(b, &rc.Agent); err != nil {
			return nil, fmt.Errorf("unmarshal agent: %w", err)
		}
	}
	if b, ok := data["apisix"]; ok && len(b) > 0 {
		if err := json.Unmarshal(b, &rc.Apisix); err != nil {
			return nil, fmt.Errorf("unmarshal apisix: %w", err)
		}
	}
	if b, ok := data["marketplace"]; ok && len(b) > 0 {
		if err := json.Unmarshal(b, &rc.Marketplace); err != nil {
			return nil, fmt.Errorf("unmarshal marketplace: %w", err)
		}
	}
	if b, ok := data["links"]; ok && len(b) > 0 {
		if err := json.Unmarshal(b, &rc.Links); err != nil {
			return nil, fmt.Errorf("unmarshal links: %w", err)
		}
	}
	if b, ok := data["members"]; ok && len(b) > 0 {
		if err := json.Unmarshal(b, &rc.Members); err != nil {
			return nil, fmt.Errorf("unmarshal members: %w", err)
		}
	}
	if b, ok := data["docker"]; ok && len(b) > 0 {
		if err := json.Unmarshal(b, &rc.Docker); err != nil {
			return nil, fmt.Errorf("unmarshal docker: %w", err)
		}
	}
	if b, ok := data["server"]; ok && len(b) > 0 {
		if err := json.Unmarshal(b, &rc.Server); err != nil {
			return nil, fmt.Errorf("unmarshal server: %w", err)
		}
	}
	return rc, nil
}

func clonePermissions(p map[string]string) map[string]string {
	if p == nil {
		return nil
	}
	out := make(map[string]string, len(p))
	for k, v := range p {
		out[k] = v
	}
	return out
}
