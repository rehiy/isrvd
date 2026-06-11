// Package filer 文件管理业务服务
package filer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rehiy/libgo/archive"
	"github.com/rehiy/libgo/filer"
	"github.com/rehiy/libgo/logman"

	"isrvd/config"
)

// Service 文件管理业务服务
type Service struct{}

// NewService 创建文件管理业务服务
func NewService() *Service {
	return &Service{}
}

// FileInfo 文件信息
type FileInfo struct {
	Path    string    `json:"path"`
	Name    string    `json:"name"`
	Size    int64     `json:"size"`
	IsDir   bool      `json:"isDir"`
	Mode    string    `json:"mode"`
	ModeO   string    `json:"modeO"`
	ModTime time.Time `json:"modTime"`
}

// AbsPath 解析用户相对路径为绝对路径，并防止目录遍历和符号链接逃逸
// 安全策略：解析所有符号链接后，验证实际路径仍在 home 目录内
func (s *Service) AbsPath(username, path string) (string, error) {
	home := filepath.Clean(filepath.Join(config.Server.RootDirectory, "share"))
	if username != "" {
		if member, ok := config.Members[username]; ok {
			home = filepath.Clean(member.HomeDirectory)
		}
	}

	// 构建请求的绝对路径
	abs := filepath.Clean(filepath.Join(home, path))

	// 解析 home 目录的符号链接（home 本身可能是符号链接）
	homeReal, err := filepath.EvalSymlinks(home)
	if err != nil {
		homeReal = home // 如果 home 不存在，使用原始路径
	}

	// 尝试解析请求路径的符号链接
	// 如果路径不存在，逐级向上查找存在的父目录并验证
	realPath := abs
	for {
		real, err := filepath.EvalSymlinks(realPath)
		if err == nil {
			realPath = real
			break
		}
		// 路径不存在，检查父目录
		parent := filepath.Dir(realPath)
		if parent == realPath {
			// 已到达根目录，无法继续向上
			break
		}
		realPath = parent
	}

	// 验证实际路径是否在 home 目录内
	rel, err := filepath.Rel(homeReal, realPath)
	if err != nil {
		logman.Warn("Path escape attempt", "username", username, "path", path, "realPath", realPath, "error", err)
		return "", fmt.Errorf("路径越界，拒绝访问")
	}
	if rel == ".." || filepath.IsAbs(rel) || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		logman.Warn("Path escape attempt", "username", username, "path", path, "realPath", realPath, "home", homeReal)
		return "", fmt.Errorf("路径越界，拒绝访问")
	}

	return abs, nil
}

// FileList 列出目录下的文件
func (s *Service) FileList(absPath, relPath string) ([]*FileInfo, error) {
	list, err := filer.List(absPath)
	if err != nil {
		return nil, err
	}
	var result []*FileInfo
	for _, f := range list {
		p := filepath.ToSlash(filepath.Join(relPath, f.Name))
		result = append(result, &FileInfo{
			Path:    p,
			Name:    f.Name,
			Size:    f.Size,
			IsDir:   f.IsDir,
			Mode:    f.Mode.String(),
			ModeO:   strconv.FormatInt(int64(f.Mode), 8),
			ModTime: time.Unix(f.ModTime, 0),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].IsDir && !result[j].IsDir {
			return true
		}
		if !result[i].IsDir && result[j].IsDir {
			return false
		}
		return result[i].Name < result[j].Name
	})
	return result, nil
}

// FileOpen 打开文件并返回文件信息，用于流式读取
func (s *Service) FileOpen(absPath string) (*os.File, os.FileInfo, error) {
	file, err := os.Open(absPath)
	if err != nil {
		return nil, nil, err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, nil, err
	}
	if info.IsDir() {
		file.Close()
		return nil, nil, fmt.Errorf("路径是目录")
	}
	return file, info, nil
}

// FileRead 读取文件内容
func (s *Service) FileRead(absPath string) ([]byte, error) {
	return os.ReadFile(absPath)
}

// FileWrite 写入文件内容（覆盖）
func (s *Service) FileWrite(absPath string, content []byte) error {
	return os.WriteFile(absPath, content, 0644)
}

// FileCreate 创建文件（使用 pango/filer）
func (s *Service) FileCreate(absPath string, content []byte) error {
	return filer.Write(absPath, content)
}

// FileMkdir 创建目录
func (s *Service) FileMkdir(absPath string) error {
	return os.Mkdir(absPath, 0755)
}

// FileDelete 删除文件或目录
func (s *Service) FileDelete(absPath string) error {
	return os.RemoveAll(absPath)
}

// FileRename 重命名文件
func (s *Service) FileRename(absPath, targetPath string) error {
	return os.Rename(absPath, targetPath)
}

// FileChmod 修改文件权限
func (s *Service) FileChmod(absPath string, modeStr string) error {
	mode, err := strconv.ParseUint(modeStr, 8, 32)
	if err != nil {
		return fmt.Errorf("无效的权限值: %w", err)
	}
	return os.Chmod(absPath, os.FileMode(mode))
}

// DirSize 计算目录大小（包含所有子目录和文件）
func (s *Service) DirSize(absPath string) (int64, error) {
	return filer.DirSize(absPath)
}

// FileZip 压缩文件或目录
func (s *Service) FileZip(absPath string) error {
	return archive.NewZipper().Zip(absPath)
}

// FileUnzip 解压 zip 文件
// absPath: zip 文件的绝对路径
// targetPath: 空值则在文件同级目录解压；非空则在该绝对路径目录解压
func (s *Service) FileUnzip(absPath, targetPath string) error {
	if targetPath == "" {
		return archive.NewZipper().Unzip(absPath)
	}

	zipName := filepath.Base(absPath)
	zipTargetPath := filepath.Join(targetPath, zipName)

	// 确保目标目录存在
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		return fmt.Errorf("无法创建目标目录: %w", err)
	}

	// 移动 zip 到目标目录
	if err := os.Rename(absPath, zipTargetPath); err != nil {
		return fmt.Errorf("无法移动文件到目标目录: %w", err)
	}

	// 解压
	if err := archive.NewZipper().Unzip(zipTargetPath); err != nil {
		// 解压失败，移回原位置
		_ = os.Rename(zipTargetPath, absPath)
		return err
	}

	// 移回 zip 文件
	return os.Rename(zipTargetPath, absPath)
}

// previewContentTypes 支持在线预览的文件扩展名 → Content-Type 映射
var previewContentTypes = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".bmp":  "image/bmp",
	".svg":  "image/svg+xml",
	".webp": "image/webp",
	".ico":  "image/x-icon",
	".tiff": "image/tiff",
	".tif":  "image/tiff",
	".mp3":  "audio/mpeg",
	".wav":  "audio/wav",
	".ogg":  "audio/ogg",
	".m4a":  "audio/mp4",
	".flac": "audio/flac",
	".aac":  "audio/aac",
	".mp4":  "video/mp4",
	".webm": "video/webm",
	".mov":  "video/quicktime",
	".m4v":  "video/x-m4v",
	".mkv":  "video/x-matroska",
	".pdf":  "application/pdf",
}

// PreviewContentType 根据文件扩展名返回预览 Content-Type。
// 返回空字符串表示不支持预览。
func (s *Service) PreviewContentType(ext string) string {
	return previewContentTypes[strings.ToLower(ext)]
}
