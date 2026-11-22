package webssh

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

const sftpConcurrentRequests = 8

func (s *Service) withSFTP(ctx context.Context, hostID string, run func(*sftp.Client) error) error {
	conn, closeConn, err := s.sftpConnect(ctx, hostID)
	if err != nil {
		return err
	}
	defer closeConn()
	err = run(conn)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}

// sftpConnect 为每次操作建立独立连接，取消时只关闭该请求的传输。
func (s *Service) sftpConnect(ctx context.Context, hostID string) (*sftp.Client, func(), error) {
	opt, err := s.hostOption(hostID)
	if err != nil {
		return nil, nil, err
	}
	var auth ssh.AuthMethod
	switch {
	case opt.Password != "":
		auth = ssh.Password(opt.Password)
	case opt.PrivateKey != "":
		key, err := ssh.ParsePrivateKey([]byte(opt.PrivateKey))
		if err != nil {
			return nil, nil, err
		}
		auth = ssh.PublicKeys(key)
	default:
		return nil, nil, errors.New("SSHClient: no Password or PrivateKey")
	}
	addr := opt.Addr
	if _, _, err := net.SplitHostPort(addr); err != nil {
		addr = net.JoinHostPort(strings.Trim(addr, "[]"), "22")
	}
	const connectTimeout = 5 * time.Second
	conn, err := (&net.Dialer{Timeout: connectTimeout}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, nil, err
	}
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	closeConn := func() {
		stop()
		_ = conn.Close()
	}
	defer func() {
		if err != nil {
			closeConn()
		}
	}()
	if err = conn.SetDeadline(time.Now().Add(connectTimeout)); err != nil {
		return nil, nil, err
	}
	transport := &sftpTransport{Conn: conn, idleTimeout: 30 * time.Second}
	sshConn, channels, requests, err := ssh.NewClientConn(transport, addr, &ssh.ClientConfig{
		User: opt.User, Auth: []ssh.AuthMethod{auth}, HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
	if err != nil {
		return nil, nil, err
	}
	client, err := sftp.NewClient(ssh.NewClient(sshConn, channels, requests),
		sftp.MaxConcurrentRequestsPerFile(sftpConcurrentRequests), sftp.UseFstat(true))
	if err == nil {
		err = transport.startIdleTimeout()
	}
	if err != nil {
		return nil, nil, err
	}
	return client, closeConn, nil
}

// sftpTransport 将握手期限切换为传输空闲期限。
type sftpTransport struct {
	net.Conn
	idleTimeout time.Duration
	ready       atomic.Bool
}

func (c *sftpTransport) startIdleTimeout() error {
	c.ready.Store(true)
	return c.SetDeadline(time.Now().Add(c.idleTimeout))
}

func (c *sftpTransport) Read(p []byte) (int, error) {
	if c.ready.Load() {
		if err := c.SetReadDeadline(time.Now().Add(c.idleTimeout)); err != nil {
			return 0, err
		}
	}
	return c.Conn.Read(p)
}

func (c *sftpTransport) Write(p []byte) (int, error) {
	if c.ready.Load() {
		if err := c.SetWriteDeadline(time.Now().Add(c.idleTimeout)); err != nil {
			return 0, err
		}
	}
	return c.Conn.Write(p)
}
