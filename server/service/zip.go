package service

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"

	"isrvd/server/helper"
)

// 归档服务
type ZipService struct{}

// 归档服务实例
var ZipInstance *ZipService

// 创建归档服务实例
func GetZipService() *ZipService {
	if ZipInstance == nil {
		ZipInstance = &ZipService{}
	}
	return ZipInstance
}

// 创建归档文件
func (zs *ZipService) Zip(path, zipName string) error {
	if !helper.ValidatePath(path) || !helper.ValidatePath(zipName) {
		return os.ErrPermission
	}

	// 确保 zipName 有 .zip 扩展名
	if !strings.HasSuffix(strings.ToLower(zipName), ".zip") {
		zipName += ".zip"
	}

	srcPath := helper.GetAbsolutePath(path)

	// 在源路径的父目录中创建归档文件
	parentDir := filepath.Dir(srcPath)
	zipPath := filepath.Join(parentDir, zipName)

	f, err := os.Create(zipPath)
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
}

// 解压归档文件
func (zs *ZipService) Unzip(path, zipName string) error {
	if !helper.ValidatePath(path) || !helper.ValidatePath(zipName) {
		return os.ErrPermission
	}

	zipPath := filepath.Join(helper.GetAbsolutePath(path), zipName)
	extractDir := helper.GetAbsolutePath(path)

	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
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
