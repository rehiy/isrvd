package compose

import (
	"os"
	"path/filepath"
	"testing"
)

func checkEnv(t *testing.T, dir, want string) {
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
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf(".env mode = %04o, want 0600", got)
	}
}

func TestEnvSavePrivateAtomicReplace(t *testing.T) {
	dir := t.TempDir()
	if err := EnvSave(dir, "PASSWORD=secret\n"); err != nil {
		t.Fatal(err)
	}
	checkEnv(t, dir, "PASSWORD=secret\n")

	// Older releases created 0644 files; saving must not preserve those permissions.
	if err := os.Chmod(filepath.Join(dir, EnvFileName), 0644); err != nil {
		t.Fatal(err)
	}
	if err := EnvSave(dir, "PASSWORD=replaced\n"); err != nil {
		t.Fatal(err)
	}
	checkEnv(t, dir, "PASSWORD=replaced\n")

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("unexpected temp files: %v", entries)
	}
}

func TestEnvEnsurePrivateExistingContent(t *testing.T) {
	dir := t.TempDir()
	if err := EnvEnsure(dir); err != nil {
		t.Fatal(err)
	}
	checkEnv(t, dir, "")
	if err := os.WriteFile(filepath.Join(dir, EnvFileName), []byte("EXISTING=1"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(dir, EnvFileName), 0644); err != nil {
		t.Fatal(err)
	}
	if err := EnvEnsure(dir); err != nil {
		t.Fatal(err)
	}
	checkEnv(t, dir, "EXISTING=1")
}

func TestEnvStateRestorePrivate(t *testing.T) {
	dir := t.TempDir()
	if err := EnvStateRestore(dir, EnvState{Exists: true, Content: "TOKEN=restored"}); err != nil {
		t.Fatal(err)
	}
	checkEnv(t, dir, "TOKEN=restored")
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
