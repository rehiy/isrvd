package overview

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/rehiy/libgo/logman"
	"github.com/rehiy/libgo/request"
	"github.com/rehiy/libgo/upgrade"

	"isrvd/server/config"
)

// 本文件负责二进制自升级：后台任务的生命周期与发布包（tar.gz）解包。
// 检查、下载、校验、替换及回滚复用 libgo；升级源与站点类型见 version.go（本文件单向依赖它）。

// executablePath 启动时记录，避免升级替换后 /proc/self/exe 失效
var executablePath, _ = os.Executable()

// selfUpgrade 跨配置重载保留（Service 会随重载重建）：同一时间只有一个升级任务；
// 服务关闭后拒绝新的升级与重启，并等待进行中的任务结束，确保不打断文件替换。
var selfUpgrade struct {
	sync.Mutex
	stopped bool
	cancel  context.CancelFunc // 非 nil 表示有任务在执行
	wg      sync.WaitGroup
}

// StartSelfUpgrade 在后台升级并重启，重复请求不再启动第二次升级。
func (s *Service) StartSelfUpgrade() error {
	selfUpgrade.Lock()
	defer selfUpgrade.Unlock()
	if selfUpgrade.stopped {
		return context.Canceled
	}
	if selfUpgrade.cancel != nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 270*time.Second)
	selfUpgrade.cancel = cancel
	selfUpgrade.wg.Add(1)
	go func() {
		defer selfUpgrade.wg.Done()
		defer func() {
			cancel()
			selfUpgrade.Lock()
			selfUpgrade.cancel = nil
			selfUpgrade.Unlock()
		}()

		logger := logman.Named("upgrade")
		if err := applySelfUpgrade(ctx); err != nil {
			if errors.Is(err, upgrade.ErrNoUpdate) {
				logger.Info("no update available")
			} else {
				logger.Error("upgrade failed", "error", err)
			}
			return
		}
		time.Sleep(500 * time.Millisecond) // 让升级请求的响应先送出

		// 持锁检查，使关闭与重启不会交错；重启成功时进程被替换，不会返回。
		selfUpgrade.Lock()
		defer selfUpgrade.Unlock()
		if selfUpgrade.stopped {
			return
		}
		u := upgrade.NewUpdater("", "")
		u.TargetPath = executablePath
		if err := u.Restart(); err != nil {
			logger.Error("upgrade restart failed", "error", err)
		}
	}()
	return nil
}

// StopSelfUpgrade 取消升级并等待实际工作退出，之后不再接受升级。
func (s *Service) StopSelfUpgrade() {
	selfUpgrade.Lock()
	selfUpgrade.stopped = true
	if selfUpgrade.cancel != nil {
		selfUpgrade.cancel()
	}
	selfUpgrade.Unlock()
	selfUpgrade.wg.Wait()
}

// applySelfUpgrade 只有发布包的 tar.gz 解包是自己实现的，其余复用 libgo。
func applySelfUpgrade(ctx context.Context) error {
	if executablePath == "" {
		return errors.New("获取可执行文件路径失败")
	}
	binaryName := "isrvd-" + runtime.GOOS + "-" + runtime.GOARCH
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	server := upgradeServer
	if inChina(ctx) {
		server += "?area=cn" // 通知升级服务优先下发国内镜像
	}
	u := upgrade.NewUpdaterContext(ctx, server, config.Version)
	u.TargetPath = executablePath
	u.Download = func(pkgURL, outputPath string) (string, error) {
		res, err := (&request.Client{Method: http.MethodGet, Url: pkgURL}).DoContext(ctx, http.DefaultClient)
		if err != nil {
			return "", err
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			return "", fmt.Errorf("download returned HTTP %d", res.StatusCode)
		}
		gz, err := gzip.NewReader(res.Body)
		if err != nil {
			return "", err
		}
		defer gz.Close()
		tr := tar.NewReader(io.LimitReader(gz, 512<<20)) // 解压后体积上限 512 MiB
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				return "", fmt.Errorf("archive: %s not found", binaryName)
			}
			if err != nil {
				return "", err
			}
			if hdr.Typeflag == tar.TypeReg && filepath.Base(hdr.Name) == binaryName {
				return request.SaveStream(tr, outputPath)
			}
		}
	}
	return u.Apply()
}
