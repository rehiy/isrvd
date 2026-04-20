// Package compose 业务层：组合 pkgs/compose 通用能力 + 项目约定路径。
package compose

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/rehiy/pango/logman"

	"isrvd/pkgs/archive"
	"isrvd/pkgs/compose"
	"isrvd/pkgs/docker"
	"isrvd/pkgs/swarm"
)

// safeName 校验实例名，防止路径穿越
var safeName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)

// DeployService 统一的 Compose 部署业务服务
//
// 提供两种独立的部署入口：
//   - DeployYml：基于 compose 文本直接部署（docker 容器 / swarm 服务）
//   - DeployZip：下载 zip → 解压 → 写 .env → 部署为单机容器（仅 docker）
type DeployService struct {
	docker  *docker.DockerService
	compose *compose.ComposeService
	swarm   *swarm.SwarmManager
}

// NewDeployService 创建 compose 部署服务
// swarm 可为 nil，此时 target="swarm" 的请求会返回未初始化错误
func NewDeployService(d *docker.DockerService, c *compose.ComposeService, s *swarm.SwarmManager) (*DeployService, error) {
	if d == nil || c == nil {
		return nil, fmt.Errorf("docker 或 compose 服务未提供")
	}
	return &DeployService{docker: d, compose: c, swarm: s}, nil
}

// ComposeDeployTarget 部署目标
type ComposeDeployTarget string

const (
	TargetDocker ComposeDeployTarget = "docker"
	TargetSwarm  ComposeDeployTarget = "swarm"
)

// DeployYmlRequest 基于 compose 文本的部署请求
type DeployYmlRequest struct {
	Target       ComposeDeployTarget `json:"target" binding:"required"` // docker | swarm
	Content      string              `json:"content" binding:"required"`
	Env          map[string]any      `json:"env"`          // 可选：注入 compose 解析期的环境变量
	ProjectName  string              `json:"projectName"`  // 可选：compose project 名
	InternalOnly bool                `json:"internalOnly"` // 可选：仅内网模式，剥离所有 ports 宿主机映射
}

// DeployYmlResult 文本部署结果
type DeployYmlResult struct {
	Target ComposeDeployTarget `json:"target"` // 部署目标
	Items  []string            `json:"items"`  // 容器名 / swarm 服务名，格式 "name (shortId)"
}

// DeployZipRequest 基于 zip 压缩包的部署请求（仅 docker）
type DeployZipRequest struct {
	URL          string         `json:"url" binding:"required"`  // 远程 zip 地址
	Name         string         `json:"name" binding:"required"` // 实例名（作为目录名与 compose project 名）
	Env          map[string]any `json:"env"`                     // 可选：写入 .env 的变量（CONTAINER_NAME / APP_NAME / NETWORK_NAME 等）
	InternalOnly bool           `json:"internalOnly"`            // 可选：仅内网模式，剥离所有 ports 宿主机映射
}

// DeployZipResult zip 部署结果
type DeployZipResult struct {
	InstallDir string   `json:"installDir"` // 安装目录
	Items      []string `json:"items"`      // 容器名，格式 "name (shortId)"
}

// DeployYml 基于 compose 文本直接部署
func (s *DeployService) DeployYml(ctx context.Context, req DeployYmlRequest) (*DeployYmlResult, error) {
	if req.Content == "" {
		return nil, fmt.Errorf("compose 内容不能为空")
	}
	project, err := compose.LoadProjectFromContent(ctx, req.Content, req.ProjectName)
	if err != nil {
		return nil, err
	}
	if req.InternalOnly {
		stripProjectPorts(project)
	}
	items, err := s.deployProject(ctx, req.Target, project)
	if err != nil {
		return nil, err
	}
	return &DeployYmlResult{Target: req.Target, Items: items}, nil
}

