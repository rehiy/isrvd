package service

import (
	"os"
	"path/filepath"
	"sort"

	"isrvd/server/helper"
	"isrvd/server/model"
)

// 文件服务
type FileService struct{}

// 文件服务实例
var FileServiceInstance *FileService

// 创建文件服务实例
func NewFileService() *FileService {
	if FileServiceInstance == nil {
		FileServiceInstance = &FileService{}
	}
	return FileServiceInstance
}

// 获取文件列表
func (fs *FileService) ListFiles(path string) ([]model.FileInfo, error) {
	if !helper.ValidatePath(path) {
		return nil, os.ErrPermission
	}

	absPath := helper.GetAbsolutePath(path)
	files, err := os.ReadDir(absPath)
	if err != nil {
		return nil, err
	}

	var fileList []model.FileInfo
	for _, f := range files {
		info, _ := f.Info()
		fileList = append(fileList, model.FileInfo{
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

// 删除文件或目录
func (fs *FileService) DeleteFile(path string) error {
	if !helper.ValidatePath(path) {
		return os.ErrPermission
	}

	absPath := helper.GetAbsolutePath(path)
	return os.RemoveAll(absPath)
}

// 创建目录
func (fs *FileService) CreateDirectory(path, name string) error {
	if !helper.ValidatePath(path) || !helper.ValidatePath(name) {
		return os.ErrPermission
	}

	absPath := filepath.Join(helper.GetAbsolutePath(path), name)
	return os.Mkdir(absPath, 0755)
}

// 创建文件
func (fs *FileService) CreateFile(path, name, content string) error {
	if !helper.ValidatePath(path) || !helper.ValidatePath(name) {
		return os.ErrPermission
	}

	absPath := filepath.Join(helper.GetAbsolutePath(path), name)
	return os.WriteFile(absPath, []byte(content), 0644)
}

// 读取文件内容
func (fs *FileService) ReadFile(path string) (string, error) {
	if !helper.ValidatePath(path) {
		return "", os.ErrPermission
	}

	absPath := helper.GetAbsolutePath(path)
	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// 写入文件内容
func (fs *FileService) WriteFile(path, content string) error {
	if !helper.ValidatePath(path) {
		return os.ErrPermission
	}

	absPath := helper.GetAbsolutePath(path)
	return os.WriteFile(absPath, []byte(content), 0644)
}

// 重命名文件
func (fs *FileService) RenameFile(path, newPath string) error {
	if !helper.ValidatePath(path) || !helper.ValidatePath(newPath) {
		return os.ErrPermission
	}

	oldAbsPath := helper.GetAbsolutePath(path)
	newAbsPath := filepath.Join(filepath.Dir(oldAbsPath), newPath)
	return os.Rename(oldAbsPath, newAbsPath)
}

// 修改文件权限
func (fs *FileService) ChangeMode(path string, mode os.FileMode) error {
	if !helper.ValidatePath(path) {
		return os.ErrPermission
	}

	absPath := helper.GetAbsolutePath(path)
	return os.Chmod(absPath, mode)
}

// 获取文件信息
func (fs *FileService) GetFileInfo(path string) (os.FileInfo, error) {
	if !helper.ValidatePath(path) {
		return nil, os.ErrPermission
	}

	absPath := helper.GetAbsolutePath(path)
	return os.Stat(absPath)
}
