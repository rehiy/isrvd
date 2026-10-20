package node

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"runtime"
	"time"

	"github.com/rehiy/libgo/wstunnel"
)

// ProtocolVersion 中控与受管机的隧道协议版本。
const ProtocolVersion = 1

// AgentInfoPath 仅供隧道内协议协商，不经 /api/ 转发或暴露给页面。
const AgentInfoPath = "/agent/info"

// AgentInfo agent 自身的版本与平台信息，供中控做协议协商
type AgentInfo struct {
	Mode     string `json:"mode"`     // 运行模式，agent 恒为 agent
	Version  string `json:"version"`  // 程序版本
	Protocol int    `json:"protocol"` // 协议版本，须与中控一致
	Hostname string `json:"hostname"` // 主机名
	OS       string `json:"os"`       // 操作系统
	Arch     string `json:"arch"`     // CPU 架构
}

// AgentOptions 受管机隧道客户端配置
type AgentOptions struct {
	CenterURL string                 // 中控地址（http(s)://host:port，不含 /api）
	NodeToken string                 // 节点令牌，由中控生成并从环境变量传入；不另存凭据文件
	Upstream  *url.URL               // 本机 isrvd 的回环地址
	Token     func() (string, error) // 返回访问本机 isrvd 的 API 令牌；每个转发的请求都会取一次，转发的请求以令牌所属成员身份执行
	Version   string                 // 程序版本
}

// Agent 受管机侧隧道客户端：用节点令牌连接中控、保持常连，并把中控经隧道转发来的请求交给本机 isrvd。
// 自身不监听任何端口。
type Agent struct {
	opt     AgentOptions
	handler http.Handler
}

// NewAgent 创建隧道客户端
func NewAgent(opt AgentOptions) *Agent {
	a := &Agent{opt: opt}
	a.handler = a.newHandler()
	return a
}

const retryDelay = time.Minute

// Run 阻塞运行常连循环，ctx 取消后返回
func (a *Agent) Run(ctx context.Context) {
	for ctx.Err() == nil {
		logger.Info("正在连接中控", "center", a.opt.CenterURL)
		err := wstunnel.Maintain(ctx, wstunnel.MaintainOptions{
			Dial: func(ctx context.Context) (*wstunnel.Session, error) {
				return wstunnel.Dial(ctx, wstunnel.DialOptions{
					URL:    a.opt.CenterURL + "/api/node/connect",
					Header: http.Header{"Authorization": {"Bearer " + a.opt.NodeToken}},
				})
			},
			Serve:        a.serve,
			OnConnect:    func(*wstunnel.Session) { logger.Info("已连接中控", "center", a.opt.CenterURL) },
			OnDisconnect: func(*wstunnel.Session) { logger.Warn("与中控的连接已断开，将自动重连") },
			OnError:      func(err error) { logger.Warn("连接中控失败", "error", err) },
		})
		if errors.Is(err, wstunnel.ErrUnauthorized) {
			// 节点被吊销或删除，与中控数据被重置、恢复了旧备份，在这里看起来完全一样，
			// 所以只记录错误并重试：中控数据恢复后自动重连。
			logger.Error("中控入口拒绝节点接入，每分钟重试；请检查中控或反向代理认证",
				"center", a.opt.CenterURL, "error", err)
			if !wstunnel.Sleep(ctx, retryDelay) {
				return
			}
			continue
		}
		return
	}
}

// serve 在隧道上提供 HTTP 服务，直到会话关闭
func (a *Agent) serve(sess *wstunnel.Session) {
	srv := &http.Server{
		Handler:           a.handler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	startedAt := time.Now()
	if err := srv.Serve(sess.Listener()); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Warn("节点隧道服务结束", "error", err, "duration", time.Since(startedAt))
	}
}

// newHandler 只放行 /api/ 与自身信息接口；其余路径一律 404。
// 转发给本机 isrvd 的请求会换成本机签发的 API 令牌，用户凭据与来源 Origin 都不下发。
func (a *Agent) newHandler() http.Handler {
	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(a.opt.Upstream)
			h := pr.Out.Header
			token, err := a.opt.Token()
			if err != nil {
				logger.Warn("签发本机访问令牌失败", "error", err)
			}
			h.Set("Authorization", "Bearer "+token)
			h.Del("Cookie")
			h.Del("Origin") // 来源已由中控确认，本机 isrvd 若配置了 Origin 白名单不应拒绝它
			q := pr.Out.URL.Query()
			q.Del("token")
			pr.Out.URL.RawQuery = q.Encode()
		},
		FlushInterval: -1, // SSE、容器日志等流式响应需要立即刷新
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			if errors.Is(err, context.Canceled) {
				return
			}
			logger.Warn("转发到本机 isrvd 失败", "path", r.URL.Path, "error", err)
			writeError(w, r, http.StatusBadGateway, "本机 isrvd 不可用")
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET "+AgentInfoPath, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "payload": localAgentInfo(a.opt.Version)})
	})
	mux.Handle("/api/", proxy)
	return mux
}

// localAgentInfo 返回本机作为受管机对外报告的信息
func localAgentInfo(version string) *AgentInfo {
	hostname, _ := os.Hostname()
	return &AgentInfo{
		Mode:     "agent",
		Version:  version,
		Protocol: ProtocolVersion,
		Hostname: hostname,
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
	}
}
