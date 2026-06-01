package jsonl

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// dateLayout 日期格式（YYYY-MM-DD）
const dateLayout = "2006-01-02"

// CleanOlderThan
// CleanOlderThan 删除早于 retainDays 天的匹配文件。
func CleanOlderThan(dir string, naming Naming, retainDays int) error {
	if retainDays <= 0 {
		return nil
	}
	cutoffDate := time.Now().AddDate(0, 0, -retainDays).Format(dateLayout)
	pattern := filepath.Join(dir, naming.Prefix+naming.Sep+"????-??-??"+naming.Suffix)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("jsonl: glob %s: %w", pattern, err)
	}

	var errs []error
	for _, path := range matches {
		base := filepath.Base(path)
		mid := strings.TrimSuffix(base, naming.Suffix)
		if naming.Prefix != "" {
			mid = strings.TrimPrefix(mid, naming.Prefix+naming.Sep)
		}
		if len(mid) != len(dateLayout) || mid > cutoffDate {
			continue
		}
		if err := os.Remove(path); err != nil {
			errs = append(errs, fmt.Errorf("remove %s: %w", path, err))
		}
	}
	return errors.Join(errs...)
}

// daysInRange 返回 [fromUnix, toUnix] 范围内的所有日期（YYYY-MM-DD）。
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
