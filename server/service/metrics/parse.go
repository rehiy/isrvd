package metrics

import (
	"errors"
	"sort"
	"strconv"
	"strings"
)

// sample 单个原始指标样本。
type sample struct {
	name   string
	labels map[string]string
	value  float64
}

// key 使用指标名和排序后的完整标签，标签顺序变化不影响序列身份。
func (sm sample) key() string {
	keys := make([]string, 0, len(sm.labels))
	for key := range sm.labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString(sm.name)
	for _, key := range keys {
		b.WriteByte(',')
		b.WriteString(key)
		b.WriteByte('=')
		b.WriteString(strconv.Quote(sm.labels[key]))
	}
	return b.String()
}

// ─── 辅助函数 ───

var errInvalidMetricsText = errors.New("invalid metrics text")

// parseText 验证经典指标文本；允许仅有 HELP/TYPE 的无流量响应，不接受空白或异常页面。
// 通用指标可有 NaN/Inf，业务计数器的有限性由采样层检查。
func parseText(data []byte) ([]sample, error) {
	var out []sample
	metadata := false
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if line[0] == '#' {
			fields := strings.Fields(line)
			if len(fields) >= 3 && (fields[1] == "HELP" || fields[1] == "TYPE") && validMetricName(fields[2]) {
				metadata = true
			}
			continue
		}
		name, labels, rest, ok := splitSample(line)
		if !ok || !validMetricName(name) {
			return nil, errInvalidMetricsText
		}
		fields := strings.Fields(rest)
		if len(fields) < 1 || len(fields) > 2 {
			return nil, errInvalidMetricsText
		}
		value, err := strconv.ParseFloat(fields[0], 64)
		if err != nil {
			return nil, errInvalidMetricsText
		}
		if len(fields) == 2 {
			if _, err := strconv.ParseInt(fields[1], 10, 64); err != nil {
				return nil, errInvalidMetricsText
			}
		}
		parsed, err := parseLabels(labels)
		if err != nil {
			return nil, err
		}
		out = append(out, sample{name: name, labels: parsed, value: value})
	}
	if len(out) == 0 && !metadata {
		return nil, errInvalidMetricsText
	}
	return out, nil
}

// splitSample 把样本行拆为指标名、标签原文（不含花括号）及值。
func splitSample(line string) (name, labels, rest string, ok bool) {
	end := strings.IndexAny(line, "{ \t")
	if end <= 0 {
		return "", "", "", false
	}
	name = line[:end]
	if line[end] != '{' {
		return name, "", line[end:], true
	}
	inQuote, escaped := false, false
	for i := end + 1; i < len(line); i++ {
		c := line[i]
		switch {
		case escaped:
			escaped = false
		case c == '\\' && inQuote:
			escaped = true
		case c == '"':
			inQuote = !inQuote
		case c == '}' && !inQuote:
			return name, line[end+1 : i], line[i+1:], true
		}
	}
	return "", "", "", false
}

// parseLabels 验证完整标签，拒绝重复名称、非法转义和截断内容。
func parseLabels(text string) (map[string]string, error) {
	out := map[string]string{}
	for text = strings.TrimSpace(text); text != ""; {
		eq := strings.IndexByte(text, '=')
		if eq < 0 {
			return nil, errInvalidMetricsText
		}
		key := strings.TrimSpace(text[:eq])
		if !validMetricName(key) || strings.Contains(key, ":") {
			return nil, errInvalidMetricsText
		}
		if _, duplicate := out[key]; duplicate {
			return nil, errInvalidMetricsText
		}
		text = strings.TrimSpace(text[eq+1:])
		if len(text) == 0 || text[0] != '"' {
			return nil, errInvalidMetricsText
		}
		quoted, err := strconv.QuotedPrefix(text)
		if err != nil {
			return nil, errInvalidMetricsText
		}
		for i := 1; i < len(quoted)-1; i++ {
			if quoted[i] == '\\' {
				i++
				if !strings.ContainsRune(`\"n`, rune(quoted[i])) {
					return nil, errInvalidMetricsText
				}
			}
		}
		out[key], _ = strconv.Unquote(quoted)
		text = strings.TrimSpace(text[len(quoted):])
		if text == "" {
			break
		}
		if text[0] != ',' {
			return nil, errInvalidMetricsText
		}
		text = strings.TrimSpace(text[1:])
	}
	return out, nil
}

func validMetricName(name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c == '_' || c == ':' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || i > 0 && c >= '0' && c <= '9' {
			continue
		}
		return false
	}
	return true
}
