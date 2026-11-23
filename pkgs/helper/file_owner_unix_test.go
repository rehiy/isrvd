//go:build unix

package helper

import (
	"io"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestWriteRootFileRetainsOwnership(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("changing file ownership requires root")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "private")
	if err := os.WriteFile(target, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chown(target, 1000, 1000); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := WriteRootFile(t.Context(), root, "private", false, func(out io.Writer) error {
		_, err := io.WriteString(out, "new")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	info, err := root.Stat("private")
	if err != nil {
		t.Fatal(err)
	}
	owner := info.Sys().(*syscall.Stat_t)
	if owner.Uid != 1000 || owner.Gid != 1000 || info.Mode().Perm() != 0600 {
		t.Fatalf("metadata changed: uid=%d gid=%d mode=%v", owner.Uid, owner.Gid, info.Mode())
	}
}
