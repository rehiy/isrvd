package models

import (
	"sync"
	"time"
)

// Session 会话管理器
type Session struct {
	Sessions map[string]time.Time
	Mutex    sync.RWMutex
}
