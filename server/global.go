package server

import (
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	BaseDir  = "."                    // 基础目录
	UserMap  = map[string]string{}    // 用户名:明文密码
	Sessions = map[string]time.Time{} // 会话存储
)

// 文件信息结构
type FileInfo struct {
	Name    string    `json:"name"`
	Size    int64     `json:"size"`
	IsDir   bool      `json:"isDir"`
	Mode    string    `json:"mode"`
	ModTime time.Time `json:"modTime"`
	Path    string    `json:"path"`
}

// 解析参数信息
func InitConfig() {
	if baseDir := os.Getenv("BASE_DIR"); baseDir != "" {
		BaseDir = baseDir
	}

	if usersEnv := os.Getenv("USERS"); usersEnv != "" {
		for _, pair := range strings.Split(usersEnv, ",") {
			kv := strings.SplitN(pair, ":", 2)
			if len(kv) == 2 {
				UserMap[kv[0]] = kv[1]
			}
		}
	} else {
		UserMap["admin"] = "admin"
	}
}

// 注册接口路由
func InitRouter() {
	http.HandleFunc("/api/login", corsMiddleware(loginHandler))
	http.HandleFunc("/api/logout", corsMiddleware(authMiddleware(logoutHandler)))

	http.HandleFunc("/api/files", corsMiddleware(authMiddleware(filesHandler)))
	http.HandleFunc("/api/upload", corsMiddleware(authMiddleware(uploadHandler)))
	http.HandleFunc("/api/download", corsMiddleware(authMiddleware(downloadHandler)))
	http.HandleFunc("/api/delete", corsMiddleware(authMiddleware(deleteHandler)))
	http.HandleFunc("/api/mkdir", corsMiddleware(authMiddleware(mkdirHandler)))
	http.HandleFunc("/api/newfile", corsMiddleware(authMiddleware(newfileHandler)))
	http.HandleFunc("/api/edit", corsMiddleware(authMiddleware(editHandler)))
	http.HandleFunc("/api/rename", corsMiddleware(authMiddleware(renameHandler)))
	http.HandleFunc("/api/chmod", corsMiddleware(authMiddleware(chmodHandler)))

	http.HandleFunc("/api/zip", corsMiddleware(authMiddleware(zipHandler)))
	http.HandleFunc("/api/unzip", corsMiddleware(authMiddleware(unzipHandler)))
}
