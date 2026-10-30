package helper

import (
	"bytes"
	"errors"
)

// MaxEditableFileBytes 在线编辑文件的内容上限，所有在线编辑入口共用。
const MaxEditableFileBytes int64 = 8 << 20

// ErrTooLarge 写入内容超过 MaxEditableFileBytes。
var ErrTooLarge = errors.New("文件超过在线编辑上限")

// ErrBinary 内容为二进制数据，不支持在线编辑。
var ErrBinary = errors.New("文件不是文本文件，无法在线编辑")

// EditBuffer 最多接收 MaxEditableFileBytes+1 字节的缓冲：超出后 Write 返回 ErrTooLarge
// 以中断上游复制；多出的 1 字节用于通过 Exceeded 区分「恰好等于上限」与「超过上限」。
type EditBuffer struct {
	bytes.Buffer
}

// Write 实现 io.Writer。
func (b *EditBuffer) Write(p []byte) (int, error) {
	remaining := MaxEditableFileBytes + 1 - int64(b.Len())
	if remaining <= 0 {
		return 0, ErrTooLarge
	}
	if int64(len(p)) > remaining {
		n, _ := b.Buffer.Write(p[:int(remaining)])
		return n, ErrTooLarge
	}
	return b.Buffer.Write(p)
}

// Exceeded 已写入内容是否超过 MaxEditableFileBytes。
func (b *EditBuffer) Exceeded() bool {
	return int64(b.Len()) > MaxEditableFileBytes
}
