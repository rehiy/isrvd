package service

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"

	"isrvd/server/helper"
	"isrvd/server/model"
)

// zip服务
type ZipService struct{}

// zip服务实例
var ZipServiceInstance *ZipService

// 创建zip服务实例
func NewZipService() *ZipService {
	if ZipServiceInstance == nil {
		ZipServiceInstance = &ZipService{}
	}
	return ZipServiceInstance
}

// 创建压缩文件
func (zs *ZipService) CreateZip(path, zipName string) error {
	if !helper.ValidatePath(path) || !helper.ValidatePath(zipName) {
		return os.ErrPermission
	}

	srcPath := helper.GetAbsolutePath(path)

	// 确保zipName有.zip扩展名
	if !strings.HasSuffix(strings.ToLower(zipName), ".zip") {
		zipName += ".zip"
	}

	// 在源路径的父目录中创建zip文件
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

// 解压文件
func (zs *ZipService) ExtractZip(path, zipName string) error {
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

	return nil
}

// 判断文件是否为zip文件
func (zs *ZipService) IsZipFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return ext == ".zip"
}

// 获取zip文件信息
func (zs *ZipService) GetZipInfo(zipPath string) ([]model.ZipFileInfo, error) {
	if !helper.ValidatePath(zipPath) {
		return nil, os.ErrPermission
	}

	fullPath := helper.GetAbsolutePath(zipPath)
	reader, err := zip.OpenReader(fullPath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var files []model.ZipFileInfo
	for _, f := range reader.File {
		fileInfo := model.ZipFileInfo{
			Name:           f.Name,
			Size:           int64(f.UncompressedSize64),
			CompressedSize: int64(f.CompressedSize64),
			ModTime:        f.Modified,
			IsDir:          f.FileInfo().IsDir(),
			CRC32:          f.CRC32,
		}
		files = append(files, fileInfo)
	}

	return files, nil
}
