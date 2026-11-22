package webssh

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/pkg/sftp"
	"github.com/rehiy/libgo/filer"
	"github.com/rehiy/libgo/webssh"

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
		return s.sftpWrite(ctx, hostID, conn, dest, file, size, false)
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
	return s.writeSFTPText(ctx, hostID, filePath, content, false)
}

func (s *Service) SFTPCreateOnly(ctx context.Context, hostID, filePath, content string) error {
	return s.writeSFTPText(ctx, hostID, filePath, content, true)
}

func (s *Service) writeSFTPText(ctx context.Context, hostID, filePath, content string, createOnly bool) error {
	return s.withSFTP(ctx, hostID, func(conn *sftp.Client) error {
		return s.sftpWrite(ctx, hostID, conn, filePath, strings.NewReader(content), int64(len(content)), createOnly)
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
