package model

// shell消息结构
type ShellMessage struct {
	Type string `json:"type"`           // "input", "output", "error", "resize"
	Data string `json:"data"`           // 消息内容
	Cols int    `json:"cols,omitempty"` // 终端列数 (resize时使用)
	Rows int    `json:"rows,omitempty"` // 终端行数 (resize时使用)
}

// shell会话信息
type ShellSession struct {
	ID        string `json:"id"`
	StartTime string `json:"startTime"`
	IsActive  bool   `json:"isActive"`
	Command   string `json:"command"`
}
