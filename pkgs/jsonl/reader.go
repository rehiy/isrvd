package jsonl

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"os"
	"time"

	"github.com/rehiy/libgo/logman"
)

// Filter 在不反序列化整个对象的前提下，对单行 JSON 做快速过滤判断。
// 返回 true 表示保留该行；false 表示跳过。
type Filter func(line []byte) bool

// LineHandler 流式回调；返回 false 表示提前结束扫描。
type LineHandler func(line []byte) bool

// scanFile 对指定文件按行扫描，filter 可为 nil（不过滤）。
// 内部使用 bufio.Scanner，避免一次性加载整个文件到内存。
// handler 接收的 []byte 仅在回调内有效，下次扫描会复用底层缓冲区。
func scanFile(path string, filter Filter, handler LineHandler) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	// 单行最大 1MB，足够覆盖审计/监控记录
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		if filter != nil && !filter(line) {
			continue
		}
		if !handler(line) {
			return nil
		}
	}
	return scanner.Err()
}

// ScanSince 按 cutoffUnix 过滤后，从老到新流式扫描多天文件。
// tsField 为时间戳字段名（如 "ts"、"timestamp"）；当字段为 RFC3339 字符串时
// 会自动解析为 Unix 秒。extra 为额外过滤条件（可为 nil）；limit<=0 表示不限制。
func ScanSince(s *Store, cutoffUnix int64, tsField string, extra Filter, limit int, handler LineHandler) error {
	dates := daysInRange(cutoffUnix, time.Now().Unix(), s.opts.loc)
	combined := and(tsGTEFilter(tsField, cutoffUnix), extra)

	count := 0
	for _, date := range dates {
		if limit > 0 && count >= limit {
			return nil
		}
		path := s.FilePath(date)
		err := scanFile(path, combined, func(line []byte) bool {
			if !handler(line) {
				return false
			}
			count++
			return !(limit > 0 && count >= limit)
		})
		if err != nil {
			logman.Warn("jsonl: scan file failed", "path", path, "error", err)
			continue
		}
	}
	return nil
}

// DecodeSince 是 ScanSince 的泛型便捷形式，将匹配行直接反序列化为 []T。
func DecodeSince[T any](s *Store, cutoffUnix int64, tsField string, extra Filter, limit int) ([]T, error) {
	var out []T
	err := ScanSince(s, cutoffUnix, tsField, extra, limit, func(line []byte) bool {
		var v T
		if err := json.Unmarshal(line, &v); err != nil {
			return true // 跳过坏行
		}
		out = append(out, v)
		return true
	})
	return out, err
}

// tailLines 从文件末尾向前读最近 n 条匹配的行，返回顺序为"由新到旧"。
// 实现：以 chunkSize 块从文件尾部反向读取，按 \n 切分；不会一次性加载整个文件。
func tailLines(path string, n int, filter Filter) ([][]byte, error) {
	if n <= 0 {
		return nil, nil
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := stat.Size()
	if size == 0 {
		return nil, nil
	}

	const chunkSize = 32 * 1024
	buf := make([]byte, chunkSize)
	var carry []byte // 块边界遗留（行未读完）的字节
	result := make([][]byte, 0, n)

	pos := size
	for pos > 0 && len(result) < n {
		readSize := int64(chunkSize)
		if pos < readSize {
			readSize = pos
		}
		pos -= readSize
		if _, err := f.ReadAt(buf[:readSize], pos); err != nil && !errors.Is(err, io.EOF) {
			return nil, err
		}
		// 拼接：本次块 + 上次遗留头部
		merged := make([]byte, 0, int(readSize)+len(carry))
		merged = append(merged, buf[:readSize]...)
		merged = append(merged, carry...)

		// 反向按 \n 切分
		end := len(merged)
		for i := len(merged) - 1; i >= 0; i-- {
			if merged[i] != '\n' {
				continue
			}
			line := merged[i+1 : end]
			end = i
			if len(line) == 0 {
				continue
			}
			if filter != nil && !filter(line) {
				continue
			}
			dup := make([]byte, len(line))
			copy(dup, line)
			result = append(result, dup)
			if len(result) >= n {
				return result, nil
			}
		}
		// end 之前的内容（不含 \n 边界的剩余首段）作为下次的 carry
		carry = make([]byte, end)
		copy(carry, merged[:end])

		// 文件起始：carry 即为第一行
		if pos == 0 && len(carry) > 0 {
			line := carry
			carry = nil
			if filter == nil || filter(line) {
				dup := make([]byte, len(line))
				copy(dup, line)
				result = append(result, dup)
			}
		}
	}
	return result, nil
}

// DecodeTail 是 tailLines 的泛型便捷形式
func DecodeTail[T any](path string, n int, filter Filter) ([]T, error) {
	lines, err := tailLines(path, n, filter)
	if err != nil {
		return nil, err
	}
	out := make([]T, 0, len(lines))
	for _, line := range lines {
		var v T
		if json.Unmarshal(line, &v) == nil {
			out = append(out, v)
		}
	}
	return out, nil
}

// DecodeTailDays 从 Store 当前日期起向前回扫最多 days 天，每天倒序取行，
// 直到收集到 limit 条命中 filter 的记录。返回顺序为"由新到旧"。
// 适用于"按业务字段过滤、跨天倒序查询最近 N 条"的场景（如计划任务执行历史）。
func DecodeTailDays[T any](s *Store, days, limit int, filter Filter) ([]T, error) {
	if s == nil || days <= 0 || limit <= 0 {
		return nil, nil
	}
	out := make([]T, 0, limit)
	now := time.Now().In(s.opts.loc)
	for back := 0; back < days && len(out) < limit; back++ {
		date := now.AddDate(0, 0, -back).Format(dateLayout)
		path := s.FilePath(date)
		lines, err := tailLines(path, limit-len(out), filter)
		if err != nil {
			logman.Warn("jsonl: tail file failed", "path", path, "error", err)
			continue
		}
		for _, line := range lines {
			var v T
			if json.Unmarshal(line, &v) == nil {
				out = append(out, v)
			}
		}
	}
	return out, nil
}
