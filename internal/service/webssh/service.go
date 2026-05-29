// Package webssh 提供 WebSSH 主机管理和 SSH 终端会话业务服务
package webssh

import (
	"fmt"
	"path/filepath"

	"github.com/rehiy/libgo/logman"
	"github.com/rehiy/libgo/websocket"
	libwebssh "github.com/rehiy/libgo/webssh"

	"isrvd/config"
)

// logger 为 webssh 包创建带名称的 logger
var logger = logman.Named("webssh")

// Service WebSSH 业务服务
type Service struct {
	store    *store
	sftpPool *sftpPool
}

// NewService 创建 WebSSH 业务服务
func NewService() (*Service, error) {
	storePath := filepath.Join(config.Server.RootDirectory, "webssh.yml")
	s, err := newStore(storePath)
	if err != nil {
		return nil, fmt.Errorf("初始化 WebSSH 存储失败: %w", err)
	}
	return &Service{store: s, sftpPool: newSFTPPool()}, nil
}

// Close 释放 Service 持有的所有资源（连接池等），应在应用退出时调用
func (s *Service) Close() {
	s.sftpPool.close()
}

// HostList 列出所有主机（密码不回显）
func (s *Service) HostList() []*HostView {
	return s.store.hostList()
}

// HostInspect 查看指定主机详情（密码不回显）
func (s *Service) HostInspect(id string) *HostView {
	return s.store.hostInspect(id)
}

// HostCreate 新建主机配置
func (s *Service) HostCreate(req *HostUpsertRequest) (*HostView, error) {
	h := &Host{
		Name:        req.Name,
		Addr:        req.Addr,
		User:        req.User,
		Password:    req.Password,
		PrivateKey:  req.PrivateKey,
		Description: req.Description,
	}
	if err := s.store.hostCreate(h); err != nil {
		return nil, fmt.Errorf("创建主机失败: %w", err)
	}
	logger.Info("WebSSH 主机已创建", "id", h.ID, "name", h.Name, "addr", h.Addr)
	return s.store.hostInspect(h.ID), nil
}

// HostUpdate 更新主机配置
func (s *Service) HostUpdate(id string, req *HostUpsertRequest) (*HostView, error) {
	h := &Host{
		Name:        req.Name,
		Addr:        req.Addr,
		User:        req.User,
		Password:    req.Password,
		PrivateKey:  req.PrivateKey,
		Description: req.Description,
	}
	if err := s.store.hostUpdate(id, h); err != nil {
		return nil, fmt.Errorf("更新主机失败: %w", err)
	}
	logger.Info("WebSSH 主机已更新", "id", id, "name", req.Name)
	return s.store.hostInspect(id), nil
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
	host, err := s.store.hostGetOption(hostID)
	if err != nil {
		conn.Die("[错误: " + err.Error() + "]\r\n")
		return
	}

	opt := &libwebssh.SSHClientOption{
		Addr:       host.Addr,
		User:       host.User,
		Password:   host.Password,
		PrivateKey: host.PrivateKey,
	}

	logger.Info("WebSSH 会话开始", "host", host.Name, "addr", host.Addr, "user", host.User)

	if err := libwebssh.Connect(conn.Conn, opt); err != nil {
		logger.Error("WebSSH 会话结束", "host", host.Name, "error", err)
	} else {
		logger.Info("WebSSH 会话正常结束", "host", host.Name)
	}
}

// HostUpsertRequest 主机新建/更新请求
type HostUpsertRequest struct {
	Name        string `json:"name" binding:"required"`
	Addr        string `json:"addr" binding:"required"`
	User        string `json:"user" binding:"required"`
	Password    string `json:"password"`
	PrivateKey  string `json:"privateKey"`
	Description string `json:"description"`
}
