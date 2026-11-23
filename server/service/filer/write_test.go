package filer

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileCreateUsesAtomicWriteAndRetainsMode(t *testing.T) {
	service := NewService()
	dir := t.TempDir()
	target := filepath.Join(dir, "nested", "file.txt")
	if err := service.FileCreate(context.Background(), target, []byte("first")); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(target, 0600); err != nil {
		t.Fatal(err)
	}
	if err := service.FileCreate(context.Background(), target, []byte("replacement")); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "replacement" {
		t.Fatalf("unexpected saved content %q: %v", data, err)
	}
	info, err := os.Stat(target)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("existing permissions not retained: %v, %v", info, err)
	}
}

type partialErrorReader struct {
	delivered bool
}

func (r *partialErrorReader) Read(buf []byte) (int, error) {
	if !r.delivered {
		r.delivered = true
		return copy(buf, "partial"), nil
	}
	return 0, io.ErrUnexpectedEOF
}

func TestFileWriteFromDoesNotReplaceOnError(t *testing.T) {
	service := NewService()
	dir := t.TempDir()
	target := filepath.Join(dir, "existing.txt")
	if err := os.WriteFile(target, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := service.FileWriteFrom(context.Background(), target, &partialErrorReader{}); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("expected copy failure, got %v", err)
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "original" {
		t.Fatalf("failed write changed original file: %q, %v", data, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".isrvd-write-") {
			t.Fatalf("temporary file leaked: %s", entry.Name())
		}
	}
}

func TestFileCreateCanceledDoesNotOverwrite(t *testing.T) {
	service := NewService()
	dir := t.TempDir()
	target := filepath.Join(dir, "existing.txt")
	if err := os.WriteFile(target, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := service.FileCreate(ctx, target, []byte("changed")); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "original" {
		t.Fatalf("canceled write changed original file: %q, %v", data, err)
	}
}
