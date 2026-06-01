package jsonl

import (
	"time"

	"github.com/tidwall/gjson"
)

// Filter 行过滤函数，返回 true 保留。
type Filter func(line []byte) bool

// LineHandler 流式处理回调，返回 false 提前结束。
type LineHandler func(line []byte) bool

// And 组合多个 Filter，全部通过才保留；nil 被忽略；全 nil 返回 nil。
func And(filters ...Filter) Filter {
	var fs []Filter
	for _, f := range filters {
		if f != nil {
			fs = append(fs, f)
		}
	}

	switch len(fs) {
	case 0:
		return nil
	case 1:
		return fs[0]
	}
	return func(line []byte) bool {
		for _, f := range fs {
			if !f(line) {
				return false
			}
		}
		return true
	}
}

// StrEq 字符串相等过滤器，path 或 value 为空返回 nil。
func StrEq(path, value string) Filter {
	if path == "" || value == "" {
		return nil
	}
	return func(line []byte) bool {
		return gjson.GetBytes(line, path).String() == value
	}
}

// tsGTEFilter 时间戳 >= 过滤器，支持 Unix 秒和 RFC3339 字符串。
// 字段不存在或解析失败返回 false。
func tsGTEFilter(path string, cutoffUnix int64) Filter {
	if path == "" {
		return nil
	}
	return func(line []byte) bool {
		ts, ok := ParseTimestamp(line, path)
		return ok && ts >= cutoffUnix
	}
}

// ParseTimestamp 从 JSONL 行中读取指定字段并解析为 Unix 秒。
// 支持 Unix 秒（number）和 RFC3339 字符串两种格式；字段不存在或解析失败返回 false。
func ParseTimestamp(line []byte, path string) (int64, bool) {
	if path == "" {
		return 0, false
	}
	v := gjson.GetBytes(line, path)
	switch v.Type {
	case gjson.Number:
		return v.Int(), true
	case gjson.String:
		t, err := time.Parse(time.RFC3339Nano, v.Str)
		if err != nil {
			return 0, false
		}
		return t.Unix(), true
	default:
		return 0, false
	}
}
