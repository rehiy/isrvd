package helper

import (
	"context"
	"errors"
	"io"
	"os"
	"runtime"
	"testing"
)

func TestWriteRootFilePrivateCreation(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	for _, createOnly := range []bool{false, true} {
		name := "overwrite"
		if createOnly {
			name = "create-only"
		}
		if err := WriteRootFile(t.Context(), root, name, createOnly, func(out io.Writer) error {
			_, err := io.WriteString(out, "private")
			return err
		}); err != nil {
			t.Fatal(err)
		}
		info, err := root.Stat(name)
		if err != nil {
			t.Fatal(err)
		}
		if runtime.GOOS != "windows" && info.Mode().Perm()&0077 != 0 {
			t.Fatalf("new file is not private: %v", info.Mode())
		}
	}
}

func TestWriteRootFileCreateOnlyRace(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	err = WriteRootFile(context.Background(), root, "file", true, func(out io.Writer) error {
		if err := root.WriteFile("file", []byte("concurrent"), 0600); err != nil {
			return err
		}
		_, err := io.WriteString(out, "replacement")
		return err
	})
	if !errors.Is(err, os.ErrExist) {
		t.Fatalf("expected existing file error, got %v", err)
	}
	data, err := root.ReadFile("file")
	if err != nil || string(data) != "concurrent" {
		t.Fatalf("concurrent file overwritten: %q, %v", data, err)
	}
}
