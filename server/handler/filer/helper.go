package filer

import (
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/pango/filer"

	"isrvd/server/config"
)

// FileInfo 文件信息结构
type FileInfo struct {
	Path    string    `json:"path"`
	Name    string    `json:"name"`
	Size    int64     `json:"size"`
	IsDir   bool      `json:"isDir"`
	Mode    string    `json:"mode"`
	ModeO   string    `json:"modeO"`
	ModTime time.Time `json:"modTime"`
}

// getAbsolutePath 获取用户的绝对路径
func getAbsolutePath(c *gin.Context, path string) string {
	home := filepath.Join(config.RootDirectory, "share")

	if name := c.GetString("username"); name != "" {
		if member, ok := config.Members[name]; ok {
			home = member.HomeDirectory
		}
	}

	abs := filepath.Clean(filepath.Join(home, path))
	if len(abs) >= len(home) && abs[:len(home)] == home {
		return abs
	}

	return home
}

// fileList 获取目录下的文件列表
func fileList(path, rely string) ([]*FileInfo, error) {
	list, err := filer.List(path)
	if err != nil {
		return nil, err
	}

	var result []*FileInfo
	for _, f := range list {
		p := filepath.ToSlash(filepath.Join(rely, f.Name))
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
