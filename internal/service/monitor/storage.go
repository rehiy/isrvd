package monitor

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/rehiy/libgo/logman"

	"isrvd/pkgs/jsonl"
)

const (
	// retainDays 文件保留天数
	retainDays = 3
	// HostPrefix 主机监控文件前缀
	HostPrefix = "host"
	// ContainerPrefix 容器监控文件前缀
	ContainerPrefix = "ctr"
	// fileSuffix 监控文件后缀
	fileSuffix = ".jsonl"
)

// stores 缓存 dir+prefix -> Store，避免同一前缀重复打开句柄
var (
	storesMu sync.Mutex
	stores   = make(map[string]*jsonl.Store)
)

// getStore 获取或创建指定 (dir, prefix) 的 Store
func getStore(dir, prefix string) *jsonl.Store {
	key := dir + "/" + prefix
	storesMu.Lock()
	defer storesMu.Unlock()

	if s, ok := stores[key]; ok {
		return s
	}
	s, err := jsonl.New(dir, jsonl.Naming{Prefix: prefix, Sep: "_", Suffix: fileSuffix})
	if err != nil {
		logman.Warn("monitor: open jsonl store failed", "dir", dir, "prefix", prefix, "error", err)
		return nil
	}
	stores[key] = s
	return s
}

// AppendRawRecord 直接追加已序列化的数据（避免重复序列化）
// containerID 为空时表示主机监控
func AppendRawRecord(dir, prefix, containerID string, ts int64, raw json.RawMessage) {
	s := getStore(dir, prefix)
	if s == nil {
		return
	}
	if err := s.Append(&Record{Ts: ts, Data: raw, ContainerID: containerID}); err != nil {
		logman.Warn("monitor: write record failed", "prefix", prefix, "error", err)
	}
}

// ReadSince 读取 dir 下 prefix_*.jsonl 中 ts >= (now-sinceSeconds) 的所有行
// 按时间窗口确定需要读哪几天的文件，合并后按 ts 顺序返回
// containerID 为空时返回所有记录，非空时只返回指定容器的记录
// limit <= 0 表示不限制条数
func ReadSince[T any](dir, prefix, containerID string, sinceSeconds int64, limit int) ([]T, error) {
	s := getStore(dir, prefix)
	if s == nil {
		return nil, nil
	}
	cutoff := time.Now().Unix() - sinceSeconds
	extra := jsonl.StrEq("container_id", containerID)
	return jsonl.DecodeSince[T](s, cutoff, "ts", extra, limit)
}

// CleanOldFiles 删除 dir 下所有 *_YYYY-MM-DD.jsonl 中超过 retainDays 天的旧文件
func CleanOldFiles(dir string) {
	for _, prefix := range []string{HostPrefix, ContainerPrefix} {
		jsonl.CleanOlderThan(
			dir,
			jsonl.Naming{Prefix: prefix, Sep: "_", Suffix: fileSuffix},
			retainDays,
		)
	}
}
