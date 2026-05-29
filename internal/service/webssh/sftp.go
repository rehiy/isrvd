package webssh

import (
	"io"
	"mime/multipart"
	"path"

	libwebssh "github.com/rehiy/libgo/webssh"
)

// SFTPFileInfo SFTP 文件/目录信息（透传 pkgs/webssh.FileInfo）
type SFTPFileInfo = libwebssh.FileInfo

// SFTPListResult 目录列表结果（透传 pkgs/webssh.ListResult）
type SFTPListResult = libwebssh.ListResult

// SFTPList 列出目录内容，返回实际路径和文件列表
func (s *Service) SFTPList(hostID, dirPath string) (*SFTPListResult, error) {
	opt, err := s.store.hostGetOption(hostID)
	if err != nil {
		return nil, err
	}
	return s.sftpClient.List(opt, dirPath)
}

// SFTPDownload 下载文件，返回文件内容 Reader 和文件大小
func (s *Service) SFTPDownload(hostID, filePath string) (io.ReadCloser, int64, error) {
	opt, err := s.store.hostGetOption(hostID)
	if err != nil {
		return nil, 0, err
	}
	return s.sftpClient.Download(opt, filePath)
}

// SFTPUpload 上传单个文件。
// relativePath 为文件相对于 dirPath 的路径（含文件名），上传目录时由前端传入 webkitRelativePath；
// 普通单文件上传时直接传文件名即可。
// 若 relativePath 包含子目录，会自动在远端创建对应目录。
func (s *Service) SFTPUpload(hostID, dirPath string, file multipart.File, relativePath string) error {
	opt, err := s.store.hostGetOption(hostID)
	if err != nil {
		return err
	}
	destPath := path.Join(dirPath, relativePath)
	// 若路径中含子目录，先确保目录存在
	if dir := path.Dir(destPath); dir != "/" && dir != "." && dir != dirPath {
		if err := s.sftpClient.Mkdir(opt, dir); err != nil {
			return err
		}
	}
	return s.sftpClient.Upload(opt, destPath, file)
}

// SFTPRemove 删除文件或目录，目录支持递归删除
func (s *Service) SFTPRemove(hostID, targetPath string, recursive bool) error {
	opt, err := s.store.hostGetOption(hostID)
	if err != nil {
		return err
	}
	return s.sftpClient.Remove(opt, targetPath, recursive)
}

// SFTPMkdir 创建目录
func (s *Service) SFTPMkdir(hostID, dirPath string) error {
	opt, err := s.store.hostGetOption(hostID)
	if err != nil {
		return err
	}
	return s.sftpClient.Mkdir(opt, dirPath)
}

// SFTPRename 重命名/移动文件或目录
func (s *Service) SFTPRename(hostID, oldPath, newPath string) error {
	opt, err := s.store.hostGetOption(hostID)
	if err != nil {
		return err
	}
	return s.sftpClient.Rename(opt, oldPath, newPath)
}
