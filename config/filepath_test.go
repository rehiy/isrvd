package config

import (
	"path/filepath"
	"testing"
)

func TestPathToAbs_Basic(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		rootDir string
		want    string
	}{
		{"empty path with root", "", "/root", filepath.Clean("/root")},
		{"empty path without root", "", "", "."},
		{"relative path", "docker", "/root", filepath.Clean("/root/docker")},
		{"absolute path", "/absolute/path", "/root", filepath.Clean("/absolute/path")},
		{"empty root with relative path", "docker", "", filepath.Clean("docker")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PathToAbs(tt.path, tt.rootDir)
			if got != tt.want {
				t.Errorf("PathToAbs(%q, %q) = %q, want %q", tt.path, tt.rootDir, got, tt.want)
			}
		})
	}
}

func TestPathToRel_Basic(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		rootDir string
		want    string
	}{
		{"inside root", "/root/docker", "/root", "." + string(filepath.Separator) + "docker"},
		{"outside root", "/other/path", "/root", "/other/path"},
		{"empty path", "", "/root", ""},
		{"relative path", "docker", "/root", "docker"},
		{"empty rootDir", "/path", "", "/path"},
		{"exact root", "/root", "/root", "." + string(filepath.Separator)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PathToRel(tt.path, tt.rootDir)
			if got != tt.want {
				t.Errorf("PathToRel(%q, %q) = %q, want %q", tt.path, tt.rootDir, got, tt.want)
			}
		})
	}
}

func TestDenormalizePathsWithOldRoot(t *testing.T) {
	// 模拟旧配置
	oldRoot := "/old/root"
	conf := &Config{
		Server: &ServerConfig{
			RootDirectory: oldRoot,
		},
		Docker: &DockerConfig{
			ContainerRoot: "/old/root/docker",
		},
		Members: []*MemberConfig{
			{Username: "test", HomeDirectory: "/old/root/home/test"},
		},
	}

	// 调用 denormalizePaths（不需要设置 Server）
	denormalizePaths(conf)

	// 验证：应该使用 oldRoot 转换
	expectedDocker := "." + string(filepath.Separator) + "docker"
	if conf.Docker.ContainerRoot != expectedDocker {
		t.Errorf("Docker.ContainerRoot = %q, want %q", conf.Docker.ContainerRoot, expectedDocker)
	}

	expectedHome := "." + string(filepath.Separator) + "home/test"
	if conf.Members[0].HomeDirectory != expectedHome {
		t.Errorf("Members[0].HomeDirectory = %q, want %q", conf.Members[0].HomeDirectory, expectedHome)
	}
}

func TestDenormalizePathsWithNil(t *testing.T) {
	// 测试 nil conf
	denormalizePaths(nil) // 不应该 panic

	// 测试 nil Server
	conf := &Config{Server: nil}
	denormalizePaths(conf) // 不应该 panic

	// 测试空 RootDirectory
	conf2 := &Config{Server: &ServerConfig{RootDirectory: ""}}
	denormalizePaths(conf2) // 不应该 panic
}

func TestPathToRelWithSymlinks(t *testing.T) {
	// 测试符号链接场景（不实际创建，测试逻辑分支）
	tests := []struct {
		name    string
		path    string
		rootDir string
	}{
		{"path with symlink", "/root/link/to/docker", "/root"},
		{"root with symlink", "/root/docker", "/root/link"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 只是确保不会 panic
			result := PathToRel(tt.path, tt.rootDir)
			t.Logf("PathToRel(%q, %q) = %q", tt.path, tt.rootDir, result)
		})
	}
}
