package config

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

// ErrInvalidPortMapConfig 表示中控端口映射（端口转发）配置无效。
var ErrInvalidPortMapConfig = errors.New("端口转发配置无效")

// Enabled 判断是否配置了端口转发的监听端口范围；nil 视为未启用。
func (f *PortMapConfig) Enabled() bool {
	_, hi := f.PortRange()
	return hi > 0
}

// PortRange 返回允许分配的监听端口范围；未配置或格式无效时返回 0, 0。
func (f *PortMapConfig) PortRange() (int, int) {
	if f == nil {
		return 0, 0
	}
	lo, hi, ok := parsePortRange(f.Range)
	if !ok {
		return 0, 0
	}
	return lo, hi
}

// PortAllowed 判断端口是否在允许分配的监听范围内。
func (f *PortMapConfig) PortAllowed(port int) bool {
	lo, hi := f.PortRange()
	return hi > 0 && port >= lo && port <= hi
}

// BindAddr 返回监听地址，未配置时仅绑定回环。
func (f *PortMapConfig) BindAddr() string {
	if f == nil || f.Bind == "" {
		return "127.0.0.1"
	}
	return f.Bind
}

// validatePortMap 校验 portmap 配置；未配置（nil）合法。
func validatePortMap(f *PortMapConfig) error {
	if f == nil {
		return nil
	}
	if f.Range != "" {
		if _, _, ok := parsePortRange(f.Range); !ok {
			return fmt.Errorf("%w：portmap.range 格式应为 20000-20999 或单个端口（1–65535）", ErrInvalidPortMapConfig)
		}
	}
	if f.Bind != "" && net.ParseIP(f.Bind) == nil {
		return fmt.Errorf("%w：portmap.bind 必须是 IP 地址", ErrInvalidPortMapConfig)
	}
	return nil
}

// parsePortRange 解析 "20000-20999" 或单个端口 "20022"；空串返回 0, 0, true 表示未启用。
func parsePortRange(value string) (lo, hi int, ok bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, 0, true
	}
	loText, hiText, found := strings.Cut(value, "-")
	if !found {
		hiText = loText
	}
	lo, err1 := strconv.Atoi(strings.TrimSpace(loText))
	hi, err2 := strconv.Atoi(strings.TrimSpace(hiText))
	if err1 != nil || err2 != nil || lo < 1 || hi > 65535 || lo > hi {
		return 0, 0, false
	}
	return lo, hi, true
}
