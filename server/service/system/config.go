// Package system 系统配置查询与修改
package system

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"isrvd/server/config"
)

// ErrInvalidNotifyConfig 表示提交的应用故障阈值无效。
var ErrInvalidNotifyConfig = errors.New("故障告警配置无效")

// AllConfig 全部配置聚合（请求/响应共用）。
// 作 GET 响应时：敏感字段已脱敏；作 PUT 请求时：nil 分区跳过更新，密钥为空保留原值。
type AllConfig struct {
	Server      *config.ServerConfig      `json:"server"`      // 服务配置（JWTSecret：响应脱敏 / 请求空保留）
	Password    *config.PasswordConfig    `json:"password"`    // 密码登录配置
	Passkey     *config.PasskeyConfig     `json:"passkey"`     // Passkey 认证配置
	OIDC        *config.OIDCConfig        `json:"oidc"`        // OIDC 配置（ClientSecret：响应脱敏 / 请求空保留）
	THA         *config.THAConfig         `json:"tha"`         // 代理 Header 认证配置
	Copilot     *config.CopilotConfig     `json:"copilot"`     // Copilot LLM 配置（APIKey：响应脱敏 / 请求空保留）
	Notify      *config.NotifyConfig      `json:"notify"`      // 告警通知配置
	Apisix      *config.ApisixConfig      `json:"apisix"`      // APISIX 配置（AdminKey：响应脱敏 / 请求空保留）
	Caddy       *config.CaddyConfig       `json:"caddy"`       // Caddy 配置
	Docker      *config.DockerConfig      `json:"docker"`      // Docker 配置（registry.Password：响应脱敏 / 请求空保留）
	Monitor     *config.MonitorConfig     `json:"monitor"`     // 监控配置
	Marketplace *config.MarketplaceConfig `json:"marketplace"` // 应用市场配置
	Links       []*config.LinkConfig      `json:"links"`       // 导航链接列表
}

// ConfigService 系统配置业务服务
type ConfigService struct{}

// NewConfigService 创建系统配置业务服务
func NewConfigService() *ConfigService {
	return &ConfigService{}
}

// ConfigAll 获取全部配置：深拷贝隔离当前快照后清空敏感字段
func (s *ConfigService) ConfigAll() *AllConfig {
	snapshot := config.Current()
	src := &AllConfig{
		Server:      snapshot.Server,
		Password:    snapshot.Password,
		Passkey:     snapshot.Passkey,
		OIDC:        snapshot.OIDC,
		THA:         snapshot.THA,
		Copilot:     snapshot.Copilot,
		Notify:      snapshot.Notify,
		Apisix:      snapshot.Apisix,
		Caddy:       snapshot.Caddy,
		Docker:      snapshot.Docker,
		Monitor:     snapshot.Monitor,
		Marketplace: snapshot.Marketplace,
		Links:       snapshot.Links,
	}
	dst, err := deepCopyJSON(src)
	if err != nil || dst == nil {
		return &AllConfig{}
	}
	if dst.Server != nil {
		dst.Server.JWTSecret = ""
	}
	if dst.OIDC != nil {
		dst.OIDC.ClientSecret = ""
	}
	if dst.Copilot != nil {
		dst.Copilot.APIKey = ""
	}
	if dst.Apisix != nil {
		dst.Apisix.AdminKey = ""
	}
	if dst.Docker != nil {
		for _, registry := range dst.Docker.Registries {
			if registry != nil {
				registry.Password = ""
			}
		}
	}
	return dst
}

