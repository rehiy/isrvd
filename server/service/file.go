package service

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"isrvd/server/helper"
	"isrvd/server/model"
)

// 文件服务
type FileService struct{}

// 文件服务实例
var FileInstance *FileService

// 创建文件服务实例
func GetFileService() *FileService {
	if FileInstance == nil {
		FileInstance = &FileService{}
	}
	return FileInstance
}

// 获取文件列表
func (fs *FileService) List(path string) ([]model.FileInfo, error) {
	if !helper.ValidatePath(path) {
		return nil, os.ErrPermission
	}

	files, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var fileList []model.FileInfo
	for _, f := range files {
		info, _ := f.Info()
		fileList = append(fileList, model.FileInfo{
			Path:    filepath.Join(path, info.Name()),
			Name:    info.Name(),
			Size:    info.Size(),
			IsDir:   info.IsDir(),
			Mode:    info.Mode().Perm().String(),
			ModeO:   strconv.FormatInt(int64(info.Mode()), 8),
			ModTime: info.ModTime(),
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
	return os.RemoveAll(path)
}

// 创建目录
func (fs *FileService) Mkdir(path string) error {
	if !helper.ValidatePath(path) {
		return os.ErrPermission
	}
	return os.Mkdir(path, 0755)
}

// 创建文件
func (fs *FileService) Create(path, content string) error {
	if !helper.ValidatePath(path) {
		return os.ErrPermission
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// 读取文件内容
func (fs *FileService) Read(path string) (string, error) {
	if !helper.ValidatePath(path) {
		return "", os.ErrPermission
	}
	if content, err := os.ReadFile(path); err == nil {
		return string(content), nil
	} else {
		return "", err
	}
}

// 写入文件内容
func (fs *FileService) Modify(path, content string) error {
	if !helper.ValidatePath(path) {
		return os.ErrPermission
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// 重命名文件
func (fs *FileService) Rename(path, target string) error {
	target = filepath.Join(filepath.Dir(path), target)
	if !helper.ValidatePath(path) || !helper.ValidatePath(target) {
		return os.ErrPermission
	}
	return os.Rename(path, target)
}

// 修改文件权限
func (fs *FileService) Chmod(path string, mode os.FileMode) error {
	if !helper.ValidatePath(path) {
		return os.ErrPermission
	}
	return os.Chmod(path, mode)
}

// 获取文件信息
func (fs *FileService) GetFileInfo(path string) (os.FileInfo, error) {
	if !helper.ValidatePath(path) {
		return nil, os.ErrPermission
	}
	return os.Stat(path)
}
