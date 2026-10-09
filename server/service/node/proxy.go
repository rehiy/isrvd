package node

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
)

var errOffline = errors.New("节点离线")

// ProxyServe 把请求原样转发到指定节点（含流式响应与 WebSocket 升级）。
// 调用方须已完成认证、权限与审计。
// 失败时直接写出统一格式的 JSON 错误：节点不存在 404，离线 503，隧道错误 502，超时 504。
func (s *Service) ProxyServe(w http.ResponseWriter, r *http.Request, id string) {
	c, err := s.connFor(id)
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "节点不存在")
		return
	case errors.Is(err, errOffline):
		writeError(w, http.StatusServiceUnavailable, "节点离线")
		return
	case err != nil:
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	c.proxy.ServeHTTP(w, r)
}

// Fetch 经隧道对节点发起一次 GET 请求并返回响应体（最多 1 MiB）。
func (s *Service) Fetch(ctx context.Context, id, path string) ([]byte, error) {
	c, err := s.connFor(id)
	if err != nil {
		return nil, err
	}
	resp, err := c.get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("节点返回 HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 1<<20))
}

// newProxy 创建指向单个节点的反向代理。
func newProxy(transport *http.Transport) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			out := pr.Out
			out.URL.Scheme = "http"
			out.URL.Host = "agent" // 占位，Transport 的 DialContext 会忽略地址
			// 用户凭据只属于中控，不下发给节点
			out.Header.Del("Authorization")
			out.Header.Del("Cookie")
			q := out.URL.Query()
			if q.Has("token") {
				q.Del("token")
				out.URL.RawQuery = q.Encode()
			}
			pr.SetXForwarded()
		},
		Transport:     transport,
		FlushInterval: -1, // SSE、容器日志等流式响应需要立即刷新
		ErrorHandler:  proxyError,
	}
}

// ─── 辅助函数 ───

func proxyError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, context.Canceled) {
		return // 浏览器已断开，无需响应
	}
	status, message := http.StatusBadGateway, "节点请求失败"
	var netErr net.Error
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout()) {
		status, message = http.StatusGatewayTimeout, "节点响应超时"
	}
	logger.Warn("节点代理失败", "method", r.Method, "path", r.URL.Path, "error", err)
	writeError(w, status, message)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"success": false, "message": message})
}
