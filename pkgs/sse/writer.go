package sse

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// NewWriter 初始化 SSE 响应头并返回按 data 事件写入的 writer。
func NewWriter(w http.ResponseWriter) (io.Writer, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("当前响应不支持流式输出")
	}

	header := w.Header()
	header.Set("Content-Type", "text/event-stream; charset=utf-8")
	header.Set("Cache-Control", "no-cache")
	header.Set("Connection", "keep-alive")
	header.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	return &writer{w: w, flusher: flusher}, nil
}

// NewErrorWriter 与 NewWriter 相同，但额外支持按 event 类型写入。
type Writer interface {
	io.Writer
	WriteEvent(event, data string) error
}

// NewEventWriter 返回支持 WriteEvent 的扩展 writer。
func NewEventWriter(w http.ResponseWriter) (Writer, error) {
	raw, err := NewWriter(w)
	if err != nil {
		return nil, err
	}
	return raw.(*writer), nil
}

type writer struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

func (w *writer) Write(p []byte) (int, error) {
	// 一次 Write 作为一个 SSE 事件，多行内容用多个 data: 字段承载。
	// 浏览器会将同一事件的多个 data: 字段用 \n 拼接后赋给 event.data。
	text := strings.TrimRight(strings.ReplaceAll(string(p), "\r\n", "\n"), "\n")
	if text == "" {
		return len(p), nil
	}
	for _, line := range strings.Split(text, "\n") {
		if _, err := fmt.Fprintf(w.w, "data: %s\n", line); err != nil {
			return 0, err
		}
	}
	if _, err := fmt.Fprint(w.w, "\n"); err != nil {
		return 0, err
	}
	w.flusher.Flush()
	return len(p), nil
}

func (w *writer) WriteEvent(event, data string) error {
	if _, err := fmt.Fprintf(w.w, "event: %s\ndata: %s\n\n", event, data); err != nil {
		return err
	}
	w.flusher.Flush()
	return nil
}
