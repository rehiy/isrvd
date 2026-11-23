package helper

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
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

func TestCopyZIPContentActualByteLimit(t *testing.T) {
	payload := strings.Repeat("x", 128<<10)
	written, err := copyZIPContent(io.Discard, strings.NewReader(payload), 1024, true)
	if err == nil || !strings.Contains(err.Error(), "超过限制") {
		t.Fatalf("expected actual byte limit rejection, got %v", err)
	}
	if written != 1025 {
		t.Fatalf("copied %d bytes, want limit + 1", written)
	}
	written, err = copyZIPContent(io.Discard, strings.NewReader(payload), 0, false)
	if err != nil || written != int64(len(payload)) {
		t.Fatalf("unlimited copy wrote %d bytes, error=%v", written, err)
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
