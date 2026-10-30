package gateway

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/libgo/httpd"
	"github.com/rehiy/libgo/websocket"

	"isrvd/pkgs/helper"
	"isrvd/server/service/node"
)

func (g *gateway) nodeList(w http.ResponseWriter, r *http.Request) {
	success(w, r, "", g.Nodes.NodeList())
}

func (g *gateway) nodeCreate(w http.ResponseWriter, r *http.Request) {
	var req node.NodeRequest
	if decode(w, r, &req) {
		resp, err := g.Nodes.NodeCreate(&req)
		reply(w, r, "创建成功", resp, err)
	}
}

func (g *gateway) nodeUpdate(w http.ResponseWriter, r *http.Request) {
	var req node.NodeRequest
	if decode(w, r, &req) {
		n, err := g.Nodes.NodeUpdate(r.PathValue("id"), &req)
		reply(w, r, "节点更新成功", n, err)
	}
}

func (g *gateway) nodeRevoke(w http.ResponseWriter, r *http.Request) {
	reply(w, r, "节点已吊销", nil, g.Nodes.NodeRevoke(r.PathValue("id")))
}

func (g *gateway) nodeDelete(w http.ResponseWriter, r *http.Request) {
	reply(w, r, "节点删除成功", nil, g.Nodes.NodeDelete(r.PathValue("id")))
}

func (g *gateway) nodePortMapUpdate(w http.ResponseWriter, r *http.Request) {
	var req node.PortMapRequest
	if decode(w, r, &req) {
		n, err := g.Nodes.NodePortMapUpdate(r.PathValue("id"), &req)
		reply(w, r, "端口转发已更新", n, err)
	}
}

// nodeConnect 受管机建立隧道的入口。WebSocket 升级与二进制流适配交给 libgo 的 websocket.ServerConfig，
// 这里只做它不负责的两件事：拒绝浏览器发起的连接、校验节点令牌（令牌只经 Authorization 头传递，不进 URL）。
func (g *gateway) nodeConnect() http.Handler {
	ws := &websocket.ServerConfig{WriteTimeout: 15 * time.Second}
	r := gin.New()
	r.Use(httpd.Recovery())
	r.GET("/api/node/connect", func(c *gin.Context) {
		// 浏览器发起的 WebSocket 必带 Origin；受管机不是浏览器，带 Origin 的一律拒绝，
		// 这样即使令牌泄漏，也无法被跨站页面利用
		if c.GetHeader("Origin") != "" {
			fail(c.Writer, c.Request, http.StatusForbidden, "不接受浏览器发起的隧道连接")
			c.Abort()
			return
		}
		n, err := g.Nodes.Authenticate(c.Request, helper.RemoteHost(c.Request))
		if err != nil {
			nodeFail(c.Writer, c.Request, err)
			c.Abort()
			return
		}
		defer g.Nodes.ReleaseConnection(n)
		ws.Handler(func(conn *websocket.ServerConn) {
			conn.SetMessageType(websocket.BinaryMessage)
			g.Nodes.Serve(c.Request.Context(), conn, c.Request.RemoteAddr, n)
		})(c)
	})
	return r
}

// ─── 辅助函数 ───

// decode 解析 JSON 请求体（上限 1 MiB）；失败时已写出 400
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			fail(w, r, http.StatusRequestEntityTooLarge, "请求体过大")
			return false
		}
		fail(w, r, http.StatusBadRequest, "请求体无效: "+err.Error())
		return false
	}
	return true
}

// reply 把节点服务的结果写成统一响应：出错时映射状态码，成功时返回提示与数据
func reply(w http.ResponseWriter, r *http.Request, message string, payload any, err error) {
	if err != nil {
		nodeFail(w, r, err)
		return
	}
	success(w, r, message, payload)
}

// nodeFail 把节点服务错误映射为 HTTP 状态码
func nodeFail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, node.ErrNotFound):
		fail(w, r, http.StatusNotFound, err.Error())
	case errors.Is(err, node.ErrAlreadyOnline):
		fail(w, r, http.StatusConflict, err.Error())
	case errors.Is(err, node.ErrRateLimited):
		fail(w, r, http.StatusTooManyRequests, err.Error())
	case errors.Is(err, node.ErrUnauthorized):
		fail(w, r, http.StatusForbidden, err.Error())
	default:
		fail(w, r, http.StatusBadRequest, err.Error())
	}
}
