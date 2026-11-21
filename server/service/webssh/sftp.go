package webssh

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"strings"
	"sync/atomic"
	"time"

	"github.com/pkg/sftp"
	"github.com/rehiy/libgo/filer"
	"github.com/rehiy/libgo/logman"
	"github.com/rehiy/libgo/webssh"
	"golang.org/x/crypto/ssh"

	"isrvd/pkgs/helper"
)

// SFTPListResult SFTP 目录列表结果。
type SFTPListResult = webssh.ListResult

const sftpConcurrentRequests = 8

func (s *Service) SFTPList(ctx context.Context, hostID, dirPath string) (*SFTPListResult, error) {
	conn, closeConn, err := s.sftpConnect(ctx, hostID)
	if err != nil {
		return nil, err
	}
	defer closeConn()
	if dirPath == "" || dirPath == "~" {
		dirPath, err = conn.Getwd()
		if err != nil {
			dirPath = "/"
		}
	}
	entries, err := conn.ReadDirContext(ctx, dirPath)
	if err != nil {
		return nil, fmt.Errorf("读取目录失败: %w", err)
	}
	files := make([]*webssh.FileInfo, 0, len(entries))
	for _, entry := range entries {
		file := &webssh.FileInfo{
			Name: entry.Name(), Size: entry.Size(), Mode: entry.Mode().String(),
			ModTime: entry.ModTime().Unix(), IsDir: entry.IsDir(), IsLink: entry.Mode()&os.ModeSymlink != 0,
		}
		if file.IsLink {
			name := path.Join(dirPath, file.Name)
			if target, err := conn.Stat(name); err == nil {
				file.IsDir = target.IsDir()
			}
			file.LinkTarget, _ = conn.ReadLink(name)
		}
		files = append(files, file)
	}
	return &SFTPListResult{Path: dirPath, Files: files}, ctx.Err()
}

