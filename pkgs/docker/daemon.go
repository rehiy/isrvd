package docker

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

const daemonConfigPath = "/etc/docker/daemon.json"

// SyncMirrorsToDaemon 将镜像加速地址写入 daemon.json 并 reload dockerd
func SyncMirrorsToDaemon(mirrors []string) error {
	if runtime.GOOS != "linux" {
		return nil
	}

	cfg, err := readDaemonConfig()
	if err != nil {
		return fmt.Errorf("读取 daemon.json 失败: %w", err)
	}

	if len(mirrors) == 0 {
		delete(cfg, "registry-mirrors")
	} else {
		cfg["registry-mirrors"] = mirrors
	}

	if err := writeDaemonConfig(cfg); err != nil {
		return fmt.Errorf("写入 daemon.json 失败: %w", err)
	}

	return reloadDockerd()
}

func readDaemonConfig() (map[string]any, error) {
	data, err := os.ReadFile(daemonConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]any), nil
		}
		return nil, err
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func writeDaemonConfig(cfg map[string]any) error {
	data, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(daemonConfigPath, data, 0644)
}

func reloadDockerd() error {
	return exec.Command("systemctl", "reload", "docker").Run()
}
