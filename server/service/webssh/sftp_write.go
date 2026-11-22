package webssh

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"time"

	"github.com/pkg/sftp"
	"github.com/rehiy/libgo/logman"
)

func (s *Service) sftpWrite(ctx context.Context, hostID string, conn *sftp.Client, name string, source io.Reader, size int64, createOnly bool) error {
	existing, err := conn.Lstat(name)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if createOnly && existing != nil {
		return os.ErrExist
	}
	if existing != nil && existing.Mode()&os.ModeSymlink != 0 {
		name, err = conn.RealPath(name)
		if err != nil {
			return err
		}
		existing, err = conn.Stat(name)
		if err != nil {
			return err
		}
	}
	_, replace := conn.HasExtension("posix-rename@openssh.com")
	if existing != nil {
		if !existing.Mode().IsRegular() {
			return os.ErrInvalid
		}
		if !replace {
			return errors.New("SFTP 服务器不支持安全覆盖已有文件")
		}
		target, err := conn.OpenFile(name, os.O_WRONLY)
		if err != nil {
			return err
		}
		if err := target.Close(); err != nil {
			return err
		}
	}

	staged := path.Join(path.Dir(name), ".isrvd-upload-"+rand.Text())
	file, err := conn.OpenFile(staged, os.O_WRONLY|os.O_CREATE|os.O_EXCL)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			s.removeSFTPTemp(ctx, hostID, conn, staged)
		}
	}()
	info, err := conn.Stat(staged)
	if err == nil {
		err = conn.Chmod(staged, 0600)
	}
	var written int64
	if err == nil {
		if size > 32<<10 {
			written, err = file.ReadFromWithConcurrency(source, sftpConcurrentRequests)
		} else {
			written, err = file.ReadFrom(source)
		}
	}
	if err := errors.Join(err, file.Close()); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}
	if written != size {
		return fmt.Errorf("写入文件失败: %w", io.ErrUnexpectedEOF)
	}
	mode := info.Mode()
	if existing != nil {
		mode = existing.Mode()
		oldOwner, oldOK := existing.Sys().(*sftp.FileStat)
		newOwner, newOK := info.Sys().(*sftp.FileStat)
		if oldOK && newOK && (oldOwner.UID != newOwner.UID || oldOwner.GID != newOwner.GID) {
			if err := conn.Chown(staged, int(oldOwner.UID), int(oldOwner.GID)); err != nil {
				return err
			}
		}
	}
	if err := conn.Chmod(staged, mode); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if existing != nil {
		err = conn.PosixRename(staged, name)
	} else {
		err = conn.Rename(staged, name)
	}
	if err != nil {
		if createOnly && ctx.Err() == nil {
			if _, statErr := conn.Lstat(name); statErr == nil {
				return os.ErrExist
			}
		}
		return fmt.Errorf("重命名失败: %w", err)
	}
	committed = true
	return nil
}

func (s *Service) removeSFTPTemp(ctx context.Context, hostID string, conn *sftp.Client, staged string) {
	if ctx.Err() == nil {
		if err := conn.Remove(staged); err == nil || errors.Is(err, os.ErrNotExist) {
			return
		}
	}
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()
	cleanup, closeConn, err := s.sftpConnect(cleanupCtx, hostID)
	if err == nil {
		defer closeConn()
		err = cleanup.Remove(staged)
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		logman.Warn("SFTP temporary file cleanup failed", "host", hostID, "path", staged, "error", err)
	}
}
