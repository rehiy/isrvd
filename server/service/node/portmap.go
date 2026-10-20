package node

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf8"

	"isrvd/server/config"
)

// 端口转发：中控监听 Listen，经隧道在受管机上连接 Target，双向中继 TCP 字节流或 UDP 数据报。
// 规则由中控管理；转发流与 HTTP 流共用同一条隧道，以首字节区分：
//
//	HTTP 流  首字节是请求方法的 ASCII 字母
//	转发流   portmapMagic | 协议(1=tcp 2=udp) | 目标长度(1 字节) | 目标 host:port
//
// UDP 在流上以「2 字节大端长度 + 数据报」分帧。

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

// 转发协议与线路格式
const (
	ProtoTCP = "tcp"
	ProtoUDP = "udp"

	// FeaturePortMap 写入 AgentInfo.Features，表示受管机可处理转发流。
	FeaturePortMap = "portmap"

	portmapMagic byte = 0x00
	maxPortMaps       = 32
	maxTargetLen      = 200
)

// normalizePortMaps 校验并规整规则；监听端口须在中控配置 portmap.range 允许的范围内。
func normalizePortMaps(in []PortMap, allow *config.PortMapConfig) ([]PortMap, error) {
	if len(in) > maxPortMaps {
		return nil, fmt.Errorf("每个节点最多 %d 条转发规则", maxPortMaps)
	}
	seen := map[string]bool{}
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
		key := portmapKey(f.Proto, f.Listen)
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

func portmapKey(proto string, port int) string {
	return proto + "/" + strconv.Itoa(port)
}

// ─── 受管机侧：处理中控发起的转发流 ───

// servePortMap 读取转发头部、拨号目标并双向中继；r 是偷看过首字节的读取缓冲。
func servePortMap(conn net.Conn, r *bufio.Reader) {
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	proto, target, err := readPortMapHeader(r)
	_ = conn.SetReadDeadline(time.Time{})
	if err != nil {
		logger.Warn("转发头部无效", "error", err)
		return
	}
	normalized, err := normalizeTarget(target)
	if err != nil {
		logger.Warn("转发目标无效", "target", target, "error", err)
		return
	}
	target = normalized
	out, err := (&net.Dialer{Timeout: 10 * time.Second}).Dial(proto, target)
	if err != nil {
		logger.Warn("连接转发目标失败", "proto", proto, "target", target, "error", err)
		return
	}
	defer out.Close()

	stream := &bufConn{Conn: conn, r: r}
	if proto == ProtoUDP {
		relayUDP(stream, out)
		return
	}
	pipe(stream, out)
}

func writePortMapHeader(w io.Writer, proto, target string) error {
	kind := byte(1)
	if proto == ProtoUDP {
		kind = 2
	}
	_, err := w.Write(append([]byte{portmapMagic, kind, byte(len(target))}, target...))
	return err
}

func readPortMapHeader(r io.Reader) (proto, target string, err error) {
	var head [3]byte
	if _, err = io.ReadFull(r, head[:]); err != nil {
		return "", "", err
	}
	switch head[1] {
	case 1:
		proto = ProtoTCP
	case 2:
		proto = ProtoUDP
	default:
		return "", "", fmt.Errorf("未知的转发协议 %d", head[1])
	}
	buf := make([]byte, head[2])
	if _, err = io.ReadFull(r, buf); err != nil {
		return "", "", err
	}
	return proto, string(buf), nil
}

// relayUDP 在转发流（带长度分帧）与已连接的 UDP 套接字之间中继；任一侧结束即返回，由调用方关闭两端。
func relayUDP(stream, out net.Conn) {
	go func() {
		buf := make([]byte, 65535)
		for {
			n, err := out.Read(buf)
			if errors.Is(err, syscall.ECONNREFUSED) {
				continue // 目标暂不可达的 ICMP 回执，不终止会话
			}
			if err != nil || writeFrame(stream, buf[:n]) != nil {
				_ = stream.Close()
				return
			}
		}
	}()
	buf := make([]byte, 65535)
	for {
		n, err := readFrame(stream, buf)
		if err != nil {
			return
		}
		if _, err := out.Write(buf[:n]); err != nil && !errors.Is(err, syscall.ECONNREFUSED) {
			return
		}
	}
}

// ─── 辅助函数：两端共用的中继工具 ───

// bufConn 让已偷看首字节的连接继续经 bufio 读取，不丢失缓冲内的数据。
type bufConn struct {
	net.Conn
	r *bufio.Reader
}

func (c *bufConn) Read(p []byte) (int, error) { return c.r.Read(p) }

// pipe 双向拷贝：一侧正常结束时只半关闭写端，让另一方向继续排空；任一方向出错则关闭两端。
func pipe(a, b net.Conn) {
	var wg sync.WaitGroup
	cp := func(dst, src net.Conn) {
		defer wg.Done()
		if _, err := io.Copy(dst, src); err != nil {
			_ = a.Close()
			_ = b.Close()
			return
		}
		closeWrite(dst)
	}
	wg.Add(2)
	go cp(a, b)
	go cp(b, a)
	wg.Wait()
	_ = a.Close()
	_ = b.Close()
}

// closeWrite 半关闭写端；隧道流（yamux）的 Close 本身就是半关闭。
func closeWrite(c net.Conn) {
	if cw, ok := c.(interface{ CloseWrite() error }); ok {
		_ = cw.CloseWrite()
		return
	}
	_ = c.Close()
}

// writeFrame 写入一个 UDP 数据报帧：2 字节大端长度 + 数据；超长的数据报直接丢弃。
func writeFrame(w io.Writer, p []byte) error {
	if len(p) > 0xffff {
		return nil
	}
	buf := make([]byte, 2+len(p))
	binary.BigEndian.PutUint16(buf, uint16(len(p)))
	copy(buf[2:], p)
	_, err := w.Write(buf)
	return err
}

// readFrame 读取一个 UDP 数据报帧到 buf（至少 65535 字节），返回数据长度。
func readFrame(r io.Reader, buf []byte) (int, error) {
	var head [2]byte
	if _, err := io.ReadFull(r, head[:]); err != nil {
		return 0, err
	}
	n := int(binary.BigEndian.Uint16(head[:]))
	_, err := io.ReadFull(r, buf[:n])
	return n, err
}
