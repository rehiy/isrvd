package helper

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeZIP(t *testing.T, entries map[string]string) string {
	t.Helper()
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	for name, content := range entries {
		w, err := z.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "data.zip")
	if err := os.WriteFile(path, buf.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestExtractZIPMaximumActualBytes(t *testing.T) {
	// Go 的 ZIP 读取器验证实际大小的时机可能晚于数据写出。
	// 将中央目录的未压缩长度改为 4，但解压内容仍有 128 KiB。
	path := writeZIP(t, map[string]string{"inflated.txt": strings.Repeat("x", 128<<10)})
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	central := bytes.LastIndex(data, []byte{'P', 'K', 1, 2})
	if central < 0 {
		t.Fatal("ZIP central directory not found")
	}
	binary.LittleEndian.PutUint32(data[central+24:central+28], 4)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	cleanup, err := ExtractZIP(context.Background(), path, dir, false, 1024)
	if err == nil || !strings.Contains(err.Error(), "超过限制") {
		t.Fatalf("expected actual size rejection, got %v", err)
	}
	if cleanup != nil {
		if err := cleanup(); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "inflated.txt")); !os.IsNotExist(err) {
		t.Fatalf("overflow file published despite error: %v", err)
	}
}

func TestExtractZIPCumulativeLimitAndRollback(t *testing.T) {
	path := writeZIP(t, map[string]string{"one.txt": strings.Repeat("a", 10), "two.txt": strings.Repeat("b", 10)})
	dir := t.TempDir()
	cleanup, err := ExtractZIP(context.Background(), path, dir, false, 16)
	if err == nil || !strings.Contains(err.Error(), "超过限制") {
		t.Fatalf("expected cumulative limit rejection, got %v", err)
	}
	if cleanup == nil {
		t.Fatal("failed extraction did not return rollback function")
	}
	if err := cleanup(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"one.txt", "two.txt"} {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Errorf("%s left after rollback: %v", name, err)
		}
	}
}

func TestExtractZIPExactLimit(t *testing.T) {
	path := writeZIP(t, map[string]string{"data.txt": "1234567890"})
	dir := t.TempDir()
	_, err := ExtractZIP(context.Background(), path, dir, false, 10)
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "data.txt"))
	if err != nil || string(content) != "1234567890" {
		t.Fatalf("content mismatch: %q, err=%v", content, err)
	}
}
