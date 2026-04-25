package archive

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// 归档服务
type Zipper struct{}

func NewZipper() *Zipper {
	return &Zipper{}
}

// 创建归档文件
func (zs *Zipper) Zip(path string) error {
	srcPath := path
	zipName := filepath.Base(srcPath) + ".zip"

	// 在源路径的父目录中创建归档文件
	f, err := os.Create(filepath.Join(filepath.Dir(srcPath), zipName))
	if err != nil {
		return err
	}
	defer f.Close()

	zipWriter := zip.NewWriter(f)
	defer zipWriter.Close()

	baseName := filepath.Base(srcPath)
	return filepath.Walk(srcPath, func(filePath string, info os.FileInfo, err error) error {
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

		// zip 规范要求内部路径使用正斜杠
		zipPath := filepath.ToSlash(filepath.Join(baseName, relPath))
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
}

// 解压归档文件
func (zs *Zipper) Unzip(path string) error {
	extractDir := filepath.Dir(path)
	reader, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, f := range reader.File {
		// 将 zip 内路径分隔符统一为系统分隔符，防止 Windows 下路径拼接异常
		name := filepath.FromSlash(f.Name)

		// 防止 Zip Slip 攻击
		destPath := filepath.Join(extractDir, name)
		rel, err := filepath.Rel(extractDir, destPath)
		if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
			continue
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		outFile, err := os.Create(destPath)
		if err != nil {
			rc.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}

	return nil
}
