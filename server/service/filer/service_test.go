package filer

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"isrvd/pkgs/helper"
)

func TestFileReadRejectsInvalidUTF8(t *testing.T) {
	s := NewService()
	dir := t.TempDir()
	valid := []byte("你好, text\n")
	for name, data := range map[string][]byte{
		"valid.txt":   valid,
		"invalid.txt": {'a', 0xff, 'b'},
	} {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.FileRead(filepath.Join(dir, "valid.txt"))
	if err != nil || !bytes.Equal(got, valid) {
		t.Fatalf("UTF-8 text read failed: %q, %v", got, err)
	}
	if _, err := s.FileRead(filepath.Join(dir, "invalid.txt")); !errors.Is(err, helper.ErrBinary) {
		t.Fatalf("non-UTF8 text should be rejected for editing, got %v", err)
	}
}
