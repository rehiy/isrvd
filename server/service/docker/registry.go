package docker

import (
	"context"
	"fmt"

	"isrvd/pkgs/docker"
	"isrvd/server/config"
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

func (s *Service) updateRegistries(mutate func(*config.DockerConfig) error) error {
	var committed []*docker.RegistryConfig
	return config.Update(func(draft *config.Snapshot) error {
		if err := mutate(draft.Docker); err != nil {
			return err
		}
		committed = make([]*docker.RegistryConfig, 0, len(draft.Docker.Registries))
		for _, registry := range draft.Docker.Registries {
			if registry != nil {
				committed = append(committed, &docker.RegistryConfig{
					Name: registry.Name, Description: registry.Description, URL: registry.URL,
					Username: registry.Username, Password: registry.Password,
				})
			}
		}
		return nil
	}, func(_ *config.Snapshot) {
		s.docker.ReplaceRegistries(committed)
	})
}

// RegistryCreate 新建镜像仓库
func (s *Service) RegistryCreate(req RegistryUpsertRequest) error {
	if req.Name == "" || req.URL == "" {
		return fmt.Errorf("仓库名称和地址不能为空")
	}
	return s.updateRegistries(func(dockerConfig *config.DockerConfig) error {
		if registryIndex(dockerConfig.Registries, req.URL) >= 0 {
			return fmt.Errorf("仓库地址已存在: %s", req.URL)
		}
		dockerConfig.Registries = append(dockerConfig.Registries, &config.DockerRegistry{
			Name: req.Name, URL: req.URL, Username: req.Username,
			Password: req.Password, Description: req.Description,
		})
		return nil
	})
}

// RegistryUpdate 更新镜像仓库
func (s *Service) RegistryUpdate(originalURL string, req RegistryUpsertRequest) error {
	if originalURL == "" {
		return fmt.Errorf("缺少 url 参数")
	}
	if req.Name == "" || req.URL == "" {
		return fmt.Errorf("仓库名称和地址不能为空")
	}
	return s.updateRegistries(func(dockerConfig *config.DockerConfig) error {
		index := registryIndex(dockerConfig.Registries, originalURL)
		if index < 0 {
			return fmt.Errorf("仓库不存在: %s", originalURL)
		}
		if req.URL != originalURL && registryIndex(dockerConfig.Registries, req.URL) >= 0 {
			return fmt.Errorf("仓库地址已存在: %s", req.URL)
		}
		password := req.Password
		if password == "" {
			password = dockerConfig.Registries[index].Password
		}
		dockerConfig.Registries[index] = &config.DockerRegistry{
			Name: req.Name, URL: req.URL, Username: req.Username,
			Password: password, Description: req.Description,
		}
		return nil
	})
}

// RegistryDelete 删除镜像仓库
func (s *Service) RegistryDelete(url string) error {
	if url == "" {
		return fmt.Errorf("缺少 url 参数")
	}
	return s.updateRegistries(func(dockerConfig *config.DockerConfig) error {
		index := registryIndex(dockerConfig.Registries, url)
		if index < 0 {
			return fmt.Errorf("仓库不存在: %s", url)
		}
		dockerConfig.Registries = append(dockerConfig.Registries[:index], dockerConfig.Registries[index+1:]...)
		return nil
	})
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

// ─── 辅助函数 ───

func registryIndex(registries []*config.DockerRegistry, url string) int {
	for i, registry := range registries {
		if registry != nil && registry.URL == url {
			return i
		}
	}
	return -1
}
