package docker

import (
	"context"
	"fmt"

	"isrvd/config"
	pkgdocker "isrvd/pkgs/docker"
)

// RegistryUpsertRequest 仓库新建/更新请求
type RegistryUpsertRequest struct {
	Name        string `json:"name" binding:"required"`
	URL         string `json:"url" binding:"required"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	Description string `json:"description"`
}

// ListRegistries 列出已配置的镜像仓库
func (s *Service) ListRegistries() any {
	return s.docker.ListRegistries()
}

// syncRegistriesToConfig 将当前 DockerService 的仓库同步到全局 config 并落盘
func (s *Service) syncRegistriesToConfig() error {
	regs := s.docker.GetRegistryConfigs()
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

// CreateRegistry 新建镜像仓库
func (s *Service) CreateRegistry(req RegistryUpsertRequest) error {
	reg := &pkgdocker.RegistryConfig{
		Name:        req.Name,
		URL:         req.URL,
		Username:    req.Username,
		Password:    req.Password,
		Description: req.Description,
	}
	if err := s.docker.AddRegistry(reg); err != nil {
		return err
	}
	return s.syncRegistriesToConfig()
}

// UpdateRegistry 更新镜像仓库
func (s *Service) UpdateRegistry(originalURL string, req RegistryUpsertRequest) error {
	if originalURL == "" {
		return fmt.Errorf("缺少 url 参数")
	}
	reg := &pkgdocker.RegistryConfig{
		Name:        req.Name,
		URL:         req.URL,
		Username:    req.Username,
		Password:    req.Password,
		Description: req.Description,
	}
	if err := s.docker.UpdateRegistry(originalURL, reg); err != nil {
		return err
	}
	return s.syncRegistriesToConfig()
}

// DeleteRegistry 删除镜像仓库
func (s *Service) DeleteRegistry(url string) error {
	if url == "" {
		return fmt.Errorf("缺少 url 参数")
	}
	if err := s.docker.DeleteRegistry(url); err != nil {
		return err
	}
	return s.syncRegistriesToConfig()
}

// PushImage 推送镜像到仓库
func (s *Service) PushImage(ctx context.Context, req pkgdocker.ImagePushRequest) (map[string]string, error) {
	msg, targetRef, err := s.docker.PushImage(ctx, req)
	if err != nil {
		return nil, err
	}
	return map[string]string{"image": req.Image, "target": targetRef, "message": msg}, nil
}

// PullFromRegistry 从仓库拉取镜像
func (s *Service) PullFromRegistry(ctx context.Context, req pkgdocker.ImagePullFromRegistryRequest) (map[string]string, error) {
	msg, imageRef, err := s.docker.PullFromRegistry(ctx, req)
	if err != nil {
		return nil, err
	}
	return map[string]string{"image": imageRef, "message": msg}, nil
}
