package helper

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
)

// ExtractZIP 解压普通文件及目录。禁止覆盖时，返回本次新增路径的清理函数。
func ExtractZIP(ctx context.Context, source, destination string, overwrite bool, maximumBytes uint64) (func() error, error) {
	archive, err := zip.OpenReader(source)
	if err != nil {
		return nil, err
	}
	defer archive.Close()
	sourceInfo, err := os.Stat(source)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(destination)
	if err != nil {
		return nil, err
	}
	defer root.Close()

	var created []string
	rollback := func() error {
		if overwrite || len(created) == 0 {
			return nil
		}
		dir, err := os.OpenRoot(destination)
		if err != nil {
			return err
		}
		defer dir.Close()
		var failures []error
		for i := len(created) - 1; i >= 0; i-- {
			if err := dir.Remove(created[i]); err != nil && !errors.Is(err, os.ErrNotExist) {
				failures = append(failures, err)
			}
		}
		return errors.Join(failures...)
	}
	var total uint64
	for _, file := range archive.File {
		if err := ctx.Err(); err != nil {
			return rollback, err
		}
		name := filepath.FromSlash(file.Name)
		if !filepath.IsLocal(name) {
			return rollback, fmt.Errorf("非法 ZIP 路径: %s", file.Name)
		}
		name = filepath.Clean(name)
		if name == "." && file.Mode().IsDir() {
			continue
		}
		if file.Mode().IsDir() {
			if err := zipMkdir(root, name, &created); err != nil {
				return rollback, err
			}
			continue
		}
		if !file.Mode().IsRegular() {
			return rollback, fmt.Errorf("不支持的 ZIP 文件类型: %s", file.Name)
		}
		var remaining uint64
		if maximumBytes > 0 {
			remaining = maximumBytes - total
			if file.UncompressedSize64 > remaining {
				return rollback, fmt.Errorf("ZIP 解压大小超过限制")
			}
		}
		if err := zipMkdir(root, filepath.Dir(name), &created); err != nil {
			return rollback, err
		}
		if existing, err := root.Stat(name); err == nil && os.SameFile(existing, sourceInfo) {
			return rollback, fmt.Errorf("ZIP 条目与源归档同名: %s", file.Name)
		}
		written, err := zipExtractFile(ctx, root, file, name, overwrite, &created, remaining, maximumBytes > 0)
		if err != nil {
			return rollback, err
		}
		total += written
	}
	return rollback, nil
}

func zipMkdir(root *os.Root, name string, created *[]string) error {
	if name == "." {
		return nil
	}
	if err := zipMkdir(root, filepath.Dir(name), created); err != nil {
		return err
	}
	if err := root.Mkdir(name, 0755); err != nil {
		if info, statErr := root.Lstat(name); statErr == nil && info.IsDir() {
			return nil
		}
		return err
	}
	*created = append(*created, name)
	return nil
}

func zipExtractFile(ctx context.Context, root *os.Root, file *zip.File, name string, overwrite bool, created *[]string, remaining uint64, limited bool) (uint64, error) {
	in, err := file.Open()
	if err != nil {
		return 0, err
	}
	defer in.Close()
	var written int64
	err = WriteRootFile(ctx, root, name, !overwrite, func(out io.Writer) error {
		reader := io.Reader(ContextReader{Context: ctx, Reader: in})
		// ZIP 声明的未压缩大小可能被伪造，必须限制实际读取的字节数。
		// 检测多读的 1 字节；超出 int64 上限时该限制无法由单个文件触达。
		if limited && remaining < math.MaxInt64 {
			reader = io.LimitReader(reader, int64(remaining)+1)
		}
		var copyErr error
		written, copyErr = io.Copy(out, reader)
		if copyErr != nil {
			return copyErr
		}
		if limited && uint64(written) > remaining {
			return fmt.Errorf("ZIP 解压大小超过限制")
		}
		return nil
	})
	if err == nil && !overwrite {
		*created = append(*created, name)
	}
	return uint64(written), err
}

// CreateZIP 压缩文件或目录，生成同目录的 <source>.zip。
func CreateZIP(ctx context.Context, source string) error {
	parent := filepath.Dir(source)
	root, err := os.OpenRoot(parent)
	if err != nil {
		return err
	}
	defer root.Close()
	return WriteRootFile(ctx, root, filepath.Base(source)+".zip", false, func(out io.Writer) error {
		archive := zip.NewWriter(out)
		err := filepath.WalkDir(source, func(filename string, entry fs.DirEntry, walkErr error) error {
			if err := errors.Join(ctx.Err(), walkErr); err != nil {
				return err
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			name, err := filepath.Rel(parent, filename)
			if err != nil {
				return err
			}
			if !info.IsDir() && !info.Mode().IsRegular() {
				return fmt.Errorf("不支持的 ZIP 文件类型: %s", name)
			}
			header, err := zip.FileInfoHeader(info)
			if err != nil {
				return err
			}
			header.Name = filepath.ToSlash(name)
			if info.IsDir() {
				header.Name += "/"
			} else {
				header.Method = zip.Deflate
			}
			dest, err := archive.CreateHeader(header)
			if err != nil || info.IsDir() {
				return err
			}
			file, err := root.Open(name)
			if err != nil {
				return err
			}
			_, err = io.Copy(dest, ContextReader{Context: ctx, Reader: file})
			return errors.Join(err, file.Close())
		})
		return errors.Join(err, archive.Close())
	})
}
