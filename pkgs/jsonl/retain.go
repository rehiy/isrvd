package jsonl

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rehiy/libgo/logman"
)

// dateLayout 文件名中日期的固定格式
const dateLayout = "2006-01-02"

// daysInRange 返回 [fromUnix, toUnix] 涵盖的所有日期字符串（YYYY-MM-DD），按时间从老到新排列。
func daysInRange(fromUnix, toUnix int64, loc *time.Location) []string {
	if loc == nil {
		loc = time.Local
	}
	if toUnix < fromUnix {
		toUnix = fromUnix
	}
	from := time.Unix(fromUnix, 0).In(loc)
	to := time.Unix(toUnix, 0).In(loc)
	start := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, loc)
	end := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, loc)
	var days []string
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		days = append(days, d.Format(dateLayout))
	}
	return days
}

// CleanOlderThan 删除 dir 下命名匹配 naming 且日期早于 retainDays 天前的文件。
func CleanOlderThan(dir string, naming Naming, retainDays int) {
	if retainDays <= 0 {
		return
	}
	cutoffDate := time.Now().AddDate(0, 0, -retainDays).Format(dateLayout)

	pattern := filepath.Join(dir, naming.Prefix+naming.Sep+"????-??-??"+naming.Suffix)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return
	}
	for _, path := range matches {
		base := filepath.Base(path)
		mid := strings.TrimSuffix(base, naming.Suffix)
		if naming.Prefix != "" {
			mid = strings.TrimPrefix(mid, naming.Prefix+naming.Sep)
		}
		// 此时 mid 应为 YYYY-MM-DD
		if len(mid) != len(dateLayout) || mid > cutoffDate {
			continue
		}
		if err := os.Remove(path); err == nil {
			logman.Info("jsonl: removed old file", "path", path)
		}
	}
}