// SFTPOpen 打开可取消的远程文件流，调用方负责关闭。
func (s *Service) SFTPOpen(ctx context.Context, hostID, filePath string) (io.ReadCloser, int64, error) {
	conn, closeConn, err := s.sftpConnect(ctx, hostID)
	if err != nil {
		return nil, 0, err
	}
	file, err := conn.Open(filePath)
	if err != nil {
		closeConn()
		return nil, 0, fmt.Errorf("打开文件失败: %w", err)
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() < 0 {
		closeConn()
		if err == nil {
			err = os.ErrInvalid
		}
		return nil, 0, err
	}
	return &sftpFile{File: file, closeConn: closeConn}, info.Size(), nil
}

type sftpFile struct {
	*sftp.File
	closeConn func()
}

func (f *sftpFile) Close() error {
	f.closeConn()
	return nil
}

// SFTPUpload 上传单个文件；上传子目录时先创建远端父目录。
func (s *Service) SFTPUpload(ctx context.Context, hostID, dirPath string, file io.Reader, relativePath string, size int64) error {
	return s.withSFTP(ctx, hostID, func(conn *sftp.Client) error {
		dest := path.Join(dirPath, relativePath)
		if dir := path.Dir(dest); dir != "/" && dir != "." && dir != dirPath {
			if err := conn.MkdirAll(dir); err != nil {
				return err
			}
		}
		return s.sftpWrite(ctx, hostID, conn, dest, file, size)
	})
}

func (s *Service) SFTPRemove(ctx context.Context, hostID, targetPath string, recursive bool) error {
	return s.withSFTP(ctx, hostID, func(conn *sftp.Client) error {
		if recursive {
			return conn.RemoveAll(targetPath)
		}
		return conn.Remove(targetPath)
	})
}

func (s *Service) SFTPMkdir(ctx context.Context, hostID, dirPath string) error {
	return s.withSFTP(ctx, hostID, func(conn *sftp.Client) error { return conn.MkdirAll(dirPath) })
}

func (s *Service) SFTPRename(ctx context.Context, hostID, oldPath, newPath string) error {
	return s.withSFTP(ctx, hostID, func(conn *sftp.Client) error { return conn.Rename(oldPath, newPath) })
}

func (s *Service) SFTPChmod(ctx context.Context, hostID, targetPath string, mode os.FileMode) error {
	return s.withSFTP(ctx, hostID, func(conn *sftp.Client) error { return conn.Chmod(targetPath, mode) })
}

func (s *Service) SFTPChown(ctx context.Context, hostID, targetPath string, uid, gid int) error {
	return s.withSFTP(ctx, hostID, func(conn *sftp.Client) error { return conn.Chown(targetPath, uid, gid) })
}

func (s *Service) SFTPRead(ctx context.Context, hostID, filePath string) (string, error) {
	file, size, err := s.SFTPOpen(ctx, hostID, filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	if size > helper.MaxEditableFileBytes {
		return "", helper.ErrTooLarge
	}
	buf := &helper.EditBuffer{}
	_, err = io.Copy(buf, file)
	if buf.Exceeded() {
		return "", helper.ErrTooLarge
	}
	if err != nil {
		return "", err
	}
	if !filer.IsTextBytes(buf.Bytes()) {
		return "", helper.ErrBinary
	}
	return strings.ToValidUTF8(buf.String(), "\uFFFD"), nil
}

func (s *Service) SFTPWrite(ctx context.Context, hostID, filePath, content string) error {
	return s.withSFTP(ctx, hostID, func(conn *sftp.Client) error {
		return s.sftpWrite(ctx, hostID, conn, filePath, strings.NewReader(content), int64(len(content)))
	})
}

func (s *Service) SFTPDirSize(ctx context.Context, hostID, dirPath string) (int64, error) {
	var size int64
	err := s.withSFTP(ctx, hostID, func(conn *sftp.Client) error {
		walker := conn.Walk(dirPath)
		for walker.Step() {
			if err := walker.Err(); err != nil {
				return err
			}
			info := walker.Stat()
			if info != nil && !info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
				size += info.Size()
			}
		}
		return nil
	})
	return size, err
}

// ─── 辅助函数 ───

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

func (s *Service) sftpWrite(ctx context.Context, hostID string, conn *sftp.Client, name string, source io.Reader, size int64) error {
	existing, err := conn.Lstat(name)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if existing != nil && existing.Mode()&os.ModeSymlink != 0 {
		name, err = conn.RealPath(name)
		if err != nil {
			return err
		}
		existing, err = conn.Stat(name)
		if err != nil {
			return err
		}
	}
	_, replace := conn.HasExtension("posix-rename@openssh.com")
	if existing != nil {
		if !existing.Mode().IsRegular() {
			return os.ErrInvalid
		}
		if !replace {
			return errors.New("SFTP 服务器不支持安全覆盖已有文件")
		}
		target, err := conn.OpenFile(name, os.O_WRONLY)
		if err != nil {
			return err
		}
		if err := target.Close(); err != nil {
			return err
		}
	}

	staged := path.Join(path.Dir(name), ".isrvd-upload-"+rand.Text())
	file, err := conn.OpenFile(staged, os.O_WRONLY|os.O_CREATE|os.O_EXCL)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			s.removeSFTPTemp(ctx, hostID, conn, staged)
		}
	}()
	info, err := conn.Stat(staged)
	if err == nil {
		err = conn.Chmod(staged, 0600)
	}
	var written int64
	if err == nil {
		if size > 32<<10 {
			written, err = file.ReadFromWithConcurrency(source, sftpConcurrentRequests)
		} else {
			written, err = file.ReadFrom(source)
		}
	}
	if err := errors.Join(err, file.Close()); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}
	if written != size {
		return fmt.Errorf("写入文件失败: %w", io.ErrUnexpectedEOF)
	}
	mode := info.Mode()
	if existing != nil {
		mode = existing.Mode()
		oldOwner, oldOK := existing.Sys().(*sftp.FileStat)
		newOwner, newOK := info.Sys().(*sftp.FileStat)
		if oldOK && newOK && (oldOwner.UID != newOwner.UID || oldOwner.GID != newOwner.GID) {
			if err := conn.Chown(staged, int(oldOwner.UID), int(oldOwner.GID)); err != nil {
				return err
			}
		}
	}
	if err := conn.Chmod(staged, mode); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if existing != nil {
		err = conn.PosixRename(staged, name)
	} else {
		err = conn.Rename(staged, name)
	}
	if err != nil {
		return fmt.Errorf("重命名失败: %w", err)
	}
	committed = true
	return nil
}

func (s *Service) removeSFTPTemp(ctx context.Context, hostID string, conn *sftp.Client, staged string) {
	if ctx.Err() == nil {
		if err := conn.Remove(staged); err == nil || errors.Is(err, os.ErrNotExist) {
			return
		}
	}
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()
	cleanup, closeConn, err := s.sftpConnect(cleanupCtx, hostID)
	if err == nil {
		defer closeConn()
		err = cleanup.Remove(staged)
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		logman.Warn("SFTP temporary file cleanup failed", "host", hostID, "path", staged, "error", err)
	}
}
