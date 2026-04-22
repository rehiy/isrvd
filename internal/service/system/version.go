package system

import (
	"context"
	"encoding/json"
	"isrvd/config"
	"net/http"
	"strings"
	"time"

	"github.com/rehiy/pango/logman"
)

// VersionCheck 版本检测结果
type VersionCheck struct {
	Latest  string `json:"latest"`
	Update  bool   `json:"update"`
	Release string `json:"release,omitempty"`
}

var (
	cachedTag     string
	cachedURL     string
	cacheTime     time.Time
	cacheDuration = 4 * time.Hour
)

const checkURL = "https://api.github.com/repos/rehiy/isrvd/releases/latest"

// CheckVersion 检测版本更新
func (s *Service) CheckVersion(ctx context.Context) *VersionCheck {
	current := config.Version
	latest, releaseURL, err := fetchLatestTag(ctx)
	if err != nil {
		return &VersionCheck{}
	}

	return &VersionCheck{
		Latest:  latest,
		Update:  isNewerVersion(latest, current),
		Release: releaseURL,
	}
}

// fetchLatestTag 从 GitHub API 获取最新 Release 标签，带缓存
func fetchLatestTag(ctx context.Context) (tag, url string, err error) {
	if cachedTag != "" && time.Since(cacheTime) < cacheDuration {
		return cachedTag, cachedURL, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checkURL, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "isrvd-version-check")

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		logman.Warn("version check failed", "error", err)
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logman.Warn("version check failed", "status", resp.StatusCode)
		return "", "", nil
	}

	var r struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		logman.Warn("version check decode failed", "error", err)
		return "", "", err
	}

	cachedTag, cachedURL, cacheTime = r.TagName, r.HTMLURL, time.Now()
	return cachedTag, cachedURL, nil
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
