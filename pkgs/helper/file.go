package helper

import (
	"context"
	"crypto/rand"
	"errors"
	"io"
	"os"
	"path/filepath"
)

// ContextReader checks cancellation before each read.
type ContextReader struct {
	Context context.Context
	Reader  io.Reader
}

func (r ContextReader) Read(p []byte) (int, error) {
	if err := r.Context.Err(); err != nil {
		return 0, err
	}
	return r.Reader.Read(p)
}

// WriteRootFile 写完整后发布目标文件，createOnly 使用硬链接避免覆盖任何已存在的路径。
func WriteRootFile(ctx context.Context, root *os.Root, name string, createOnly bool, write func(io.Writer) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	previous, err := root.Lstat(name)
	if err == nil {
		if createOnly {
			return os.ErrExist
		}
		if !previous.Mode().IsRegular() {
			return os.ErrInvalid
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	staged := filepath.Join(filepath.Dir(name), ".isrvd-write-"+rand.Text())
	file, err := root.OpenFile(staged, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer root.Remove(staged)
	err = write(file)
	if err == nil && previous != nil {
		err = PreserveFileOwner(file, previous)
		if err == nil {
			err = file.Chmod(previous.Mode().Perm())
		}
	}
	if err := errors.Join(err, file.Close(), ctx.Err()); err != nil {
		return err
	}
	if createOnly {
		return root.Link(staged, name)
	}
	return root.Rename(staged, name)
}
