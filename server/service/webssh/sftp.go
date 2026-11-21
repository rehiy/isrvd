package webssh

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"github.com/rehiy/libgo/filer"
	"github.com/rehiy/libgo/webssh"
	"golang.org/x/crypto/ssh"

	"isrvd/pkgs/helper"
)

// SFTPListResult SFTP 目录列表结果。
type SFTPListResult = webssh.ListResult

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
	if err != nil || !info.Mode().IsRegular() {
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
func (s *Service) SFTPUpload(ctx context.Context, hostID, dirPath string, file io.Reader, relativePath string) error {
	return s.withSFTP(ctx, hostID, func(conn *sftp.Client) error {
		dest := path.Join(dirPath, relativePath)
		if dir := path.Dir(dest); dir != "/" && dir != "." && dir != dirPath {
			if err := conn.MkdirAll(dir); err != nil {
				return err
			}
		}
		return sftpWrite(conn, dest, file)
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
		return sftpWrite(conn, filePath, strings.NewReader(content))
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
	sshConn, channels, requests, err := ssh.NewClientConn(conn, addr, &ssh.ClientConfig{
		User: opt.User, Auth: []ssh.AuthMethod{auth}, HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
	if err != nil {
		return nil, nil, err
	}
	client, err := sftp.NewClient(ssh.NewClient(sshConn, channels, requests))
	if err == nil {
		err = conn.SetDeadline(time.Time{})
	}
	if err != nil {
		return nil, nil, err
	}
	return client, closeConn, nil
}

func sftpWrite(conn *sftp.Client, name string, source io.Reader) error {
	file, err := conn.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	_, err = io.Copy(file, source)
	if err := errors.Join(err, file.Close()); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}
	return nil
}
