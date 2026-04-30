// Package compose 提供基于 compose-spec 官方规范的 docker-compose 加载、
// 变量插值、校验与部署能力，完全拥抱 compose-go/v2 官方实现。
//
// 本包不负责底层容器/网络/卷的 CRUD，这些能力来自 pkgs/docker；
// compose 包只负责 compose 文件的"解析 → 转换 → 编排部署"。
package compose

import (
	"fmt"

	"isrvd/pkgs/docker"
)

// ComposeService compose 部署服务，封装对外能力入口
type ComposeService struct {
	docker *docker.DockerService
}

// NewComposeService 创建 compose 服务
// 依赖注入 docker 服务，用于复用容器/网络创建能力
func NewComposeService(d *docker.DockerService) (*ComposeService, error) {
	if d == nil {
		return nil, fmt.Errorf("docker service 未提供")
	}
	return &ComposeService{docker: d}, nil
}
