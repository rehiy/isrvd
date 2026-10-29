package overview

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/rehiy/libgo/logman"
	"github.com/rehiy/libgo/request"
	"github.com/rehiy/libgo/upgrade"

	"isrvd/server/config"
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
	cachedVersion  VersionInfo
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

var upgradeHTTPClient = &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return errors.New("too many upgrade redirects")
	}
	return validateUpgradeURL(req.URL)
}}

// CheckVersion 从升级服务器检测最新版本，带缓存
func (s *Service) CheckVersion(ctx context.Context) *VersionInfo {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	info := fetchLatestVersion(ctx)
	info.Current = config.Version
	info.UpdaterImage = resolveUpdaterImage(ctx)
	return &info
}

var (
	selfUpgradeMu       sync.Mutex
	selfUpgradeCancel   context.CancelFunc
	selfUpgradeDone     chan struct{}
	selfUpgradeStopping bool
)

// StartSelfUpgrade 在后台升级，重复请求不再启动第二次升级。
func (s *Service) StartSelfUpgrade() error {
	selfUpgradeMu.Lock()
	defer selfUpgradeMu.Unlock()
	if selfUpgradeStopping {
		return context.Canceled
	}
	if selfUpgradeCancel == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 270*time.Second)
		selfUpgradeCancel = cancel
		selfUpgradeDone = make(chan struct{})
		go s.runSelfUpgrade(ctx, cancel)
	}
	return nil
}

// StopSelfUpgrade 取消请求并等待实际工作退出，确保关闭不打断文件替换。
func (s *Service) StopSelfUpgrade() {
	selfUpgradeMu.Lock()
	selfUpgradeStopping = true
	done := selfUpgradeDone
	if selfUpgradeCancel != nil {
		selfUpgradeCancel()
	}
	selfUpgradeMu.Unlock()
	if done != nil {
		<-done
	}
}

func (s *Service) runSelfUpgrade(ctx context.Context, cancel context.CancelFunc) {
	logger := logman.Named("upgrade")
	defer func() {
		cancel()
		selfUpgradeMu.Lock()
		selfUpgradeCancel = nil
		close(selfUpgradeDone)
		selfUpgradeDone = nil
		selfUpgradeMu.Unlock()
		if recovered := recover(); recovered != nil {
			logger.Error("upgrade panicked", "error", recovered)
		}
	}()
	if err := s.ApplySelfUpgrade(ctx); err != nil {
		if errors.Is(err, upgrade.ErrNoUpdate) {
			logger.Info("no update available")
		} else {
			logger.Error("upgrade failed", "error", err)
		}
		return
	}
	time.Sleep(500 * time.Millisecond)
	if err := s.RestartSelf(); err != nil {
		logger.Error("upgrade restart failed", "error", err)
	}
}

// ApplySelfUpgrade 只适配 tar.gz 解压；检查、下载、校验、替换及回滚复用 libgo。
func (s *Service) ApplySelfUpgrade(ctx context.Context) error {
	if executablePath == "" {
		return fmt.Errorf("获取可执行文件路径失败")
	}
	binaryName := "isrvd-" + runtime.GOOS + "-" + runtime.GOARCH
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	u := upgrade.NewUpdaterContext(ctx, upgradeServer, config.Version)
	u.TargetPath = executablePath
	u.HTTPClient = upgradeHTTPClient
	u.Download = func(pkgURL, outputPath string) (string, error) {
		pkg, err := url.Parse(pkgURL)
		if err != nil {
			return "", err
		}
		if err := validateUpgradeURL(pkg); err != nil {
			return "", err
		}
		client := *u.HTTPClient
		client.Timeout = 0 // 下载使用整项升级的上下文期限。
		req := &request.Client{Method: http.MethodGet, Url: pkgURL}
		res, err := req.DoContext(ctx, &client)
		if err != nil {
			return "", err
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			return "", fmt.Errorf("download returned HTTP %d", res.StatusCode)
		}
		reader, err := gzip.NewReader(res.Body)
		if err != nil {
			return "", err
		}
		defer reader.Close()
		limited := &io.LimitedReader{R: reader, N: 512 << 20}
		tr := tar.NewReader(limited)
		for {
			if err := ctx.Err(); err != nil {
				return "", err
			}
			hdr, err := tr.Next()
			if err != nil {
				if err == io.EOF {
					return "", fmt.Errorf("archive: %s not found", binaryName)
				}
				return "", err
			}
			if hdr.Size > limited.N {
				return "", errors.New("upgrade archive exceeds 512 MiB")
			}
			if hdr.Typeflag == tar.TypeReg && filepath.Base(hdr.Name) == binaryName {
				return request.SaveStream(tr, outputPath)
			}
		}
	}
	return u.Apply()
}

func validateUpgradeURL(u *url.URL) error {
	if u.Scheme != "https" || u.User != nil || (u.Port() != "" && u.Port() != "443") {
		return errors.New("untrusted upgrade URL")
	}
	switch u.Hostname() {
	case "isrvd.rehiy.com", "github.com", "release-assets.githubusercontent.com", "objects.githubusercontent.com":
		return nil
	}
	return errors.New("untrusted upgrade URL")
}

// RestartSelf 重启当前进程
func (s *Service) RestartSelf() error {
	selfUpgradeMu.Lock()
	defer selfUpgradeMu.Unlock()
	if selfUpgradeStopping {
		return context.Canceled
	}
	u := upgrade.NewUpdater("", "")
	u.TargetPath = executablePath
	return u.Restart()
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
	u.HTTPClient = upgradeHTTPClient
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

// resolveUpdaterImage 根据 IP 国家代码选择 docker-updater 镜像，带内存缓存。
// 国家代码为 CN 时使用 CNB 镜像源，其余情况使用默认镜像；探测失败时回退到默认镜像。
func resolveUpdaterImage(ctx context.Context) string {
	countryCacheMu.Lock()
	defer countryCacheMu.Unlock()

	if cachedUpdaterImage != "" && time.Since(countryCacheTime) < countryCacheDuration {
		return cachedUpdaterImage
	}

	image := updaterImageGlobal
	client := request.Client{Method: http.MethodGet, Url: countryCodeURL, Headers: request.Header{"User-Agent": "isrvd"}, Timeout: 8 * time.Second}
	code, err := client.RequestContext(ctx)
	if ctx.Err() != nil {
		return image
	}
	if err != nil {
		logman.Warn("country code detect failed", "error", err)
	} else if strings.EqualFold(strings.TrimSpace(string(code)), "CN") {
		image = updaterImageCN
	}

	cachedUpdaterImage, countryCacheTime = image, time.Now()
	return image
}
