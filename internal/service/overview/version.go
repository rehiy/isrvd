package overview

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/rehiy/libgo/logman"
	"github.com/rehiy/libgo/upgrade"

	"isrvd/config"
)

const upgradeServer = "https://isrvd.rehiy.com/update/"

// VersionCheck 版本检测结果
type VersionCheck struct {
	Latest  string `json:"latest"`
	Update  bool   `json:"update"`
	Release string `json:"release,omitempty"`
}

var (
	versionCacheMu sync.Mutex
	cachedTag      string
	cachedURL      string
	cacheTime      time.Time
	cacheDuration  = 1 * time.Hour
)

// CheckVersion 从升级服务器检测最新版本，带 4 小时缓存
func (s *Service) CheckVersion(ctx context.Context) *VersionCheck {
	current := config.Version
	latest, releaseURL := fetchLatestTag()

	return &VersionCheck{
		Latest:  latest,
		Update:  isNewerVersion(latest, current),
		Release: releaseURL,
	}
}

// fetchLatestTag 调用升级服务器获取最新版本号和 Release URL，带内存缓存
func fetchLatestTag() (tag, releaseURL string) {
	versionCacheMu.Lock()
	defer versionCacheMu.Unlock()

	if cachedTag != "" && time.Since(cacheTime) < cacheDuration {
		return cachedTag, cachedURL
	}

	info, err := upgrade.CheckUpdate(&upgrade.UpdateParam{
		Server:  upgradeServer,
		Version: config.Version,
	})
	if err != nil && (info == nil || info.Version == "") {
		logman.Warn("version check failed", "error", err)
		return cachedTag, cachedURL
	}

	if info != nil && info.Version != "" {
		cachedTag, cachedURL, cacheTime = info.Version, info.Release, time.Now()
	}

	return cachedTag, cachedURL
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
