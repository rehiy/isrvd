package agui

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// Encoder 将 AG-UI 事件写入 SSE 流。
//
// @ag-ui/client 按空行切分帧，逐帧取以 "data:" 开头的行并 JSON 解析。
// 底层 writer（如 httpd.EventWriter）负责添加 "data:" 前缀与帧分隔，
// 这里只输出 JSON 载荷；若直接写入裸 ResponseWriter，请改用 WriteRaw。
type Encoder struct {
	w io.Writer
}

// NewEncoder 创建 AG-UI 事件编码器
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

// Write 序列化并写出单个事件（不带 data: 前缀，由底层 writer 添加）
func (e *Encoder) Write(event Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("编码 AG-UI 事件失败: %w", err)
	}
	if _, err := fmt.Fprintln(e.w, string(payload)); err != nil {
		return fmt.Errorf("写出 AG-UI 事件失败: %w", err)
	}
	return nil
}

// WriteComment 写出注释行，用于保持连接活跃。
// 仅对裸 ResponseWriter 有效；EventWriter 会将注释当作 data 载荷。
func (e *Encoder) WriteComment(text string) error {
	_, err := fmt.Fprintf(e.w, ": %s\n\n", text)
	return err
}

// Heartbeat 返回心跳 ticker，间隔为 d
func Heartbeat(d time.Duration) *time.Ticker {
	return time.NewTicker(d)
}
