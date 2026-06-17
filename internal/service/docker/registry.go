package docker

import (
	"context"
	"fmt"

	"isrvd/config"
	pkgDocker "isrvd/pkgs/docker"
)

// RegistryInfo 镜像仓库信息，保持前端稳定响应结构且不包含密码。
type RegistryInfo struct {
	Name        string `json:"name"`        // 仓库名称（唯一标识）
	URL         string `json:"url"`         // 仓库地址
	Username    string `json:"username"`    // 登录用户名
	Description string `json:"description"` // 仓库描述
}

// ImagePushRequest 镜像推送请求。
type ImagePushRequest struct {
	Image       string `json:"image" binding:"required"`       // 待推送的本地镜像（repo:tag）
	RegistryURL string `json:"registryUrl" binding:"required"` // 目标仓库地址
	Namespace   string `json:"namespace"`                      // 仓库命名空间（可选）
}

// ImagePullRequest 拉取镜像请求。
type ImagePullRequest struct {
	Image       string `json:"image" binding:"required"` // 待拉取的镜像（repo:tag）
	RegistryURL string `json:"registryUrl"`              // 来源仓库地址（可选，缺省用默认仓库）
	Namespace   string `json:"namespace"`                // 仓库命名空间（可选）
}

// RegistryUpsertRequest 仓库新建/更新请求
type RegistryUpsertRequest struct {
	Name        string `json:"name" binding:"required"` // 仓库名称（唯一标识）
	URL         string `json:"url" binding:"required"`  // 仓库地址
	Username    string `json:"username"`                // 登录用户名
	Password    string `json:"password"`                // 登录密码（为空保留原值）
	Description string `json:"description"`             // 仓库描述
}

// RegistryList 列出已配置的镜像仓库。
func (s *Service) RegistryList() []*RegistryInfo {
	regs := s.docker.Registries()
	result := make([]*RegistryInfo, 0, len(regs))
	for _, r := range regs {
		result = append(result, &RegistryInfo{
			Name:        r.Name,
			URL:         r.URL,
			Username:    r.Username,
			Description: r.Description,
		})
	}
	return result
}

// registriesConfigSync 将当前 DockerService 的仓库同步到全局 config 并落盘
func (s *Service) registriesConfigSync() error {
	regs := s.docker.Registries()
	cfgRegs := make([]*config.DockerRegistry, 0, len(regs))
	for _, r := range regs {
		cfgRegs = append(cfgRegs, &config.DockerRegistry{
			Name:        r.Name,
			Description: r.Description,
			URL:         r.URL,
			Username:    r.Username,
			Password:    r.Password,
		})
	}
	config.Docker.Registries = cfgRegs
	return config.Save()
}

// RegistryCreate 新建镜像仓库
func (s *Service) RegistryCreate(req RegistryUpsertRequest) error {
	reg := &pkgDocker.RegistryConfig{
		Name:        req.Name,
		URL:         req.URL,
		Username:    req.Username,
		Password:    req.Password,
		Description: req.Description,
	}
	if err := s.docker.RegistryCreate(reg); err != nil {
		return fmt.Errorf("创建镜像仓库失败: %w", err)
	}
	return s.registriesConfigSync()
}

// RegistryUpdate 更新镜像仓库
func (s *Service) RegistryUpdate(originalURL string, req RegistryUpsertRequest) error {
	if originalURL == "" {
		return fmt.Errorf("缺少 url 参数")
	}
	reg := &pkgDocker.RegistryConfig{
		Name:        req.Name,
		URL:         req.URL,
		Username:    req.Username,
		Password:    req.Password,
		Description: req.Description,
	}
	// 密码为空时保留原密码（前端编辑时不回显密码，空值表示不修改）
	if reg.Password == "" {
		reg.Password = s.docker.RegistryGetPassword(originalURL)
	}
	if err := s.docker.RegistryUpdate(originalURL, reg); err != nil {
		return fmt.Errorf("更新镜像仓库失败: %w", err)
	}
	return s.registriesConfigSync()
}

// RegistryDelete 删除镜像仓库
func (s *Service) RegistryDelete(url string) error {
	if url == "" {
		return fmt.Errorf("缺少 url 参数")
	}
	if err := s.docker.RegistryDelete(url); err != nil {
		return fmt.Errorf("删除镜像仓库失败: %w", err)
	}
	return s.registriesConfigSync()
}

// ImagePush 推送镜像到仓库
func (s *Service) ImagePush(ctx context.Context, req ImagePushRequest) (map[string]string, error) {
	msg, targetRef, err := s.docker.ImagePush(ctx, req.Image, req.RegistryURL, req.Namespace)
	if err != nil {
		return nil, err
	}
	return map[string]string{"image": req.Image, "target": targetRef, "message": msg}, nil
}

// ImagePull 从仓库拉取镜像
func (s *Service) ImagePull(ctx context.Context, req ImagePullRequest) (map[string]string, error) {
	msg, imageRef, err := s.docker.ImagePull(ctx, req.Image, req.RegistryURL, req.Namespace)
	if err != nil {
		return nil, err
	}
	return map[string]string{"image": imageRef, "message": msg}, nil
}