// DeployZip 下载 zip → 解压 → 写 .env → 加载并部署（仅 docker）
func (s *DeployService) DeployZip(ctx context.Context, req DeployZipRequest) (*DeployZipResult, error) {
	root := s.docker.ContainerRoot()
	if root == "" {
		return nil, fmt.Errorf("未配置容器数据根目录")
	}
	if !safeName.MatchString(req.Name) {
		return nil, fmt.Errorf("非法的实例名")
	}
	if req.URL == "" {
		return nil, fmt.Errorf("缺少安装包下载地址")
	}

	// 安装目录约定：{ContainerRoot}/{name}
	installDir := filepath.Join(root, req.Name)
	if _, err := os.Stat(installDir); err == nil {
		return nil, fmt.Errorf("目录已存在：%s，请先移除或使用其它实例名", installDir)
	}
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return nil, fmt.Errorf("创建安装目录失败: %w", err)
	}

	// 异常清理
	ok := false
	defer func() {
		if !ok {
			_ = os.RemoveAll(installDir)
		}
	}()

	// 下载并解压
	zipPath := filepath.Join(installDir, ".app.zip")
	if err := downloadFile(ctx, req.URL, zipPath); err != nil {
		return nil, fmt.Errorf("下载安装包失败: %w", err)
	}
	if err := archive.NewZipper().Unzip(zipPath); err != nil {
		return nil, fmt.Errorf("解压安装包失败: %w", err)
	}
	_ = os.Remove(zipPath)

	// 组装 env：所有变量均来自用户表单
	env := map[string]string{}
	for k, v := range req.Env {
		env[k] = fmt.Sprintf("%v", v)
	}
	if err := writeDotEnv(filepath.Join(installDir, ".env"), env); err != nil {
		return nil, fmt.Errorf("写入 .env 失败: %w", err)
	}

	// 加载并部署，项目名与实例名保持一致
	project, err := compose.LoadProject(ctx, compose.LoadOptions{
		WorkingDir:  installDir,
		ProjectName: req.Name,
		Environment: env,
	})
	if err != nil {
		return nil, err
	}
	if req.InternalOnly {
		stripProjectPorts(project)
	}
	items, err := s.deployProject(ctx, TargetDocker, project)
	if err != nil {
		return nil, fmt.Errorf("部署失败: %w", err)
	}

	ok = true
	logman.Info("Compose archive deployed",
		"name", req.Name,
		"installDir", installDir,
	)
	return &DeployZipResult{InstallDir: installDir, Items: items}, nil
}

// deployProject 根据 target 分发到 docker / swarm 部署
func (s *DeployService) deployProject(ctx context.Context, target ComposeDeployTarget, project *types.Project) ([]string, error) {
	switch target {
	case TargetDocker:
		return s.compose.DeployProject(ctx, project)
	case TargetSwarm:
		if s.swarm == nil {
			return nil, fmt.Errorf("swarm manager 未初始化")
		}
		if len(project.Services) == 0 {
			return nil, fmt.Errorf("compose 文件中没有定义服务")
		}
		var services []string
		for _, svc := range project.Services {
			req, err := compose.ServiceToSwarmRequest(project, svc)
			if err != nil {
				return services, err
			}
			id, err := s.swarm.CreateService(ctx, req)
			if err != nil {
				return services, fmt.Errorf("创建服务 %s 失败: %w", req.Name, err)
			}
			if len(id) > 12 {
				id = id[:12]
			}
			services = append(services, fmt.Sprintf("%s (%s)", req.Name, id))
			logman.Info("Swarm compose service deployed", "service", svc.Name, "id", id)
		}
		return services, nil
	default:
		return nil, fmt.Errorf("不支持的部署目标: %s", target)
	}
}

// downloadFile 下载 URL 到目标路径
func downloadFile(ctx context.Context, url, dest string) error {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := (&http.Client{Timeout: 5 * time.Minute}).Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

// stripProjectPorts 剥离 compose 项目中所有服务的宿主机端口映射
// 用于"仅内网模式"：容器仍可通过 docker 网络被其它容器 / APISIX 访问，但不占用宿主机端口
func stripProjectPorts(project *types.Project) {
	if project == nil {
		return
	}
	for name, svc := range project.Services {
		if len(svc.Ports) == 0 {
			continue
		}
		svc.Ports = nil
		project.Services[name] = svc
	}
}

// writeDotEnv 按 KEY=VALUE 纯文本写入 .env，由 compose-go 负责解析
func writeDotEnv(path string, env map[string]string) error {
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, k := range keys {
		if _, err := fmt.Fprintf(f, "%s=%s\n", k, env[k]); err != nil {
			return err
		}
	}
	return nil
}
