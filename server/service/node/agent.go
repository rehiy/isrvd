package node

import (
	"bytes"
	"context"
	"crypto/cipher"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"runtime"
	"time"

	"github.com/goccy/go-yaml"

	"isrvd/pkgs/secretbox"
	"isrvd/pkgs/tunnel"
)

const (
	// AgentInfoPath 受管机隧道客户端自身的信息接口，不会转发到本机 isrvd。
	// 该路径不在 /api/ 下，中控只会把 /api/ 下的请求经隧道转发，用户无法访问到它。
	AgentInfoPath = "/__agent/info"

	enrollPollPeriod = 10 * time.Second
	enrollRetryDelay = time.Minute
)

// AgentOptions 受管机隧道客户端配置
type AgentOptions struct {
	CenterURL  string                 // 中控地址（http(s)://host:port，不含 /api）
	EnrollCode string                 // 首次注册使用的注册码；本地已有节点令牌时可为空
	Name       string                 // 节点显示名称，默认取主机名
	Upstream   *url.URL               // 本机 isrvd 的回环地址
	Token      func() (string, error) // 返回访问本机 isrvd 的 API 令牌；每个转发的请求都会取一次，转发的请求以令牌所属成员身份执行
	StateKey   string                 // 加密本地凭据的密钥材料，需保持稳定（如 JWT 密钥）；变更后旧凭据无法解密，会重新注册
	StatePath  string                 // 本地凭据文件路径
	Version    string                 // 程序版本
}

// Agent 受管机侧隧道客户端：向中控注册、保持常连，并把中控经隧道转发来的请求交给本机 isrvd。
// 自身不监听任何端口。
type Agent struct {
	opt       AgentOptions
	handler   http.Handler
	codeSpent bool // 启动时给的注册码已经用过（成功或被中控明确拒绝），不再重试
}

// NewAgent 创建隧道客户端
func NewAgent(opt AgentOptions) *Agent {
	a := &Agent{opt: opt}
	a.handler = a.newHandler()
	return a
}

// Run 阻塞运行注册与常连循环，ctx 取消后返回
func (a *Agent) Run(ctx context.Context) {
	reenroll := false // 中控拒绝了本地令牌，且还有可用的注册码时置位
	for ctx.Err() == nil {
		state, err := a.credential(ctx, reenroll)
		reenroll = false
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			logger.Error("节点注册失败", "error", err)
			if !tunnel.Sleep(ctx, enrollRetryDelay) {
				return
			}
			continue
		}

		logger.Info("正在连接中控", "center", a.opt.CenterURL, "id", state.NodeID)
		err = tunnel.Maintain(ctx, tunnel.MaintainOptions{
			Dial: func(ctx context.Context) (*tunnel.Session, error) {
				return tunnel.Dial(ctx, tunnel.DialOptions{
					URL:    a.opt.CenterURL + "/api/node/connect",
					Header: http.Header{"Authorization": {"Bearer " + state.Token}},
				})
			},
			Serve:        a.serve,
			OnConnect:    func(*tunnel.Session) { logger.Info("已连接中控", "center", a.opt.CenterURL) },
			OnDisconnect: func(*tunnel.Session) { logger.Warn("与中控的连接已断开，将自动重连") },
			OnError:      func(err error) { logger.Warn("连接中控失败", "error", err) },
		})
		if errors.Is(err, tunnel.ErrUnauthorized) {
			// 不能据此删除本地凭据：节点被吊销或删除，与中控的数据被重置、恢复了旧备份，
			// 在这里看起来完全一样。删掉后后者将永久失联，保留则在中控数据恢复后自动重连。
			if a.opt.EnrollCode != "" && !a.codeSpent {
				logger.Warn("中控拒绝了节点令牌，改用注册码重新注册（注册成功后才会替换本地凭据）")
				reenroll = true
				continue
			}
			logger.Error("中控拒绝了节点令牌：节点可能已被吊销或删除，或中控数据被重置。本地凭据已保留并每分钟重试；"+
				"如需重新接入，请使用新的注册码重启", "center", a.opt.CenterURL, "id", state.NodeID)
			if !tunnel.Sleep(ctx, enrollRetryDelay) {
				return
			}
			continue
		}
		return
	}
}

// serve 在隧道上提供 HTTP 服务，直到会话关闭
func (a *Agent) serve(sess *tunnel.Session) {
	srv := &http.Server{
		Handler:           a.handler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	_ = srv.Serve(sess.Listener())
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
			writeError(w, http.StatusBadGateway, "本机 isrvd 不可用")
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET "+AgentInfoPath, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "payload": LocalAgentInfo(a.opt.Version)})
	})
	mux.Handle("/api/", proxy)
	return mux
}

