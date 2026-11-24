package compose

import (
	"os"
	"path/filepath"
	"testing"
)

func checkEnv(t *testing.T, dir, want string, mode os.FileMode) {
	t.Helper()
	path := filepath.Join(dir, EnvFileName)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != want {
		t.Fatalf(".env content = %q, want %q", content, want)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != mode {
		t.Fatalf(".env mode = %04o, want %04o", got, mode)
	}
}

func TestEnvApplyNewFilePermissions(t *testing.T) {
	reference := filepath.Join(t.TempDir(), "reference")
	if err := os.WriteFile(reference, nil, 0644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(reference)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := EnvApply(dir, nil); err != nil {
		t.Fatal(err)
	}
	checkEnv(t, dir, "", info.Mode().Perm())

	dir = t.TempDir()
	content := "VALUE=1\n"
	if err := EnvApply(dir, &content); err != nil {
		t.Fatal(err)
	}
	checkEnv(t, dir, content, info.Mode().Perm())
}

func TestEnvSaveAtomicReplaceRetainsPermissions(t *testing.T) {
	dir := t.TempDir()
	if err := EnvSave(dir, "PASSWORD=secret\n"); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(dir, EnvFileName), 0640); err != nil {
		t.Fatal(err)
	}
	if err := EnvSave(dir, "PASSWORD=replaced\n"); err != nil {
		t.Fatal(err)
	}
	checkEnv(t, dir, "PASSWORD=replaced\n", 0640)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("unexpected temp files: %v", entries)
	}
}

func TestEnvEnsureRetainsExistingContentAndPermissions(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, EnvFileName), []byte("EXISTING=1"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(dir, EnvFileName), 0644); err != nil {
		t.Fatal(err)
	}
	if err := EnvEnsure(dir); err != nil {
		t.Fatal(err)
	}
	checkEnv(t, dir, "EXISTING=1", 0644)
}

func TestEnvStateRestoreRetainsPermissions(t *testing.T) {
	dir := t.TempDir()
	if err := EnvSave(dir, "TOKEN=restored"); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(dir, EnvFileName), 0600); err != nil {
		t.Fatal(err)
	}
	before, err := EnvStateRead(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := EnvSave(dir, "TOKEN=updated"); err != nil {
		t.Fatal(err)
	}
	checkEnv(t, dir, "TOKEN=updated", 0600)
	if err := EnvStateRestore(dir, before); err != nil {
		t.Fatal(err)
	}
	checkEnv(t, dir, "TOKEN=restored", 0600)
	if err := EnvStateRestore(dir, EnvState{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(dir, EnvFileName)); !os.IsNotExist(err) {
		t.Fatalf("expected missing .env after rollback, got %v", err)
	}
}

func TestEnvRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	external := filepath.Join(t.TempDir(), "external.txt")
	if err := os.WriteFile(external, []byte("external"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, filepath.Join(dir, EnvFileName)); err != nil {
		t.Skipf("symlink not available: %v", err)
	}
	if err := EnvSave(dir, "SECRET=leak"); err == nil {
		t.Fatal("EnvSave accepted .env symlink")
	}
	if err := EnvEnsure(dir); err == nil {
		t.Fatal("EnvEnsure accepted .env symlink")
	}
	contents, err := os.ReadFile(external)
	if err != nil || string(contents) != "external" {
		t.Fatalf("symlink target modified: data=%q, err=%v", contents, err)
	}
}
