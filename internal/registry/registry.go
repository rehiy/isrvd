// Package registry 提供服务注册和管理功能
package registry

import "github.com/rehiy/libgo/logman"

func Init() {
	if err := initApisix(); err != nil {
		logman.Warn("Apisix client initialization skipped", "error", err)
	}
	if err := initCaddy(); err != nil {
		logman.Warn("Caddy client initialization skipped", "error", err)
	}
	if err := initDocker(); err != nil {
		logman.Warn("Docker service initialization skipped", "error", err)
	}
}

func Close() {
	if DockerService != nil {
		if err := DockerService.Close(); err != nil {
			logman.Warn("Docker service close failed", "error", err)
		}
	}
	ApisixClient = nil
	CaddyClient = nil
	DockerService = nil
	SwarmService = nil
}