// LocalAgentInfo 返回本机作为受管机对外报告的信息
func LocalAgentInfo(version string) *AgentInfo {
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

// ─── 注册与凭据 ───

// agentState 本节点私有的凭据，保存在本地文件
type agentState struct {
	NodeID    string `yaml:"nodeId"`
	Token     string `yaml:"token"`     // 加密落盘
	CenterURL string `yaml:"centerUrl"` // 令牌所属的中控
}

// credential 读取本地凭据；没有（或要求重新注册）时使用注册码向中控注册并等待审批。
// 注册码只能用一次：成功或被中控明确拒绝后不再重试，之后沿用本地凭据。
func (a *Agent) credential(ctx context.Context, reenroll bool) (*agentState, error) {
	state := a.stateLoad()
	usable := state != nil && state.CenterURL == a.opt.CenterURL
	if usable && !reenroll {
		return state, nil
	}
	if a.opt.EnrollCode == "" || a.codeSpent {
		if usable {
			return state, nil
		}
		return nil, errors.New("本地没有可用的节点令牌且没有可用的注册码（--enroll-code 或 ISRVD_ENROLL_CODE），无法向中控注册")
	}

	enrolled, err := a.enroll(ctx)
	if err == nil || rejected(err) {
		a.codeSpent = true
	}
	if err != nil && usable {
		logger.Warn("重新注册失败，继续使用本地凭据", "error", err)
		return state, nil
	}
	return enrolled, err
}

func (a *Agent) enroll(ctx context.Context) (*agentState, error) {
	client := &http.Client{
		Timeout: 20 * time.Second,
		// 不跟随重定向，避免注册码被转发到非预期的主机
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	info := LocalAgentInfo(a.opt.Version)
	name := a.opt.Name
	if name == "" {
		name = info.Hostname
	}

	var enrolled EnrollResponse
	if err := postJSON(ctx, client, a.opt.CenterURL+"/api/node/enroll", &EnrollRequest{
		Code: a.opt.EnrollCode, Name: name, Hostname: info.Hostname, OS: info.OS, Arch: info.Arch,
	}, &enrolled); err != nil {
		return nil, fmt.Errorf("提交注册失败: %w", err)
	}
	logger.Info("已向中控提交注册", "id", enrolled.NodeID, "status", enrolled.Status)

	for {
		var claimed ClaimResponse
		err := postJSON(ctx, client, a.opt.CenterURL+"/api/node/enroll/claim", &ClaimRequest{
			NodeID: enrolled.NodeID, ClaimSecret: enrolled.ClaimSecret,
		}, &claimed)
		switch {
		case ctx.Err() != nil:
			return nil, ctx.Err()
		case rejected(err):
			return nil, fmt.Errorf("领取令牌被拒绝: %w", err) // 节点被删除或吊销，重试没有意义
		case err != nil:
			logger.Warn("领取令牌失败，稍后重试", "error", err)
		case claimed.Status == StatusApproved && claimed.Token != "":
			state := &agentState{NodeID: enrolled.NodeID, Token: claimed.Token, CenterURL: a.opt.CenterURL}
			if err := a.stateSave(state); err != nil {
				return nil, fmt.Errorf("保存节点令牌失败: %w", err)
			}
			logger.Info("节点已通过审批", "id", enrolled.NodeID)
			return state, nil
		default:
			logger.Info("等待管理员审批", "id", enrolled.NodeID)
		}
		if !tunnel.Sleep(ctx, enrollPollPeriod) {
			return nil, ctx.Err()
		}
	}
}

// httpError 中控返回的非成功响应
type httpError struct {
	Status  int
	Message string
}

// rejected 判断中控是否明确拒绝了请求（4xx，限流除外）；5xx 与网络错误值得重试，不算拒绝
func rejected(err error) bool {
	var e *httpError
	return errors.As(err, &e) && e.Status >= 400 && e.Status < 500 && e.Status != http.StatusTooManyRequests
}

func (e *httpError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("HTTP %d", e.Status)
	}
	return fmt.Sprintf("HTTP %d: %s", e.Status, e.Message)
}

// postJSON 向中控提交 JSON，并把统一响应中的 payload 解码到 out
func postJSON(ctx context.Context, client *http.Client, url string, in, out any) error {
	body, err := json.Marshal(in)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	var envelope struct {
		Success bool            `json:"success"`
		Message string          `json:"message"`
		Payload json.RawMessage `json:"payload"`
	}
	_ = json.Unmarshal(raw, &envelope)
	if resp.StatusCode != http.StatusOK || !envelope.Success {
		return &httpError{Status: resp.StatusCode, Message: envelope.Message}
	}
	return json.Unmarshal(envelope.Payload, out)
}

// ─── 本地凭据存储 ───

// aead 令牌落盘加密：密钥由 StateKey 派生（同进程运行时用 JWT 密钥），无需另配密钥；
// 密钥变更后旧凭据无法解密，会重新注册。
func (a *Agent) aead() (cipher.AEAD, error) {
	return secretbox.NewAEAD("isrvd-node-agent", a.opt.StateKey)
}

// stateLoad 读取并解密本地凭据；缺失或无法解密返回 nil
func (a *Agent) stateLoad() *agentState {
	data, err := os.ReadFile(a.opt.StatePath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			logger.Warn("读取节点凭据失败", "error", err)
		}
		return nil
	}
	var state agentState
	if err := yaml.Unmarshal(data, &state); err != nil {
		logger.Warn("节点凭据文件格式无效", "error", err)
		return nil
	}
	if state.NodeID == "" {
		return nil
	}
	aead, err := a.aead()
	if err != nil {
		return nil
	}
	token, err := secretbox.Open(aead, state.Token)
	if err != nil {
		logger.Warn("节点令牌解密失败，JWT 密钥可能已变更，将重新注册")
		return nil
	}
	state.Token = token
	return &state
}

// stateSave 加密令牌后原子写入，文件权限 0600
func (a *Agent) stateSave(state *agentState) error {
	aead, err := a.aead()
	if err != nil {
		return err
	}
	sealed, err := secretbox.Seal(aead, state.Token)
	if err != nil {
		return err
	}
	data, err := yaml.Marshal(&agentState{NodeID: state.NodeID, Token: sealed, CenterURL: state.CenterURL})
	if err != nil {
		return err
	}
	tmp := a.opt.StatePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, a.opt.StatePath)
}
