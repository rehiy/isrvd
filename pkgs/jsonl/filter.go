package jsonl

import (
	"time"

	"github.com/tidwall/gjson"
)

// and 组合多个 Filter，全部通过才保留；nil 被忽略；全 nil 返回 nil。
func and(filters ...Filter) Filter {
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
		v := gjson.GetBytes(line, path)
		switch v.Type {
		case gjson.Number:
			return v.Int() >= cutoffUnix
		case gjson.String:

			t, err := time.Parse(time.RFC3339Nano, v.Str)
			if err != nil {
				return false
			}
			return t.Unix() >= cutoffUnix
		default:
			return false
		}
	}
}
