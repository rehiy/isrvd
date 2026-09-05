package agui

import (
	"encoding/json"
	"fmt"
	"io"
)

// Encoder 将 AG-UI 事件写入 SSE 流。
//
// @ag-ui/client 按空行切分帧，逐帧取以 "data:" 开头的行并 JSON 解析。
// 底层 writer（如 httpd.EventWriter）负责添加 "data:" 前缀与帧分隔，
// 这里只输出 JSON 载荷。
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
