// 启动入口：同一个二进制三种模式，由 --mode（或环境变量 ISRVD_MODE）选择：
//
//	server  单机，默认。与以往完全一致。
//	center  中控。isrvd 改为只监听回环端口，网关接管 listenAddr 作为对外入口，
//	        负责节点注册与审批、按节点转发请求。
//	agent   受管机。不监听任何对外端口，主动连接中控，中控转发来的请求交给同进程的 isrvd。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rehiy/libgo/httpd"
	"github.com/rehiy/libgo/logman"

	"isrvd/server/app"
	"isrvd/server/config"
	"isrvd/server/gateway"
	"isrvd/server/service/account"
	"isrvd/server/service/node"
)

type options struct {
	mode       string
	centerURL  string // agent：中控地址
	enrollCode string // agent：首次注册的注册码
	nodeName   string // agent：节点名称
}

// run 按模式运行，阻塞到收到退出信号。调用前须已完成 parse 与 config.Init。
func run(opt options) error {
	switch opt.mode {
	case app.ModeCenter:
		return runCenter()
	case app.ModeAgent:
		return runAgent(opt)
	default:
		app.StartApp(nil, app.ModeServer)
		return nil
	}
}

// parse 解析并校验命令行与环境变量，不读取配置，也不产生任何副作用。
func parse(args []string) (options, error) {
	var opt options
	fs := flag.NewFlagSet("isrvd", flag.ContinueOnError)
	fs.StringVar(&opt.mode, "mode", config.EnvOrDefault("ISRVD_MODE", app.ModeServer), "运行模式：server（默认）、center（中控）、agent（受管机）")
	fs.StringVar(&opt.centerURL, "center-url", config.EnvOrDefault("ISRVD_CENTER_URL", ""), "agent 模式：中控地址，如 https://center.example.com")
	fs.StringVar(&opt.enrollCode, "enroll-code", config.EnvOrDefault("ISRVD_ENROLL_CODE", ""), "agent 模式：首次注册的注册码，已有本地凭据后可省略")
	fs.StringVar(&opt.nodeName, "node-name", config.EnvOrDefault("ISRVD_NODE_NAME", ""), "agent 模式：节点名称，默认取主机名")
	if err := fs.Parse(args); err != nil {
		return opt, err
	}
	if fs.NArg() > 0 {
		return opt, fmt.Errorf("未知参数 %q", fs.Arg(0))
	}

	opt.mode = strings.ToLower(strings.TrimSpace(opt.mode))
	opt.centerURL = strings.TrimRight(strings.TrimSpace(opt.centerURL), "/")
	opt.enrollCode = strings.TrimSpace(opt.enrollCode)
	opt.nodeName = strings.TrimSpace(opt.nodeName)

	switch opt.mode {
	case app.ModeServer, app.ModeCenter:
		return opt, nil
	case app.ModeAgent:
		if err := config.ValidateHTTPURL("--center-url", opt.centerURL); err != nil {
			return opt, err
		}
		return opt, nil
	default:
		return opt, fmt.Errorf("--mode 无效: %q（可选 server / center / agent）", opt.mode)
	}
}

// runCenter 中控：isrvd 在回环端口上服务，网关占用配置的 listenAddr 作为对外入口
func runCenter() error {
	snapshot := config.Current()
	if snapshot.THA != nil && snapshot.THA.Enabled {
		logman.Warn("center 模式下不应启用代理 Header 登录：网关转发的请求在 isrvd 看来都来自本机，客户端可伪造登录头")
	}

	// 先初始化 gin（模式与日志输出），网关内的 gin 路由随之生效；app 再次调用时复用同一实例
	httpd.Engine(snapshot.Server.Debug)

	nodes, err := node.NewService()
	if err != nil {
		return err
	}
	public, err := net.Listen("tcp", snapshot.Server.ListenAddr)
	if err != nil {
		return fmt.Errorf("监听 %s 失败: %w", snapshot.Server.ListenAddr, err)
	}
	private, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("监听回环端口失败: %w", err)
	}

	srv := &http.Server{
		Handler: gateway.Handler(gateway.Options{
			Upstream:         &url.URL{Scheme: "http", Host: private.Addr().String()},
			Nodes:            nodes,
			TrustProxy:       os.Getenv("ISRVD_CENTER_TRUST_PROXY") == "1",
			AllowedOrigins:   func() []string { return config.Current().Server.AllowedOrigins },
			QueryTokenRoutes: app.QueryTokenRoutes(),
			Audit:            app.Audit(),
		}),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	go func() {
		logman.Info("center gateway start", "listen", public.Addr().String(), "upstream", private.Addr().String())
		if err := srv.Serve(public); !errors.Is(err, http.ErrServerClosed) {
			logman.Error("center gateway stopped", "error", err)
		}
	}()

	app.StartApp(private, app.ModeCenter) // 阻塞到收到退出信号

	// 隧道是被劫持的长连接，Shutdown 不会等待它们，需先主动断开
	nodes.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	return nil
}

// runAgent 受管机：isrvd 只在回环端口上服务，隧道客户端主动连接中控
func runAgent(opt options) error {
	snapshot := config.Current()
	if snapshot.THA != nil && snapshot.THA.Enabled {
		return errors.New("agent 模式需要 JWT 登录，请关闭代理 Header 登录（tha.enabled）")
	}

	private, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("监听回环端口失败: %w", err)
	}
	agent := node.NewAgent(node.AgentOptions{
		CenterURL:  opt.centerURL,
		EnrollCode: opt.enrollCode,
		Name:       opt.nodeName,
		Upstream:   &url.URL{Scheme: "http", Host: private.Addr().String()},
		Token:      account.ServiceToken,
		StateKey:   snapshot.Server.JWTSecret,
		StatePath:  filepath.Join(snapshot.Server.RootDirectory, "node-agent.yml"),
		Version:    config.Version,
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		logman.Info("agent start", "center", opt.centerURL)
		agent.Run(ctx)
	}()

	app.StartApp(private, app.ModeAgent) // 阻塞到收到退出信号

	cancel()
	<-done
	return nil
}
