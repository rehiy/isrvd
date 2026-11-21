package docker

import (
	"context"
	"io"

	"isrvd/pkgs/docker"
)

// ContainerFileListResult 目录列表结果（透传 pkgs/docker）
type ContainerFileListResult = docker.ContainerFileListResult

// ContainerFileList 列出容器内目录内容
func (s *Service) ContainerFileList(ctx context.Context, containerID, dirPath string) (*ContainerFileListResult, error) {
	return s.docker.ContainerFileList(ctx, containerID, dirPath)
}

// ContainerFileOpen 打开容器文件流。
func (s *Service) ContainerFileOpen(ctx context.Context, containerID, filePath string) (io.ReadCloser, int64, error) {
	return s.docker.ContainerFileOpen(ctx, containerID, filePath)
}

// ContainerFileUpload 上传单个文件到容器内指定目录
func (s *Service) ContainerFileUpload(ctx context.Context, containerID, destDir, fileName string, content io.Reader, size int64) error {
	return s.docker.ContainerFileUpload(ctx, containerID, destDir, fileName, content, size)
}

// ContainerFileRemove 删除容器内文件或目录
func (s *Service) ContainerFileRemove(ctx context.Context, containerID, targetPath string, recursive bool) error {
	return s.docker.ContainerFileRemove(ctx, containerID, targetPath, recursive)
}

// ContainerFileMkdir 在容器内创建目录
func (s *Service) ContainerFileMkdir(ctx context.Context, containerID, dirPath string) error {
	return s.docker.ContainerFileMkdir(ctx, containerID, dirPath)
}

// ContainerFileRename 重命名/移动容器内文件
func (s *Service) ContainerFileRename(ctx context.Context, containerID, oldPath, newPath string) error {
	return s.docker.ContainerFileRename(ctx, containerID, oldPath, newPath)
}

// ContainerFileRead 读取容器内文件文本内容
func (s *Service) ContainerFileRead(ctx context.Context, containerID, filePath string) (string, error) {
	return s.docker.ContainerFileRead(ctx, containerID, filePath)
}

// ContainerFileWrite 写入文本内容到容器内文件
func (s *Service) ContainerFileWrite(ctx context.Context, containerID, filePath, content string) error {
	return s.docker.ContainerFileWrite(ctx, containerID, filePath, content)
}

func (s *Service) ContainerFileCreateOnly(ctx context.Context, containerID, filePath, content string) error {
	return s.docker.ContainerFileCreateOnly(ctx, containerID, filePath, content)
}

// ContainerFileChmod 修改容器内文件权限
func (s *Service) ContainerFileChmod(ctx context.Context, containerID, targetPath, mode string) error {
	return s.docker.ContainerFileChmod(ctx, containerID, targetPath, mode)
}
