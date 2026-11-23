// Package filer 文件管理业务服务
package filer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/rehiy/libgo/filer"
	"github.com/rehiy/libgo/logman"

	"isrvd/pkgs/helper"
	"isrvd/server/config"
)

// Service 文件管理业务服务
type Service struct{}

// NewService 创建文件管理业务服务
func NewService() *Service {
	return &Service{}
}

// FileInfo 文件信息
type FileInfo struct {
	Path       string    `json:"path"`                 // 相对成员主目录的路径
	Name       string    `json:"name"`                 // 文件名
	Size       int64     `json:"size"`                 // 文件大小（字节）
	IsDir      bool      `json:"isDir"`                // 是否为目录
	IsLink     bool      `json:"isLink"`               // 是否为符号链接
	LinkTarget string    `json:"linkTarget,omitempty"` // 符号链接目标
	Mode       string    `json:"mode"`                 // 权限字符串（如 -rwxr-xr-x）
	ModeO      string    `json:"modeO"`                // 权限八进制表示（如 755）
	ModTime    time.Time `json:"modTime"`              // 最后修改时间
}

// AbsPath 解析用户相对路径为绝对路径，并防止目录遍历和符号链接逃逸
// 安全策略：解析所有符号链接后，验证实际路径仍在 home 目录内
func (s *Service) AbsPath(username, path string) (string, error) {
	snapshot := config.Current()
	home := filepath.Clean(filepath.Join(snapshot.Server.RootDirectory, "share"))
	if username != "" {
		if member, ok := snapshot.Members[username]; ok {
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
		itemAbsPath := filepath.Join(absPath, f.Name)
		isLink, linkTarget := resolveLinkInfo(itemAbsPath)
		isDir := f.IsDir
		if isLink {
			if targetInfo, err := os.Stat(itemAbsPath); err == nil {
				isDir = targetInfo.IsDir()
			}
		}
		result = append(result, &FileInfo{
			Path:       p,
			Name:       f.Name,
			Size:       f.Size,
			IsDir:      isDir,
			IsLink:     isLink,
			LinkTarget: linkTarget,
			Mode:       f.Mode.String(),
			ModeO:      strconv.FormatInt(int64(f.Mode), 8),
			ModTime:    time.Unix(f.ModTime, 0),
		})
	}
	slices.SortFunc(result, func(a, b *FileInfo) int {
		if a.IsDir != b.IsDir {
			if a.IsDir {
				return -1
			}
			return 1
		}
		return strings.Compare(a.Name, b.Name)
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

// FileRead 读取用于在线编辑的文本内容，超大文件应使用流式下载接口。
func (s *Service) FileRead(absPath string) ([]byte, error) {
	file, err := os.Open(absPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, helper.MaxEditableFileBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > helper.MaxEditableFileBytes {
		return nil, helper.ErrTooLarge
	}
	if !filer.IsTextBytes(data) {
		return nil, helper.ErrBinary
	}
	return data, nil
}

// FileWrite 写入文件内容（覆盖）
func (s *Service) FileWrite(ctx context.Context, absPath string, content []byte) error {
	return s.FileWriteFrom(ctx, absPath, bytes.NewReader(content))
}

// FileWriteFrom 流式写入文件（覆盖），避免大文件整体读入内存
func (s *Service) FileWriteFrom(ctx context.Context, absPath string, r io.Reader) error {
	if info, err := os.Lstat(absPath); err == nil && info.Mode()&os.ModeSymlink != 0 {
		absPath, err = filepath.EvalSymlinks(absPath)
		if err != nil {
			return err
		}
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	root, err := os.OpenRoot(filepath.Dir(absPath))
	if err != nil {
		return err
	}
	defer root.Close()
	return helper.WriteRootFile(ctx, root, filepath.Base(absPath), false, func(out io.Writer) error {
		_, err := io.Copy(out, helper.ContextReader{Context: ctx, Reader: r})
		return err
	})
}

// FileCreate 创建或覆盖文件；与在线保存共用原子发布，自动创建父目录。
func (s *Service) FileCreate(ctx context.Context, absPath string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return err
	}
	return s.FileWrite(ctx, absPath, content)
}

// FileCreateOnly 原子发布新文件。
func (s *Service) FileCreateOnly(ctx context.Context, filename string, content []byte) error {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer root.Close()
	return helper.WriteRootFile(ctx, root, filepath.Base(filename), true, func(out io.Writer) error {
		_, err := out.Write(content)
		return err
	})
}

// FileMkdir 创建目录
func (s *Service) FileMkdir(absPath string) error {
	return os.Mkdir(absPath, 0755)
}

// FileDelete 删除文件或目录
func (s *Service) FileDelete(absPath string) error {
	return os.RemoveAll(absPath)
}

// FileRename 重命名或移动文件
// 当目标父目录不存在时会自动创建（0755）。如果后续 rename 失败，已创建的空目录不会被清理。
// 调用方需保证 targetPath 已通过 AbsPath 校验，确保不越界。
func (s *Service) FileRename(absPath, targetPath string) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}
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
func (s *Service) DirSize(ctx context.Context, absPath string) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	var size int64
	err := filepath.WalkDir(absPath, func(_ string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		info, err := entry.Info()
		if err == nil {
			size += info.Size()
		}
		return err
	})
	if err != nil {
		return 0, err
	}
	return size, ctx.Err()
}

// FileZip 压缩文件或目录
func (s *Service) FileZip(ctx context.Context, absPath string) error {
	return helper.CreateZIP(ctx, absPath)
}

// FileUnzip 解压 zip 文件
// absPath: zip 文件的绝对路径
// targetPath: 空值则在文件同级目录解压；非空则在该绝对路径目录解压
func (s *Service) FileUnzip(ctx context.Context, absPath, targetPath string) error {
	if targetPath == "" {
		targetPath = filepath.Dir(absPath)
	}
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		return err
	}
	_, err := helper.ExtractZIP(ctx, absPath, targetPath, true, 0)
	return err
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

// ─── 辅助函数 ───

func resolveLinkInfo(path string) (bool, string) {
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		return false, ""
	}
	target, err := os.Readlink(path)
	if err != nil {
		return true, ""
	}
	return true, filepath.ToSlash(filepath.Clean(target))
}