// ConfigUpdate 一次性更新全部配置（任何 nil 分区将跳过）。
func (s *ConfigService) ConfigUpdate(req AllConfig) error {
	if req.Notify != nil && req.Notify.Events != nil {
		events := req.Notify.Events
		if events.RestartThreshold < 0 || events.RestartThreshold > 10000 ||
			events.RestartWindow < 0 || (events.RestartWindow > 0 && events.RestartWindow < 30) || events.RestartWindow > 86400 ||
			events.CertificateDays < 0 || events.CertificateDays > 3650 {
			return fmt.Errorf("%w：重启次数为 1–10000，窗口为 30–86400 秒，证书提前天数为 1–3650；0 使用默认值", ErrInvalidNotifyConfig)
		}
	}

	if err := config.UpdateStored(func(draft *config.Snapshot) error {
		oldRoot := draft.Server.RootDirectory
		newRoot := oldRoot
		if req.Server != nil {
			req.Server.JWTSecret = pickSecret(req.Server.JWTSecret, draft.Server.JWTSecret)
			req.Server = config.ServerNormalize(req.Server)
			newRoot = req.Server.RootDirectory
			if oldRoot != newRoot {
				if req.Docker == nil && draft.Docker != nil {
					draft.Docker.ContainerRoot = config.PathToAbs(
						config.PathToRel(draft.Docker.ContainerRoot, oldRoot), newRoot,
					)
				}
				for _, member := range draft.Members {
					if member != nil {
						member.HomeDirectory = config.PathToAbs(
							config.PathToRel(member.HomeDirectory, oldRoot), newRoot,
						)
					}
				}
			}
			draft.Server = req.Server
		}
		if req.Password != nil {
			draft.Password = req.Password
		}
		if req.Passkey != nil {
			draft.Passkey = req.Passkey
		}
		if req.OIDC != nil {
			req.OIDC.ClientSecret = pickSecret(req.OIDC.ClientSecret, draft.OIDC.ClientSecret)
			draft.OIDC = req.OIDC
		}
		if req.THA != nil {
			draft.THA = req.THA
		}
		if req.Copilot != nil {
			req.Copilot.APIKey = pickSecret(req.Copilot.APIKey, draft.Copilot.APIKey)
			draft.Copilot = req.Copilot
		}
		if req.Notify != nil {
			draft.Notify = req.Notify
		}
		if req.Apisix != nil {
			req.Apisix.AdminKey = pickSecret(req.Apisix.AdminKey, draft.Apisix.AdminKey)
			draft.Apisix = req.Apisix
		}
		if req.Caddy != nil {
			draft.Caddy = req.Caddy
		}
		if req.Docker != nil {
			if oldRoot != newRoot && draft.Docker != nil &&
				req.Docker.ContainerRoot == draft.Docker.ContainerRoot {
				req.Docker.ContainerRoot = config.PathToAbs(
					config.PathToRel(req.Docker.ContainerRoot, oldRoot), newRoot,
				)
			}
			for _, registry := range req.Docker.Registries {
				if registry == nil || registry.Password != "" {
					continue
				}
				for _, old := range draft.Docker.Registries {
					if old != nil && old.URL == registry.URL && old.Username == registry.Username {
						registry.Password = old.Password
						break
					}
				}
			}
			draft.Docker = req.Docker
		}
		if req.Monitor != nil {
			draft.Monitor = req.Monitor
		}
		if req.Marketplace != nil {
			draft.Marketplace = req.Marketplace
		}
		if req.Links != nil {
			draft.Links = req.Links
		}
		return nil
	}); err != nil {
		return err
	}

	select {
	case config.ReloadCh <- struct{}{}:
	default:
	}
	return nil
}

// ─── 辅助函数 ───

// deepCopyJSON 通过 JSON 序列化-反序列化深拷贝，结果与源对象无共享指针
func deepCopyJSON[T any](src T) (T, error) {
	var dst T
	data, err := json.Marshal(src)
	if err != nil {
		return dst, err
	}
	err = json.Unmarshal(data, &dst)
	return dst, err
}

// pickSecret 新值为空（含纯空白）时保留原值，否则用裁剪首尾空白后的新值
func pickSecret(newVal, oldVal string) string {
	newVal = strings.TrimSpace(newVal)
	if newVal == "" {
		return oldVal
	}
	return newVal
}
