package overview

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rehiy/libgo/logman"
	"github.com/rehiy/libgo/request"
	"github.com/rehiy/libgo/upgrade"

	"isrvd/server/config"
)

// 本文件是升级源与版本展示：升级服务地址、站点类型（国内/海外）、最新版本缓存。
// 不依赖 upgrade.go；升级的执行见 upgrade.go。

const (
	// upgradeServer 升级服务；area=cn 时它会优先下发国内镜像的升级包
	upgradeServer = "https://isrvd.rehiy.com/update/"

	// countryCodeURL 国家代码探测服务，与 build/script 下的安装脚本保持一致
	countryCodeURL = "https://ipip.rehi.org/country_code"

	// updaterImageCN 中国大陆使用的 docker-updater 镜像（CNB 镜像源）
	updaterImageCN = "docker.cnb.cool/rehiy/docker-updater:latest"
	// updaterImageGlobal 默认（海外）使用的 docker-updater 镜像
	updaterImageGlobal = "rehiy/docker-updater:latest"
)

// VersionInfo 版本信息
type VersionInfo struct {
	Current      string `json:"current"`           // 当前版本号
	Latest       string `json:"latest"`            // 最新版本号
	Release      string `json:"release,omitempty"` // 最新版本发布页 URL
	HasUpdate    bool   `json:"hasUpdate"`         // 是否有可用更新
	UpdaterImage string `json:"updaterImage"`      // 推荐的 docker-updater 镜像（按国家代码自动选择）
}

var (
	versionCacheMu sync.Mutex
	cachedVersion  VersionInfo
	cacheTime      time.Time
	cacheDuration  = 1 * time.Hour
)

var (
	countryCacheMu       sync.Mutex
	cachedChina          bool
	countryCacheTime     time.Time // 零值表示尚未探测成功
	countryCacheDuration = 6 * time.Hour
)

// CheckVersion 从升级服务器检测最新版本，带缓存
func (s *Service) CheckVersion(ctx context.Context) *VersionInfo {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	info := fetchLatestVersion(ctx)
	info.Current = config.Version
	info.UpdaterImage = updaterImageGlobal
	if inChina(ctx) {
		info.UpdaterImage = updaterImageCN
	}
	return &info
}

// ─── 辅助函数 ───

// fetchLatestVersion 复用 libgo 的检查结果，缓存版本信息及是否有可用升级包。
func fetchLatestVersion(ctx context.Context) VersionInfo {
	versionCacheMu.Lock()
	defer versionCacheMu.Unlock()

	if cachedVersion.Latest != "" && time.Since(cacheTime) < cacheDuration {
		return cachedVersion
	}

	u := upgrade.NewUpdaterContext(ctx, upgradeServer, config.Version)
	info, err := u.Check()
	if ctx.Err() != nil {
		return cachedVersion
	}
	if err != nil && !errors.Is(err, upgrade.ErrNoUpdate) {
		logman.Warn("version check failed", "error", err)
	}
	if info != nil && info.Version != "" {
		cachedVersion = VersionInfo{Latest: info.Version, Release: info.Release, HasUpdate: err == nil}
		cacheTime = time.Now()
	}

	return cachedVersion
}

// inChina 由本机出口 IP 的国家代码判定是否位于中国大陆，带内存缓存。
// 升级下载源与 docker-updater 镜像都据此选择；探测失败时按海外处理。
func inChina(ctx context.Context) bool {
	countryCacheMu.Lock()
	defer countryCacheMu.Unlock()

	if !countryCacheTime.IsZero() && time.Since(countryCacheTime) < countryCacheDuration {
		return cachedChina
	}

	client := request.Client{Method: http.MethodGet, Url: countryCodeURL, Headers: request.Header{"User-Agent": "isrvd"}, Timeout: 8 * time.Second}
	code, err := client.RequestContext(ctx)
	if ctx.Err() != nil {
		return false
	}
	if err != nil {
		logman.Warn("country code detect failed", "error", err)
	}

	cachedChina = err == nil && strings.EqualFold(strings.TrimSpace(string(code)), "CN")
	countryCacheTime = time.Now()
	return cachedChina
}
