package compose

import (
	"context"
	"fmt"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/rehiy/pango/logman"
)

// DeployContent 通过 compose yaml 文本部署容器
// 使用 compose-go 官方加载器完成变量插值、校验，然后按 services 逐个创建容器
// 返回已创建的容器描述列表，格式 "name (shortId)"
func (s *ComposeService) DeployContent(ctx context.Context, content string) ([]string, error) {
	project, err := LoadProjectFromContent(ctx, content, "")
	if err != nil {
		return nil, err
	}
	return s.DeployProject(ctx, project)
}

// DeployProject 部署一个已加载的 compose Project
// 在创建容器前会先确保用到的非内置网络存在
// 返回已创建的容器描述列表，格式 "name (shortId)"
func (s *ComposeService) DeployProject(ctx context.Context, project *types.Project) ([]string, error) {
	if project == nil || len(project.Services) == 0 {
		return nil, fmt.Errorf("compose 项目为空或未定义服务")
	}

	// 先确保所有用到的外部网络存在
	for _, name := range collectNetworks(project) {
		if err := s.ensureNetwork(ctx, name); err != nil {
			return nil, fmt.Errorf("确保网络 %s 存在失败: %w", name, err)
		}
	}

	var containers []string
	for _, svc := range project.Services {
		req, err := ServiceToCreateRequest(project, svc)
		if err != nil {
			return containers, err
		}

		id, err := s.docker.CreateContainer(ctx, req)
		if err != nil {
			return containers, fmt.Errorf("创建容器 %s 失败: %w", req.Name, err)
		}

		shortID := id
		if len(shortID) > 12 {
			shortID = shortID[:12]
		}
		containers = append(containers, fmt.Sprintf("%s (%s)", req.Name, shortID))

		logman.Info("Compose container deployed",
			"project", project.Name,
			"service", svc.Name,
			"container", req.Name,
			"id", shortID,
		)
	}

	return containers, nil
}

// collectNetworks 收集项目中所有需要确保存在的网络名（过滤 docker 内置网络）
func collectNetworks(project *types.Project) []string {
	set := map[string]struct{}{}
	for _, svc := range project.Services {
		for _, n := range extractNetworks(project, svc) {
			set[n] = struct{}{}
		}
	}
	result := make([]string, 0, len(set))
	for k := range set {
		result = append(result, k)
	}
	return result
}

// ensureNetwork 确保指定的 bridge 网络存在（不存在则创建）
func (s *ComposeService) ensureNetwork(ctx context.Context, name string) error {
	networks, err := s.docker.ListNetworks(ctx)
	if err != nil {
		return err
	}
	for _, n := range networks {
		if n.Name == name {
			return nil
		}
	}
	_, err = s.docker.CreateNetwork(ctx, name, "bridge")
	return err
}
