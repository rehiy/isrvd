//go:build unix

package helper

import (
	"os"
	"syscall"
)

func preserveFileOwner(file *os.File, previous os.FileInfo) error {
	current, err := file.Stat()
	if err != nil {
		return err
	}
	oldOwner, oldOK := previous.Sys().(*syscall.Stat_t)
	newOwner, newOK := current.Sys().(*syscall.Stat_t)
	if !oldOK || !newOK {
		return os.ErrInvalid
	}
	if oldOwner.Uid == newOwner.Uid && oldOwner.Gid == newOwner.Gid {
		return nil
	}
	return file.Chown(int(oldOwner.Uid), int(oldOwner.Gid))
}
