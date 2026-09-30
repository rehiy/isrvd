package webssh

import (
	"fmt"

	"github.com/rehiy/libgo/webssh"
)

// Host SSH 主机配置
type Host struct {
	ID             string `yaml:"id" json:"id"`                                         // 主机 ID（自动生成）
	Name           string `yaml:"name" json:"name"`                                     // 主机名称
	Addr           string `yaml:"addr" json:"addr"`                                     // 主机地址（host:port）
	CredentialID   string `yaml:"credentialId,omitempty" json:"credentialId,omitempty"` // 引用的凭据 ID
	CredentialName string `yaml:"-" json:"credentialName,omitempty"`                    // 凭据名称（只读，展示用）
	User           string `yaml:"user" json:"user"`                                     // 用户名（独立认证模式）
	Password       string `yaml:"password,omitempty" json:"-"`                          // 密码（仅独立认证模式，不序列化到 JSON）
	PrivateKey     string `yaml:"privateKey,omitempty" json:"-"`                        // 私钥（仅独立认证模式，不序列化到 JSON）
	Description    string `yaml:"description" json:"description"`                       // 主机描述
}

func (h *Host) fields() (*string, *string, *string) { return &h.ID, &h.Password, &h.PrivateKey }

// hostStore 主机配置存储
type hostStore = itemStore[Host, *Host]

// hostPrepare 独立认证模式下处理密码/私钥（绑定凭据时不保存认证信息）
func hostPrepare(item, old *Host) {
	if item.CredentialID == "" {
		keepSecrets(item, old)
	}
}

// hostOption 获取指定主机的 SSH 连接配置；绑定凭据时使用凭据中的认证信息
func (s *Service) hostOption(id string) (*webssh.SSHClientOption, error) {
	h := s.hostStore.get(id)
	if h == nil {
		return nil, fmt.Errorf("主机 %s 不存在", id)
	}
	opt := &webssh.SSHClientOption{Addr: h.Addr, User: h.User, Password: h.Password, PrivateKey: h.PrivateKey}
	if h.CredentialID != "" {
		if c := s.credentialStore.get(h.CredentialID); c != nil {
			opt.User, opt.Password, opt.PrivateKey = c.User, c.Password, c.PrivateKey
		}
	}
	return opt, nil
}
