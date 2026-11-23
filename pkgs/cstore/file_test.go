package cstore

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestFileStoreRejectsPathTraversal(t *testing.T) {
	dir := t.TempDir()
	store, err := newFileStore(filepath.Join(dir, "config"))
	if err != nil {
		t.Fatal(err)
	}

	keys := []string{"", ".", "..", "../outside.yml", "nested/inside.yml", "/absolute.yml", "..\\outside.yml"}
	for _, key := range keys {
		t.Run(key, func(t *testing.T) {
			if _, err := store.Get(key); err == nil {
				t.Errorf("Get(%q) accepted an invalid file key", key)
			}
			if err := store.Set(key, []byte("unexpected")); err == nil {
				t.Errorf("Set(%q) accepted an invalid file key", key)
			}
			if err := store.CheckAndSet(key, nil, []byte("unexpected")); err == nil {
				t.Errorf("CheckAndSet(%q) accepted an invalid file key", key)
			}
		})
	}
	if _, err := os.Stat(filepath.Join(dir, "outside.yml")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("invalid key wrote outside store directory: %v", err)
	}
}

func TestFileStoreAtomicUpdatesAndPermissions(t *testing.T) {
	dir := t.TempDir()
	store, err := newFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if data, err := store.Get("state.yml"); err != nil || data != nil {
		t.Fatalf("missing key: data=%q, err=%v", data, err)
	}
	if err := store.CheckAndSet("state.yml", nil, []byte("one")); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, "state.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("new key has mode %04o, want 0600", info.Mode().Perm())
	}
	if err := store.CheckAndSet("state.yml", nil, []byte("two")); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale missing-state baseline must conflict: %v", err)
	}
	if err := store.CheckAndSet("state.yml", []byte("one"), []byte("two")); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get("state.yml")
	if err != nil || string(got) != "two" {
		t.Fatalf("updated value: %q, err=%v", got, err)
	}
}

func TestDocumentRejectsStaleAndInvalidReads(t *testing.T) {
	store, err := newFileStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a, b := NewDocument(store, "state.yml"), NewDocument(store, "state.yml")
	if err := a.Set([]byte("one")); !errors.Is(err, ErrNotLoaded) {
		t.Fatalf("unloaded Set must fail: %v", err)
	}
	if _, err := a.Read(nil); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Read(nil); err != nil {
		t.Fatal(err)
	}
	if err := a.Set([]byte("one")); err != nil {
		t.Fatal(err)
	}
	if err := b.Set([]byte("stale")); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale document must conflict: %v", err)
	}
	if _, err := b.Read(nil); err != nil {
		t.Fatal(err)
	}
	if err := b.Set([]byte("two")); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Read(func([]byte) error { return errors.New("invalid payload") }); err == nil {
		t.Fatal("expected validation error")
	}
	if err := b.Set([]byte("overwrite")); !errors.Is(err, ErrNotLoaded) {
		t.Fatalf("failed validation must invalidate baseline: %v", err)
	}
}
