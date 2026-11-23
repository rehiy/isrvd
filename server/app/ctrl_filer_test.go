package app

import (
	"path/filepath"
	"testing"
)

func TestFilerHomeRootGuard(t *testing.T) {
	home := filepath.Join(t.TempDir(), "share")
	tests := []struct {
		path string
		root bool
	}{
		{".", true},
		{"/", true},
		{"nested/..", true},
		{"/nested/..", true},
		{"nested/file.txt", false},
		{"file.txt", false},
		{"nested", false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			absPath := filepath.Join(home, tt.path)
			if got := isFilerHomeRoot(absPath, home); got != tt.root {
				t.Errorf("path %q resolves to %q: root=%v, want %v", tt.path, absPath, got, tt.root)
			}
		})
	}
}
