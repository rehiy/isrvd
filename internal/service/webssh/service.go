// Package webssh 提供 WebSSH 主机管理和 SSH 终端会话业务服务
package webssh

import (
	"fmt"

	"github.com/rehiy/libgo/logman"
	"github.com/rehiy/libgo/websocket"
	libwebssh "github.com/rehiy/libgo/webssh"
)

// logger 为 webssh 包创建带名称的 logger
var logger = logman.Named("webssh")

// Service WebSSH 业务服务
type Service struct {
	store           *store
	credentialStore *credentialStore
	sftpClient      *libwebssh.SFTPClient
}

// NewService 创建 WebSSH 业务服务
func NewService() (*Service, error) {
	s, err := newHostStore()
	if err != nil {
		return nil, fmt.Errorf("初始化 WebSSH 存储失败: %w", err)
	}
	cs, err := newCredentialStore()
	if err != nil {
		return nil, fmt.Errorf("初始化凭据存储失败: %w", err)
	}
	return &Service{store: s, credentialStore: cs, sftpClient: libwebssh.NewSFTPClient(0)}, nil
}

// Close 释放 Service 持有的所有资源（连接池等），应在应用退出时调用
func (s *Service) Close() {
	s.sftpClient.Close()
}

// ─── Credential 凭据管理 ───

// CredentialList 列出所有凭据（密码/私钥不回显）
func (s *Service) CredentialList() []*CredentialView {
	return s.credentialStore.list()
}

// CredentialInspect 查看指定凭据详情（含敏感信息，用于连接）
func (s *Service) CredentialInspect(id string) *Credential {
	return s.credentialStore.get(id)
}

// CredentialCreate 新建凭据
func (s *Service) CredentialCreate(req *CredentialUpsertRequest) (*CredentialView, error) {
	c := &Credential{
		Name:        req.Name,
		Description: req.Description,
		User:        req.User,
		Password:    req.Password,
		PrivateKey:  req.PrivateKey,
	}
	if err := s.credentialStore.create(c); err != nil {
		return nil, fmt.Errorf("创建凭据失败: %w", err)
	}
	logger.Info("SSH 凭据已创建", "id", c.ID, "name", c.Name, "user", c.User)
	return c.toView(), nil
}

// CredentialUpdate 更新凭据
func (s *Service) CredentialUpdate(id string, req *CredentialUpsertRequest) (*CredentialView, error) {
	c := &Credential{
		Name:        req.Name,
		Description: req.Description,
		User:        req.User,
		Password:    req.Password,
		PrivateKey:  req.PrivateKey,
	}
	if err := s.credentialStore.update(id, c); err != nil {
		return nil, fmt.Errorf("更新凭据失败: %w", err)
	}
	logger.Info("SSH 凭据已更新", "id", id, "name", req.Name)
	return s.credentialStore.get(id).toView(), nil
}

// CredentialDelete 删除凭据
func (s *Service) CredentialDelete(id string) error {
	if err := s.credentialStore.delete(id); err != nil {
		return fmt.Errorf("删除凭据失败: %w", err)
	}
	logger.Info("SSH 凭据已删除", "id", id)
	return nil
}

// ─── Host 主机管理 ───

// HostList 列出所有主机（密码不回显，附凭据名称）
func (s *Service) HostList() []*HostView {
	hosts := s.store.hostList()
	// 填充 credentialName
	credMap := make(map[string]string)
	for _, c := range s.credentialStore.list() {
		credMap[c.ID] = c.Name
	}
	for _, h := range hosts {
		if h.CredentialID != "" {
			h.CredentialName = credMap[h.CredentialID]
		}
	}
	return hosts
}

// HostInspect 查看指定主机详情（密码不回显）
func (s *Service) HostInspect(id string) *HostView {
	h := s.store.hostInspect(id)
	if h != nil && h.CredentialID != "" {
		c := s.credentialStore.get(h.CredentialID)
		if c != nil {
			h.CredentialName = c.Name
		}
	}
	return h
}

