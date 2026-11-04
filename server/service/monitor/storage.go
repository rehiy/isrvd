package monitor

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
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

// ErrMonitorUnavailable 监控存储未就绪。
var ErrMonitorUnavailable = errors.New("监控采集器未启动")

// recordIndex 聚合各容器最后保存记录，文件读取由 JSONL 游标负责。
type recordIndex struct {
	gate   chan struct{}
	cursor jsonl.Cursor
	latest map[string]Record
}

func newRecordIndex() *recordIndex {
	return &recordIndex{
		gate:   make(chan struct{}, 1),
		latest: make(map[string]Record),
	}
}

func (i *recordIndex) read(ctx context.Context, s *jsonl.Store) ([]Record, error) {
	if s == nil {
		return nil, ErrMonitorUnavailable
	}
	select {
	case i.gate <- struct{}{}:
		defer func() { <-i.gate }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	cutoff := time.Now().AddDate(0, 0, -retainDays).Unix()
	err := i.cursor.ScanSince(ctx, s, cutoff, "ts", nil, func(line []byte) bool {
		var record Record
		if json.Unmarshal(line, &record) == nil && record.ContainerID != "" && len(record.Data) > 0 && string(record.Data) != "null" {
			i.latest[record.ContainerID] = record
		}
		return true
	})
	if err != nil {
		return nil, err
	}

	records := make([]Record, 0, len(i.latest))
	for id, record := range i.latest {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if record.Ts < cutoff {
			delete(i.latest, id)
		} else {
			records = append(records, record)
		}
	}
	slices.SortFunc(records, func(a, b Record) int { return strings.Compare(a.ContainerID, b.ContainerID) })
	return records, ctx.Err()
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

// appendRecord 追加一条监控记录；store 为 nil 时忽略
func appendRecord(s *jsonl.Store, record *Record) {
	if s == nil {
		return
	}
	if err := s.Append(record); err != nil {
		logman.Warn("monitor: write record failed", "error", err)
	}
}

// readLatest 跨保留日期尾读最后一条有效记录。
func readLatest(s *jsonl.Store, containerID string) (*Record, error) {
	if s == nil {
		return nil, ErrMonitorUnavailable
	}
	limit := 1
	if containerID != "" && len(containerID) < 64 {
		limit = 2 // 短 ID 必须只匹配一个完整容器 ID
	}
	seen := make(map[string]bool)
	records, err := s.DecodeTailDays[Record](retainDays+1, limit, func(line []byte) bool {
		var record Record
		if json.Unmarshal(line, &record) != nil || record.Ts <= 0 || len(record.Data) == 0 || string(record.Data) == "null" || seen[record.ContainerID] {
			return false
		}
		if record.ContainerID != containerID && (limit == 1 || !strings.HasPrefix(record.ContainerID, containerID)) {
			return false
		}
		seen[record.ContainerID] = true
		return true
	})
	if err != nil || len(records) == 0 {
		return nil, err
	}
	if len(records) > 1 {
		return nil, errors.New("容器 ID 不明确")
	}
	return &records[0], nil
}

// readSince 读取 ts >= (now-sinceSeconds) 的记录，containerID 非空时只返回指定容器；
// 结果按时间窗口降采样到 samplePoints 左右
func readSince(s *jsonl.Store, containerID string, sinceSeconds int64) ([]Record, error) {
	if s == nil {
		return nil, nil
	}
	cutoff := time.Now().Unix() - sinceSeconds
	return s.DecodeSinceSampled[Record](cutoff, "ts", jsonl.StrEq("container_id", containerID), samplePoints)
}

// readContainerNameSince 按原始记录中的名称查询不同实例的历史，保留各实例 ID。
func readContainerNameSince(s *jsonl.Store, name string, sinceSeconds int64) ([]Record, error) {
	if s == nil {
		return nil, nil
	}
	name = strings.TrimPrefix(name, "/")
	if name == "" {
		return nil, nil
	}
	filter := func(line []byte) bool {
		var record struct {
			ContainerID string `json:"container_id"`
			Data        struct {
				Name string `json:"name"`
			} `json:"data"`
		}
		return json.Unmarshal(line, &record) == nil && record.ContainerID != "" && strings.TrimPrefix(record.Data.Name, "/") == name
	}
	return s.DecodeSinceSampled[Record](time.Now().Unix()-sinceSeconds, "ts", filter, samplePoints)
}

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
