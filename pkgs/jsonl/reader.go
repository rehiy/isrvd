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

// Filter 行过滤函数，返回 true 保留。
type Filter func(line []byte) bool

// LineHandler 流式处理回调，返回 false 提前结束。
type LineHandler func(line []byte) bool

// scanFile 按行扫描文件，filter 为 nil 不过滤；handler 返回 false 提前结束。
// 注意：handler 中的 []byte 仅在本次回调有效。
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
	scanner.Buffer(make([]byte, 64*1024), 1024*1024) // 最大 1MB

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

// ScanSince 从 cutoffUnix 起扫描多天文件，tsField 支持 Unix 秒和 RFC3339。
// extra 可为 nil；limit<=0 不限制。
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
			// 达到限制时提前终止
			return !(limit > 0 && count >= limit)
		})

		if err != nil {
			logman.Warn("jsonl: scan file failed", "path", path, "error", err)
			continue
		}
	}
	return nil
}

// DecodeSince 泛型封装，将匹配行反序列化为 []T，跳过坏行。
func DecodeSince[T any](s *Store, cutoffUnix int64, tsField string, extra Filter, limit int) ([]T, error) {
	var out []T
	err := ScanSince(s, cutoffUnix, tsField, extra, limit, func(line []byte) bool {
		var v T
		if err := json.Unmarshal(line, &v); err != nil {
			return true // 跳过解析失败的行
		}
		out = append(out, v)
		return true
	})
	return out, err
}

// tailLines 从文件末尾向前读最近 n 条匹配行，返回"由新到旧"。
// 以 32KB 块反向读取，避免全量加载。
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

	stat, _ := f.Stat()
	size := stat.Size()
	if size == 0 {
		return nil, nil
	}

	const chunkSize = 32 * 1024
	buf := make([]byte, chunkSize)
	var carry []byte
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

		merged := make([]byte, 0, int(readSize)+len(carry))
		merged = append(merged, buf[:readSize]...)
		merged = append(merged, carry...)

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
		carry = make([]byte, end)
		copy(carry, merged[:end])

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

// DecodeTail 泛型封装，将匹配行反序列化为 []T。
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

// DecodeTailDays 从当前日期向前回扫 days 天，倒序取行直到达到 limit。
// 适用于跨天查询最近 N 条记录（如计划任务执行历史）。
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
