package webssh

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path"

	"github.com/pkg/sftp"
	libwebssh "github.com/rehiy/libgo/webssh"
)

// SFTPFileInfo SFTP 文件/目录信息
type SFTPFileInfo struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	Mode    string `json:"mode"`
	ModTime int64  `json:"modTime"`
	IsDir   bool   `json:"isDir"`
}

// SFTPListResult 目录列表结果（含实际路径）
type SFTPListResult struct {
	Path  string          `json:"path"`
	Files []*SFTPFileInfo `json:"files"`
}

// newSFTPClient 根据主机 ID 建立 SFTP 连接
func (s *Service) newSFTPClient(hostID string) (*sftp.Client, error) {
	host, err := s.store.hostGetOption(hostID)
	if err != nil {
		return nil, err
	}

	opt := &libwebssh.SSHClientOption{
		Addr:       host.Addr,
		User:       host.User,
		Password:   host.Password,
		PrivateKey: host.PrivateKey,
	}

	sshClient, err := libwebssh.NewSSHClient(opt)
	if err != nil {
		return nil, fmt.Errorf("SSH 连接失败: %w", err)
	}

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		sshClient.Close()
		return nil, fmt.Errorf("SFTP 初始化失败: %w", err)
	}

	return sftpClient, nil
}

// SFTPList 列出目录内容，返回实际路径和文件列表
func (s *Service) SFTPList(hostID, dirPath string) (*SFTPListResult, error) {
	client, err := s.newSFTPClient(hostID)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	// 未指定路径时使用远程用户的 home 目录
	if dirPath == "" || dirPath == "~" {
		if wd, err := client.Getwd(); err == nil {
			dirPath = wd
		} else {
			dirPath = "/"
		}
	}

	entries, err := client.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("读取目录失败: %w", err)
	}

	files := make([]*SFTPFileInfo, 0, len(entries))
	for _, e := range entries {
		files = append(files, &SFTPFileInfo{
			Name:    e.Name(),
			Size:    e.Size(),
			Mode:    e.Mode().String(),
			ModTime: e.ModTime().Unix(),
			IsDir:   e.IsDir(),
		})
	}
	return &SFTPListResult{Path: dirPath, Files: files}, nil
}

// SFTPDownload 下载文件，返回文件内容 Reader 和文件大小
func (s *Service) SFTPDownload(hostID, filePath string) (io.ReadCloser, int64, error) {
	client, err := s.newSFTPClient(hostID)
	if err != nil {
		return nil, 0, err
	}

	stat, err := client.Stat(filePath)
	if err != nil {
		client.Close()
		return nil, 0, fmt.Errorf("文件不存在: %w", err)
	}
	if stat.IsDir() {
		client.Close()
		return nil, 0, fmt.Errorf("不能下载目录")
	}

	f, err := client.Open(filePath)
	if err != nil {
		client.Close()
		return nil, 0, fmt.Errorf("打开文件失败: %w", err)
	}

	// 包装 closer，关闭文件时同时关闭 sftp client
	return &sftpReadCloser{ReadCloser: f, client: client}, stat.Size(), nil
}

// sftpReadCloser 包装 sftp.File，关闭时同时关闭 sftp.Client
type sftpReadCloser struct {
	io.ReadCloser
	client *sftp.Client
}

func (r *sftpReadCloser) Close() error {
	err := r.ReadCloser.Close()
	r.client.Close()
	return err
}

// SFTPUpload 上传文件
func (s *Service) SFTPUpload(hostID, dirPath string, file multipart.File, filename string) error {
	client, err := s.newSFTPClient(hostID)
	if err != nil {
		return err
	}
	defer client.Close()

	destPath := path.Join(dirPath, filename)
	f, err := client.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, file); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}
	return nil
}

// SFTPRemove 删除文件或目录（目录需为空）
func (s *Service) SFTPRemove(hostID, targetPath string) error {
	client, err := s.newSFTPClient(hostID)
	if err != nil {
		return err
	}
	defer client.Close()

	stat, err := client.Stat(targetPath)
	if err != nil {
		return fmt.Errorf("路径不存在: %w", err)
	}

	if stat.IsDir() {
		if err := client.RemoveDirectory(targetPath); err != nil {
			return fmt.Errorf("删除目录失败: %w", err)
		}
	} else {
		if err := client.Remove(targetPath); err != nil {
			return fmt.Errorf("删除文件失败: %w", err)
		}
	}
	return nil
}

// SFTPMkdir 创建目录
func (s *Service) SFTPMkdir(hostID, dirPath string) error {
	client, err := s.newSFTPClient(hostID)
	if err != nil {
		return err
	}
	defer client.Close()

	if err := client.MkdirAll(dirPath); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}
	return nil
}

// SFTPRename 重命名/移动文件或目录
func (s *Service) SFTPRename(hostID, oldPath, newPath string) error {
	client, err := s.newSFTPClient(hostID)
	if err != nil {
		return err
	}
	defer client.Close()

	if err := client.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("重命名失败: %w", err)
	}
	return nil
}
