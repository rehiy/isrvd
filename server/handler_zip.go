package server

import (
	"archive/zip"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// 压缩
func zipHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	var req struct {
		Path    string `json:"path"`
		ZipName string `json:"zipName"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}

	if req.Path == "" || req.ZipName == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Path and zip name required"})
		return
	}

	srcPath := filepath.Join(BaseDir, filepath.Clean(req.Path))
	zipPath := filepath.Join(filepath.Dir(srcPath), req.ZipName)

	f, err := os.Create(zipPath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Cannot create zip file"})
		return
	}
	defer f.Close()

	zipWriter := zip.NewWriter(f)
	defer zipWriter.Close()

	baseName := filepath.Base(srcPath)
	err = filepath.Walk(srcPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(srcPath, filePath)
		if err != nil {
			return err
		}

		zipPath := filepath.Join(baseName, relPath)

		w, err := zipWriter.Create(zipPath)
		if err != nil {
			return err
		}

		file, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(w, file)
		return err
	})

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Cannot create zip archive"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Archive created successfully"})
}

// 解压
func unzipHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	var req struct {
		Path    string `json:"path"`
		ZipName string `json:"zipName"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}

	if req.Path == "" || req.ZipName == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Path and zip name required"})
		return
	}

	zipPath := filepath.Join(BaseDir, filepath.Clean(req.Path), req.ZipName)
	extractDir := filepath.Join(BaseDir, filepath.Clean(req.Path))

	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Cannot open zip file"})
		return
	}
	defer reader.Close()

	for _, f := range reader.File {
		// 防止 Zip Slip 攻击
		destPath := filepath.Join(extractDir, f.Name)
		rel, err := filepath.Rel(extractDir, destPath)
		if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
			continue
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(destPath, 0755)
			continue
		}

		os.MkdirAll(filepath.Dir(destPath), 0755)

		rc, err := f.Open()
		if err != nil {
			continue
		}

		outFile, err := os.Create(destPath)
		if err != nil {
			rc.Close()
			continue
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			continue
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Archive extracted successfully"})
}