// HostCreate 新建主机配置
func (s *Service) HostCreate(req *HostUpsertRequest) (*HostView, error) {
	h := &Host{
		Name:         req.Name,
		Addr:         req.Addr,
		CredentialID: req.CredentialID,
		Password:     req.Password,
		PrivateKey:   req.PrivateKey,
		Description:  req.Description,
	}
	// 如果指定了凭据，仅校验凭据存在；连接时再解析凭据内容，避免在主机配置中冗余保存敏感信息
	if req.CredentialID != "" {
		c := s.credentialStore.get(req.CredentialID)
		if c == nil {
			return nil, fmt.Errorf("凭据 %s 不存在", req.CredentialID)
		}
		h.User = c.User
		h.Password = ""
		h.PrivateKey = ""
	} else {
		h.User = req.User
	}
	if err := s.store.hostCreate(h); err != nil {
		return nil, fmt.Errorf("创建主机失败: %w", err)
	}
	logger.Info("WebSSH 主机已创建", "id", h.ID, "name", h.Name, "addr", h.Addr)
	return s.HostInspect(h.ID), nil
}

// HostUpdate 更新主机配置
func (s *Service) HostUpdate(id string, req *HostUpsertRequest) (*HostView, error) {
	// 先获取现有主机
	old := s.store.hostInspectRaw(id)
	if old == nil {
		return nil, fmt.Errorf("主机 %s 不存在", id)
	}

	h := &Host{
		Name:         req.Name,
		Addr:         req.Addr,
		CredentialID: req.CredentialID,
		Description:  req.Description,
	}

	// 凭据模式 vs 独立认证模式
	if req.CredentialID != "" {
		c := s.credentialStore.get(req.CredentialID)
		if c == nil {
			return nil, fmt.Errorf("凭据 %s 不存在", req.CredentialID)
		}
		h.User = c.User
		h.Password = ""
		h.PrivateKey = ""
	} else {
		h.User = req.User
		// 密码/私钥为空时保留原值
		if req.Password == "" {
			h.Password = old.Password
		} else {
			h.Password = req.Password
		}
		if req.PrivateKey == "" {
			h.PrivateKey = old.PrivateKey
		} else {
			h.PrivateKey = req.PrivateKey
		}
	}

	if err := s.store.hostUpdate(id, h); err != nil {
		return nil, fmt.Errorf("更新主机失败: %w", err)
	}
	logger.Info("WebSSH 主机已更新", "id", id, "name", req.Name)
	return s.HostInspect(id), nil
}

// HostDelete 删除主机配置
func (s *Service) HostDelete(id string) error {
	if err := s.store.hostDelete(id); err != nil {
		return fmt.Errorf("删除主机失败: %w", err)
	}
	logger.Info("WebSSH 主机已删除", "id", id)
	return nil
}

// RunTerminal 建立到指定主机的 SSH 终端会话并与 WebSocket 连接桥接
func (s *Service) RunTerminal(conn *websocket.ServerConn, hostID string) {
	opt, err := s.store.hostGetOption(hostID, s.credentialStore)
	if err != nil {
		conn.Die("[错误: " + err.Error() + "]\r\n")
		return
	}

	logger.Info("WebSSH 会话开始", "hostID", hostID, "addr", opt.Addr, "user", opt.User)

	if err := libwebssh.Connect(conn.Conn, opt); err != nil {
		logger.Error("WebSSH 会话结束", "hostID", hostID, "error", err)
	} else {
		logger.Info("WebSSH 会话正常结束", "hostID", hostID)
	}
}

// ─── 请求结构体 ───

// HostUpsertRequest 主机新建/更新请求
type HostUpsertRequest struct {
	Name         string `json:"name" binding:"required"`
	Addr         string `json:"addr" binding:"required"`
	CredentialID string `json:"credentialId"`
	User         string `json:"user"`
	Password     string `json:"password"`
	PrivateKey   string `json:"privateKey"`
	Description  string `json:"description"`
}

// CredentialUpsertRequest 凭据新建/更新请求
type CredentialUpsertRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	User        string `json:"user" binding:"required"`
	Password    string `json:"password"`
	PrivateKey  string `json:"privateKey"`
}
