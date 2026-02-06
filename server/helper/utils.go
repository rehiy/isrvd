package helper

import (
	"crypto/md5"
	"encoding/hex"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/pango/filer"

	"isrvd/server/config"
	"isrvd/server/model"
)

// 计算MD5哈希
func Md5sum(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

// 获取用户的绝对路径
func GetAbsolutePath(c *gin.Context, path string) string {
	home := filepath.Join(config.RootDirectory, "share")

	// 根据用户名获取用户主目录
	if name := c.GetString("username"); name != "" {
		if member, ok := config.Members[name]; ok {
			home = member.HomeDirectory
		}
	}

	// 验证最终路径是否在允许的基础目录下（防止目录遍历攻击）
	abs := filepath.Clean(filepath.Join(home, path))
	if strings.HasPrefix(abs, home) && !strings.Contains(abs, "..") {
		return abs
	}

	return home
}

// 获取目录下的文件列表
func FileList(path, rely string) ([]*model.FileInfo, error) {
	list, err := filer.List(path)
	if err != nil {
		return nil, err
	}

	var fileList []*model.FileInfo
	for _, f := range list {
		fileList = append(fileList, &model.FileInfo{
			Path:    filepath.Join(rely, f.Name),
			Name:    f.Name,
			Size:    f.Size,
			IsDir:   f.IsDir,
			Mode:    f.Mode.String(),
			ModeO:   strconv.FormatInt(int64(f.Mode), 8),
			ModTime: time.Unix(f.ModTime, 0),
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
