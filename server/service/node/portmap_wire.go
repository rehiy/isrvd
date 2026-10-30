package node

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"syscall"
	"time"
)

// 端口转发的线路格式与双向中继工具：头部、分帧、pipe 为两端共用；servePortMap 是受管机侧入口。
// 转发流与 HTTP 流共用同一条隧道，以首字节区分：
//
//	HTTP 流  首字节是请求方法的 ASCII 字母
//	转发流   portmapMagic | 协议(1=tcp 2=udp) | 目标长度(1 字节) | 目标 host:port
//
// UDP 在流上以「2 字节大端长度 + 数据报」分帧。

const portmapMagic byte = 0x00

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

// bufConn 让已偷看首字节的连接继续经 bufio 读取，不丢失缓冲内的数据。
type bufConn struct {
	net.Conn
	r *bufio.Reader
}

func (c *bufConn) Read(p []byte) (int, error) {
	return c.r.Read(p)
}

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

// servePortMap 受管机侧处理中控发起的转发流：读取转发头部、拨号目标并双向中继；
// r 是偷看过首字节的读取缓冲。
func servePortMap(conn net.Conn, r *bufio.Reader) {
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	proto, raw, err := readPortMapHeader(r)
	_ = conn.SetReadDeadline(time.Time{})
	if err != nil {
		logger.Warn("转发头部无效", "error", err)
		return
	}
	target, err := normalizeTarget(raw)
	if err != nil {
		logger.Warn("转发目标无效", "target", raw, "error", err)
		return
	}
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
