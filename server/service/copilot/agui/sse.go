package agui

import "strings"

// splitSSEEvent 按 SSE 空行分隔事件，兼容 LF、CRLF 与 CR 三种行终止符。
func splitSSEEvent(data []byte, atEOF bool) (advance int, token []byte, err error) {
	for i := 0; i < len(data); i++ {
		if data[i] != '\n' && data[i] != '\r' {
			continue
		}
		width := 1
		if data[i] == '\r' {
			if i+1 == len(data) && !atEOF {
				return 0, nil, nil
			}
			if i+1 < len(data) && data[i+1] == '\n' {
				width = 2
			}
		}
		next := i + width
		if next == len(data) {
			break
		}
		if data[next] == '\n' || data[next] == '\r' {
			width2 := 1
			if data[next] == '\r' {
				if next+1 == len(data) && !atEOF {
					return 0, nil, nil
				}
				if next+1 < len(data) && data[next+1] == '\n' {
					width2 = 2
				}
			}
			return next + width2, data[:i], nil
		}
		i = next - 1
	}
	if atEOF && len(data) > 0 {
		return len(data), data, nil
	}
	return 0, nil, nil
}

// ssePayload 把一个 SSE 事件中的多个 data 字段按标准以换行拼接，忽略注释和其他字段。
func ssePayload(frame string) string {
	frame = strings.ReplaceAll(strings.ReplaceAll(frame, "\r\n", "\n"), "\r", "\n")
	lines := strings.Split(frame, "\n")
	var parts []string
	for _, line := range lines {
		if line == "data" {
			parts = append(parts, "")
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		value := strings.TrimPrefix(line, "data:")
		value = strings.TrimPrefix(value, " ")
		parts = append(parts, value)
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}
