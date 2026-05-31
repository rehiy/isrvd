package jsonl

import (
	"time"

	"github.com/tidwall/gjson"
)

// and 组合多个 Filter，全部通过才视为命中。nil 元素会被忽略；
// 全部为 nil 时返回 nil（表示不过滤）。
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

// StrEq 判断 path 字段（gjson 路径）等于 value（区分大小写）。
// path 为空、value 为空时不过滤（返回 nil）。
func StrEq(path, value string) Filter {
	if path == "" || value == "" {
		return nil
	}
	return func(line []byte) bool {
		return gjson.GetBytes(line, path).String() == value
	}
}

// tsGTEFilter 内部使用：根据字段类型自动适配 ts 比较。
// 字段为数字时按 Unix 秒比较；字段为 RFC3339(Nano) 字符串时解析后比较。
// 字段不存在或解析失败时返回 false（视为不命中）。
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
			// RFC3339Nano 的 layout 同时兼容标准 RFC3339（无小数秒）
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
