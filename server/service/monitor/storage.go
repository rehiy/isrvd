package monitor

import (
	"time"

	"github.com/rehiy/libgo/jsonl"
	"github.com/rehiy/libgo/logman"
)

const (
	// retainDays 文件保留天数
	retainDays = 3
	// samplePoints 监控历史查询目标返回点数
	samplePoints = 300
	// HostPrefix 主机监控文件前缀
	HostPrefix = "host"
	// ContainerPrefix 容器监控文件前缀
	ContainerPrefix = "ctr"
)

// appendRecord 追加一条监控记录；store 为 nil 时忽略
func appendRecord(s *jsonl.Store, record *Record) {
	if s == nil {
		return
	}
	if err := s.Append(record); err != nil {
		logman.Warn("monitor: write record failed", "error", err)
	}
}

// readSince 读取 ts >= (now-sinceSeconds) 的记录，containerID 非空时只返回指定容器；
// 结果按时间窗口降采样到 samplePoints 左右
func readSince(s *jsonl.Store, containerID string, sinceSeconds int64) ([]Record, error) {
	if s == nil {
		return nil, nil
	}
	cutoff := time.Now().Unix() - sinceSeconds
	return jsonl.DecodeSinceSampled[Record](s, cutoff, "ts", jsonl.StrEq("container_id", containerID), samplePoints)
}

// CleanOldFiles 删除 dir 下所有 *_YYYY-MM-DD.jsonl 中超过 retainDays 天的旧文件
func CleanOldFiles(dir string) {
	for _, prefix := range []string{HostPrefix, ContainerPrefix} {
		if err := jsonl.CleanOlderThan(dir, storeNaming(prefix), retainDays); err != nil {
			logman.Warn("monitor: clean old files failed", "dir", dir, "prefix", prefix, "error", err)
		}
	}
}

// ─── 辅助函数 ───

// openStore 打开指定前缀的监控数据存储，失败时返回 nil
func openStore(dir, prefix string) *jsonl.Store {
	s, err := jsonl.New(dir, storeNaming(prefix),
		jsonl.WithBufferSize(32*1024),          // 32KB 缓冲，减少 flush 次数
		jsonl.WithAsync(256),                   // 异步写入，采集 goroutine 不被 IO 阻塞
		jsonl.WithFlushInterval(5*time.Second), // 5s flush 一次，与最短采集间隔对齐
	)
	if err != nil {
		logman.Warn("monitor: open jsonl store failed", "dir", dir, "prefix", prefix, "error", err)
		return nil
	}
	return s
}

// closeStore 刷盘并关闭存储
func closeStore(s *jsonl.Store) {
	if s == nil {
		return
	}
	if err := s.Close(); err != nil {
		logman.Warn("monitor: close jsonl store failed", "error", err)
	}
}

func storeNaming(prefix string) jsonl.Naming {
	return jsonl.Naming{Prefix: prefix, Sep: "_", Suffix: ".jsonl"}
}
