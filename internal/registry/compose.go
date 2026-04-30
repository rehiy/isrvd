package registry

import (
	"context"

	"github.com/rehiy/pango/logman"

	"isrvd/pkgs/compose"
)

// ComposeService 全局 compose 基础服务实例（通用部署能力）
var ComposeService *compose.ComposeService

// initCompose 初始化 compose 服务（依赖 DockerService 先完成初始化）
func initCompose() error {
	if DockerService == nil {
		logman.Warn("Compose service skipped: docker service not initialized")
		return nil
	}

	svc, err := compose.NewComposeService(DockerService)
	if err != nil {
		logman.Warn("Compose service initialization failed", "error", err)
		return err
	}
	ComposeService = svc

	return nil
}

// IsComposeAvailable 检查 compose 服务是否可用
func IsComposeAvailable(ctx context.Context) bool {
	if ComposeService == nil {
		return false
	}
	return IsDockerAvailable(ctx)
}
