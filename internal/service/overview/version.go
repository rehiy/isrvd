package overview

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/rehiy/libgo/archive"
	"github.com/rehiy/libgo/logman"
	"github.com/rehiy/libgo/request"
	"github.com/rehiy/libgo/upgrade"

	"isrvd/config"
)

const (
	upgradeServer = "https://isrvd.rehiy.com/update/"

	// countryCodeURL 国家代码探测服务，与 build/script/isrvd.sh 保持一致
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
	cachedTag      string
	cachedURL      string
	cacheTime      time.Time
	cacheDuration  = 1 * time.Hour
)

var (
	countryCacheMu       sync.Mutex
	cachedUpdaterImage   string
	countryCacheTime     time.Time
	countryCacheDuration = 6 * time.Hour
)

// executablePath 启动时记录，避免升级替换后 /proc/self/exe 失效
var executablePath, _ = os.Executable()

// CheckVersion 从升级服务器检测最新版本，带缓存
func (s *Service) CheckVersion() *VersionInfo {
	latest, releaseURL := fetchLatestTag()
	return &VersionInfo{
		Current:      config.Version,
		Latest:       latest,
		Release:      releaseURL,
		HasUpdate:    isNewerVersion(latest, config.Version),
		UpdaterImage: resolveUpdaterImage(),
	}
}

// resolveUpdaterImage 根据 IP 国家代码选择 docker-updater 镜像，带内存缓存。
// 国家代码为 CN 时使用 CNB 镜像源，其余情况使用默认镜像；探测失败时回退到默认镜像。
func resolveUpdaterImage() string {
	countryCacheMu.Lock()
	defer countryCacheMu.Unlock()

	if cachedUpdaterImage != "" && time.Since(countryCacheTime) < countryCacheDuration {
		return cachedUpdaterImage
	}

	image := updaterImageGlobal
	code, err := request.TextGet(countryCodeURL, request.Header{"User-Agent": "isrvd"})
	if err != nil {
		logman.Warn("country code detect failed", "error", err)
	} else if strings.EqualFold(strings.TrimSpace(code), "CN") {
		image = updaterImageCN
	}

	cachedUpdaterImage, countryCacheTime = image, time.Now()
	return image
}

// ApplySelfUpgrade 从升级服务器下载最新 tar.gz，提取二进制并替换当前程序
// 替换成功后由调用方负责延迟重启，确保 HTTP 响应先发出
func (s *Service) ApplySelfUpgrade() error {
	if executablePath == "" {
		return fmt.Errorf("获取可执行文件路径失败")
	}

	// tar.gz 内文件名为 isrvd-{os}-{arch}，windows 加 .exe，由 build.sh 打包规则决定
	binaryName := "isrvd-" + runtime.GOOS + "-" + runtime.GOARCH
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}

	u := upgrade.NewUpdater(upgradeServer, config.Version)
	u.TargetPath = executablePath
	u.Download = func(pkgURL, outputPath string) (string, error) {
		tarGzPath, err := request.Download(pkgURL, "", false)
		if err != nil {
			return "", fmt.Errorf("下载失败: %w", err)
		}
		defer os.Remove(tarGzPath)

		if err := archive.NewTarGz().UntarFile(tarGzPath, binaryName, outputPath); err != nil {
			return "", fmt.Errorf("解压失败: %w", err)
		}
		return outputPath, nil
	}

	return u.Apply()
}

// RestartSelf 重启当前进程
func (s *Service) RestartSelf() error {
	u := upgrade.NewUpdater("", "")
	u.TargetPath = executablePath
	return u.Restart()
}

func isNewerVersion(latest, current string) bool {
	l := strings.TrimPrefix(latest, "v")
	c := strings.TrimPrefix(current, "v")
	if l == "" || c == "" {
		return false
	}
	ll := strings.Split(l, ".")
	cl := strings.Split(c, ".")
	for i := 0; i < len(ll) || i < len(cl); i++ {
		a, b := verPart(ll, i), verPart(cl, i)
		if a > b {
			return true
		}
		if a < b {
			return false
		}
	}
	return false
}

func verPart(s []string, i int) int {
	if i >= len(s) {
		return 0
	}
	n := 0
	for _, c := range s[i] {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			break
		}
	}
	return n
}

// fetchLatestTag 调用升级服务器获取最新版本号和 Release URL，带内存缓存
func fetchLatestTag() (tag, releaseURL string) {
	versionCacheMu.Lock()
	defer versionCacheMu.Unlock()

	if cachedTag != "" && time.Since(cacheTime) < cacheDuration {
		return cachedTag, cachedURL
	}

	info, err := upgrade.NewUpdater(upgradeServer, config.Version).Check()
	if err != nil && (info == nil || info.Version == "") {
		logman.Warn("version check failed", "error", err)
		return cachedTag, cachedURL
	}

	if info != nil && info.Version != "" {
		cachedTag, cachedURL, cacheTime = info.Version, info.Release, time.Now()
	}

	return cachedTag, cachedURL
}
