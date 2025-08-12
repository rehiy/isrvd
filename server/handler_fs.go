package server

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
)

// 文件列表
func filesHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "/"
	}

	absPath := filepath.Join(BaseDir, filepath.Clean(path))
	files, err := os.ReadDir(absPath)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Directory not found"})
		return
	}

	var fileList []FileInfo
	for _, f := range files {
		info, _ := f.Info()
		fileList = append(fileList, FileInfo{
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"path":  path,
		"files": fileList,
	})
}

// 上传文件
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	path := r.FormValue("path")
	if path == "" {
		path = "/"
	}

	absPath := filepath.Join(BaseDir, filepath.Clean(path), header.Filename)
	f, err := os.Create(absPath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Cannot create file"})
		return
	}
	defer f.Close()

	_, err = io.Copy(f, file)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Cannot write file"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "File uploaded successfully"})
}

// 下载文件
func downloadHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("file")
	if path == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "No file specified"})
		return
	}

	absPath := filepath.Join(BaseDir, filepath.Clean(path))
	f, err := os.Open(absPath)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "File not found"})
		return
	}
	defer f.Close()

	w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(absPath))
	io.Copy(w, f)
}

// 删除文件或目录
func deleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "DELETE" && r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	path := r.URL.Query().Get("file")
	if path == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "No file specified"})
		return
	}

	absPath := filepath.Join(BaseDir, filepath.Clean(path))
	err := os.RemoveAll(absPath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Cannot delete file"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "File deleted successfully"})
}

// 新建目录
func mkdirHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	var req struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}

	if req.Name == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Directory name required"})
		return
	}

	absPath := filepath.Join(BaseDir, filepath.Clean(req.Path), req.Name)
	err := os.Mkdir(absPath, 0755)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Cannot create directory"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Directory created successfully"})
}

// 新建文件
func newfileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	var req struct {
		Path    string `json:"path"`
		Name    string `json:"name"`
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}

	if req.Name == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "File name required"})
		return
	}

	absPath := filepath.Join(BaseDir, filepath.Clean(req.Path), req.Name)
	err := os.WriteFile(absPath, []byte(req.Content), 0644)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Cannot create file"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "File created successfully"})
}

// 编辑文件
func editHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("file")
	if path == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "No file specified"})
		return
	}

	absPath := filepath.Join(BaseDir, filepath.Clean(path))

	// 读取文件内容
	if r.Method == "GET" {
		content, err := os.ReadFile(absPath)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "File not found"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"path":    path,
			"content": string(content),
		})
		return
	}

	// 保存文件内容
	if r.Method == "POST" || r.Method == "PUT" {
		var req struct {
			Content string `json:"content"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
			return
		}

		err := os.WriteFile(absPath, []byte(req.Content), 0644)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Cannot save file"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "File saved successfully"})
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
	json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
}

// 重命名
func renameHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" && r.Method != "PUT" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	var req struct {
		OldPath string `json:"oldPath"`
		NewName string `json:"newName"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}

	if req.OldPath == "" || req.NewName == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Old path and new name required"})
		return
	}

	oldAbsPath := filepath.Join(BaseDir, filepath.Clean(req.OldPath))
	newAbsPath := filepath.Join(filepath.Dir(oldAbsPath), req.NewName)

	err := os.Rename(oldAbsPath, newAbsPath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Cannot rename file"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "File renamed successfully"})
}

// 修改权限
func chmodHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("file")
	if path == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "No file specified"})
		return
	}

	absPath := filepath.Join(BaseDir, filepath.Clean(path))

	// 获取当前权限
	if r.Method == "GET" {
		info, err := os.Stat(absPath)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "File not found"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"file": filepath.Base(path),
			"mode": strconv.FormatUint(uint64(info.Mode().Perm()), 8),
		})
		return
	}

	// 修改权限
	if r.Method == "POST" || r.Method == "PUT" {
		var req struct {
			Mode string `json:"mode"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
			return
		}

		mode, err := strconv.ParseUint(req.Mode, 8, 32)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid mode"})
			return
		}

		err = os.Chmod(absPath, os.FileMode(mode))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Cannot change permissions"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "Permissions changed successfully"})
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
	json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
}
