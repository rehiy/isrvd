package services

import (
	"os"
	"path/filepath"
	"sort"

	"isrvd/internal/models"
	"isrvd/pkg/utils"
)

// FileService 文件服务
type FileService struct{}

// NewFileService 创建文件服务实例
func NewFileService() *FileService {
	return &FileService{}
}

// ListFiles 获取文件列表
func (fs *FileService) ListFiles(path string) ([]models.FileInfo, error) {
	if !utils.ValidatePath(path) {
		return nil, os.ErrPermission
	}

	absPath := utils.GetAbsolutePath(path)
	files, err := os.ReadDir(absPath)
	if err != nil {
		return nil, err
	}

	var fileList []models.FileInfo
	for _, f := range files {
		info, _ := f.Info()
		fileList = append(fileList, models.FileInfo{
			Name:    info.Name(),
			Size:    info.Size(),
			IsDir:   info.IsDir(),
			Mode:    info.Mode().Perm().String(),
			ModTime: info.ModTime(),
			Path:    filepath.Join(path, info.Name()),
		})
	}

	// 排序：目录在前，然后按名称排序
	sort.Slice(fileList, func(i, j int) bool {
		if fileList[i].IsDir && !fileList[j].IsDir {
			return true
		}
		if !fileList[i].IsDir && fileList[j].IsDir {
			return false
		}
		return fileList[i].Name < fileList[j].Name
	})

	return fileList, nil
}

// DeleteFile 删除文件或目录
func (fs *FileService) DeleteFile(path string) error {
	if !utils.ValidatePath(path) {
		return os.ErrPermission
	}

	absPath := utils.GetAbsolutePath(path)
	return os.RemoveAll(absPath)
}

// CreateDirectory 创建目录
func (fs *FileService) CreateDirectory(path, name string) error {
	if !utils.ValidatePath(path) || !utils.ValidatePath(name) {
		return os.ErrPermission
	}

	absPath := filepath.Join(utils.GetAbsolutePath(path), name)
	return os.Mkdir(absPath, 0755)
}

// CreateFile 创建文件
func (fs *FileService) CreateFile(path, name, content string) error {
	if !utils.ValidatePath(path) || !utils.ValidatePath(name) {
		return os.ErrPermission
	}

	absPath := filepath.Join(utils.GetAbsolutePath(path), name)
	return os.WriteFile(absPath, []byte(content), 0644)
}

// ReadFile 读取文件内容
func (fs *FileService) ReadFile(path string) (string, error) {
	if !utils.ValidatePath(path) {
		return "", os.ErrPermission
	}

	absPath := utils.GetAbsolutePath(path)
	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// WriteFile 写入文件内容
func (fs *FileService) WriteFile(path, content string) error {
	if !utils.ValidatePath(path) {
		return os.ErrPermission
	}

	absPath := utils.GetAbsolutePath(path)
	return os.WriteFile(absPath, []byte(content), 0644)
}

// RenameFile 重命名文件
func (fs *FileService) RenameFile(path, newPath string) error {
	if !utils.ValidatePath(path) || !utils.ValidatePath(newPath) {
		return os.ErrPermission
	}

	oldAbsPath := utils.GetAbsolutePath(path)
	newAbsPath := filepath.Join(filepath.Dir(oldAbsPath), newPath)
	return os.Rename(oldAbsPath, newAbsPath)
}

// ChangeMode 修改文件权限
func (fs *FileService) ChangeMode(path string, mode os.FileMode) error {
	if !utils.ValidatePath(path) {
		return os.ErrPermission
	}

	absPath := utils.GetAbsolutePath(path)
	return os.Chmod(absPath, mode)
}

// GetFileInfo 获取文件信息
func (fs *FileService) GetFileInfo(path string) (os.FileInfo, error) {
	if !utils.ValidatePath(path) {
		return nil, os.ErrPermission
	}

	absPath := utils.GetAbsolutePath(path)
	return os.Stat(absPath)
}

// Global file service instance
var FileServiceInstance = NewFileService()
