package monitor

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rehiy/libgo/logman"
)

const (
	// retainDays 文件保留天数
	retainDays = 3
	// HostPrefix 主机监控文件前缀
	HostPrefix = "host"
	// ContainerPrefix 容器监控文件前缀
	ContainerPrefix = "ctr"
)

// store 管理单个前缀的按天 NDJSON 文件追加写
// 每天自动轮转到新文件，无需 compact
type store struct {
	mu     sync.Mutex
	dir    string
	prefix string
	date   string // 当前文件对应的日期（YYYY-MM-DD）
	file   *os.File
}

var (
	storesMu sync.Mutex
	stores   = make(map[string]*store) // key: dir+"/"+prefix
)

// getStore 获取或创建指定前缀的 store
func getStore(dir, prefix string) (*store, error) {
	storesMu.Lock()
	defer storesMu.Unlock()

	key := dir + "/" + prefix
	if s, ok := stores[key]; ok {
		return s, nil
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	s := &store{dir: dir, prefix: prefix}
	stores[key] = s
	return s, nil
}

// filePath 返回指定日期的文件路径
func (s *store) filePath(date string) string {
	return filepath.Join(s.dir, fmt.Sprintf("%s_%s.ndjson", s.prefix, date))
}

// ensureFile 确保当前文件句柄对应今天的日期，日期变更时自动轮转
func (s *store) ensureFile() error {
	today := time.Now().In(time.Local).Format("2006-01-02")
	if s.file != nil && s.date == today {
		return nil
	}
	// 关闭旧句柄
	if s.file != nil {
		s.file.Close()
		s.file = nil
	}
	f, err := os.OpenFile(s.filePath(today), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	s.file = f
	s.date = today
	return nil
}

// append 将一条记录追加写入当天文件（一行 JSON）
func (s *store) append(v any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureFile(); err != nil {
		return err
	}

	line, err := json.Marshal(v)
	if err != nil {
		return err
	}
	line = append(line, '\n')

	_, err = s.file.Write(line)
	return err
}

// ReadSince 读取 dir 下 prefix_*.ndjson 中 ts >= (now-sinceSeconds) 的所有行
// 按时间窗口确定需要读哪几天的文件，合并后按 ts 顺序返回
func ReadSince[T any](dir, prefix string, sinceSeconds int64) ([]T, error) {
	cutoff := time.Now().Unix() - sinceSeconds

	// 确定需要读取的日期列表（从最早到今天）
	days := daysInRange(cutoff)

	var result []T
	for _, date := range days {
		path := filepath.Join(dir, fmt.Sprintf("%s_%s.ndjson", prefix, date))
		recs, err := readFileRecords[T](path, cutoff)
		if err != nil {
			logman.Warn("monitor: read file failed", "path", path, "error", err)
			continue
		}
		result = append(result, recs...)
	}
	return result, nil
}

// CleanOldFiles 删除 dir 下所有 *_YYYY-MM-DD.ndjson 中超过 retainDays 天的旧文件
func CleanOldFiles(dir string) {
	cutoffDate := time.Now().AddDate(0, 0, -retainDays).Format("2006-01-02")
	pattern := filepath.Join(dir, "*_????-??-??.ndjson")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return
	}
	for _, path := range matches {
		base := filepath.Base(path)
		// 提取末尾日期部分：任意前缀_YYYY-MM-DD.ndjson
		if len(base) < len("_2006-01-02.ndjson") {
			continue
		}
		dateStr := base[len(base)-len("2006-01-02.ndjson") : len(base)-len(".ndjson")]
		if dateStr <= cutoffDate {
			if err := os.Remove(path); err == nil {
				logman.Info("monitor: removed old file", "path", path)
			}
		}
	}
}

// AppendRecord 将任意数据序列化后追加到 dir/prefix_YYYY-MM-DD.ndjson
// 内部自动完成 JSON 序列化，写入失败时记录日志
func AppendRecord(dir, prefix string, ts int64, data any) {
	raw, err := json.Marshal(data)
	if err != nil {
		logman.Warn("monitor: marshal data failed", "prefix", prefix, "error", err)
		return
	}
	record := &Record{Ts: ts, Data: raw}
	s, err := getStore(dir, prefix)
	if err != nil {
		logman.Warn("monitor: get store failed", "prefix", prefix, "error", err)
		return
	}
	if err := s.append(record); err != nil {
		logman.Warn("monitor: write record failed", "prefix", prefix, "error", err)
	}
}

// daysInRange 返回从 cutoffUnix 所在日期到今天的日期字符串列表（YYYY-MM-DD，本地时区）
func daysInRange(cutoffUnix int64) []string {
	loc := time.Local
	t := time.Unix(cutoffUnix, 0).In(loc)
	start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	var days []string
	for d := start; !d.After(today); d = d.AddDate(0, 0, 1) {
		days = append(days, d.Format("2006-01-02"))
	}
	return days
}

// readFileRecords 读取单个文件中 ts >= cutoff 的所有记录
func readFileRecords[T any](path string, cutoff int64) ([]T, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var result []T
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var ts struct {
			Ts int64 `json:"ts"`
		}
		if json.Unmarshal(line, &ts) != nil || ts.Ts < cutoff {
			continue
		}
		var rec T
		if json.Unmarshal(line, &rec) == nil {
			result = append(result, rec)
		}
	}
	if err := scanner.Err(); err != nil {
		logman.Warn("monitor: scanner error", "path", path, "error", err)
	}
	return result, nil
}
