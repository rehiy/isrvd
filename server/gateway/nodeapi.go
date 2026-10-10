package gateway

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/libgo/httpd"
	"github.com/rehiy/libgo/websocket"

	"isrvd/server/service/node"
)

// 响应沿用 isrvd 的统一格式：{"success":bool,"message":string,"payload":any}

// ─── 节点管理（仅创始人，由 founderOnly 包装） ───

func (g *gateway) nodeList(w http.ResponseWriter, r *http.Request, _ string) {
	success(w, r, "", g.nodes.NodeList())
}

func (g *gateway) nodeUpdate(w http.ResponseWriter, r *http.Request, _ string) {
	var req node.NodeUpdateRequest
	if !decode(w, r, &req) {
		return
	}
	n, err := g.nodes.NodeUpdate(r.PathValue("id"), &req)
	if err != nil {
		nodeFail(w, r, err)
		return
	}
	success(w, r, "节点更新成功", n)
}

func (g *gateway) nodeApprove(w http.ResponseWriter, r *http.Request, _ string) {
	n, err := g.nodes.NodeApprove(r.PathValue("id"))
	if err != nil {
		nodeFail(w, r, err)
		return
	}
	success(w, r, "节点已审批", n)
}

func (g *gateway) nodeRevoke(w http.ResponseWriter, r *http.Request, _ string) {
	if err := g.nodes.NodeRevoke(r.PathValue("id")); err != nil {
		nodeFail(w, r, err)
		return
	}
	success(w, r, "节点已吊销", nil)
}

func (g *gateway) nodeDelete(w http.ResponseWriter, r *http.Request, _ string) {
	if err := g.nodes.NodeDelete(r.PathValue("id")); err != nil {
		nodeFail(w, r, err)
		return
	}
	success(w, r, "节点删除成功", nil)
}

// ─── 注册码 ───

func (g *gateway) nodeCodeList(w http.ResponseWriter, r *http.Request, _ string) {
	success(w, r, "", g.nodes.EnrollCodeList())
}

func (g *gateway) nodeCodeCreate(w http.ResponseWriter, r *http.Request, user string) {
	var req node.EnrollCodeCreateRequest
	if !decode(w, r, &req) {
		return
	}
	resp, err := g.nodes.EnrollCodeCreate(user, &req)
	if err != nil {
		nodeFail(w, r, err)
		return
	}
	success(w, r, "注册码创建成功", resp)
}

func (g *gateway) nodeCodeDelete(w http.ResponseWriter, r *http.Request, _ string) {
	if err := g.nodes.EnrollCodeDelete(r.PathValue("id")); err != nil {
		nodeFail(w, r, err)
		return
	}
	success(w, r, "注册码已撤销", nil)
}

// ─── 受管机接入（匿名，由一次性凭据、节点令牌与按 IP 限流保护） ───

func (g *gateway) nodeEnroll(w http.ResponseWriter, r *http.Request) {
	var req node.EnrollRequest
	if !decode(w, r, &req) {
		return
	}
	resp, err := g.nodes.Enroll(g.clientIP(r), &req)
	if err != nil {
		nodeFail(w, r, err)
		return
	}
	success(w, r, "注册已提交", resp)
}

func (g *gateway) nodeClaim(w http.ResponseWriter, r *http.Request) {
	var req node.ClaimRequest
	if !decode(w, r, &req) {
		return
	}
	resp, err := g.nodes.EnrollClaim(g.clientIP(r), &req)
	if err != nil {
		nodeFail(w, r, err)
		return
	}
	success(w, r, "", resp)
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
		n, err := g.nodes.Authenticate(c.Request, g.clientIP(c.Request))
		if err != nil {
			nodeFail(c.Writer, c.Request, err)
			c.Abort()
			return
		}
		ws.Handler(func(conn *websocket.ServerConn) {
			conn.SetMessageType(websocket.BinaryMessage)
			g.nodes.Serve(c.Request.Context(), conn, c.Request.RemoteAddr, n)
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

// nodeFail 把节点服务错误映射为 HTTP 状态码
func nodeFail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, node.ErrNotFound):
		fail(w, r, http.StatusNotFound, err.Error())
	case errors.Is(err, node.ErrRateLimited):
		fail(w, r, http.StatusTooManyRequests, err.Error())
	case errors.Is(err, node.ErrInvalidCode), errors.Is(err, node.ErrUnauthorized):
		fail(w, r, http.StatusForbidden, err.Error())
	default:
		fail(w, r, http.StatusBadRequest, err.Error())
	}
}
