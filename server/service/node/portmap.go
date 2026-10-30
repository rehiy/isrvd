package node

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"isrvd/server/config"
)

// 端口转发：中控监听 Listen，经隧道在受管机上连接 Target，双向中继 TCP 字节流或 UDP 数据报。
// 本文件只含规则模型与校验；线路格式与中继工具见 portmap_wire.go，中控监听见 portmap_listener.go，
// 受管机侧处理同在 portmap_wire.go。

// PortMap 端口转发规则
type PortMap struct {
	Name   string `yaml:"name,omitempty" json:"name,omitempty"` // 备注
	Proto  string `yaml:"proto" json:"proto"`                   // tcp / udp
	Listen int    `yaml:"listen" json:"listen"`                 // 中控监听端口
	Target string `yaml:"target" json:"target"`                 // 节点侧目标 host:port，host 缺省为 127.0.0.1
}

// PortMapRequest 整体替换某节点的端口转发规则
type PortMapRequest struct {
	PortMaps []PortMap `json:"portmaps"`
}

// 转发协议与规则上限
const (
	ProtoTCP = "tcp"
	ProtoUDP = "udp"

	maxPortMaps  = 32
	maxTargetLen = 200
)

// normalizePortMaps 校验并规整规则；监听端口须在中控配置 portmap.range 允许的范围内。
func normalizePortMaps(in []PortMap, allow *config.PortMapConfig) ([]PortMap, error) {
	if len(in) > maxPortMaps {
		return nil, fmt.Errorf("每个节点最多 %d 条转发规则", maxPortMaps)
	}
	seen := map[portKey]bool{}
	out := make([]PortMap, 0, len(in))
	for _, f := range in {
		f.Name = strings.TrimSpace(f.Name)
		f.Proto = strings.ToLower(strings.TrimSpace(f.Proto))
		if utf8.RuneCountInString(f.Name) > 64 {
			return nil, errors.New("转发备注不能超过 64 个字符")
		}
		if f.Proto != ProtoTCP && f.Proto != ProtoUDP {
			return nil, errors.New("转发协议只能是 tcp 或 udp")
		}
		if !allow.Enabled() {
			return nil, errors.New("中控未启用端口转发，请在配置文件 portmap.range 中设置后重启")
		}
		if !allow.PortAllowed(f.Listen) {
			lo, hi := allow.PortRange()
			return nil, fmt.Errorf("监听端口 %d 不在允许范围 %d-%d 内", f.Listen, lo, hi)
		}
		target, err := normalizeTarget(f.Target)
		if err != nil {
			return nil, err
		}
		f.Target = target
		key := portKey{f.Proto, f.Listen}
		if seen[key] {
			return nil, fmt.Errorf("监听端口 %d/%s 重复", f.Listen, f.Proto)
		}
		seen[key] = true
		out = append(out, f)
	}
	return out, nil
}

// hostPattern 目标主机名（IP 另行用 net.ParseIP 校验）
var hostPattern = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9._-]{0,251}[A-Za-z0-9])?$`)

// normalizeTarget 校验 host:port，host 缺省为 127.0.0.1。
func normalizeTarget(target string) (string, error) {
	host, portText, err := net.SplitHostPort(strings.TrimSpace(target))
	if err != nil {
		return "", errors.New("目标地址格式应为 host:port")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return "", errors.New("目标端口须为 1–65535")
	}
	if host == "" {
		host = "127.0.0.1"
	}
	if net.ParseIP(host) == nil && !hostPattern.MatchString(host) {
		return "", errors.New("目标主机无效")
	}
	out := net.JoinHostPort(host, portText)
	if len(out) > maxTargetLen {
		return "", errors.New("目标地址过长")
	}
	return out, nil
}

// portKey 监听端口的唯一标识：同一协议下的端口
type portKey struct {
	proto string
	port  int
}
