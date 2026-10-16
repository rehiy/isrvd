// 启动入口：同一个二进制三种模式，由环境变量 ISRVD_MODE 选择：
//
//	server  单机，默认。与以往完全一致。
//	center  中控。isrvd 改为只监听回环端口，网关接管 listenAddr 作为对外入口，
//	        负责节点注册、按节点转发请求。
//	agent   受管机。不监听任何对外端口，主动连接中控，中控转发来的请求交给同进程的 isrvd。
package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/rehiy/libgo/httpd"
	"github.com/rehiy/libgo/logman"

	"isrvd/server/app"
	"isrvd/server/config"
	"isrvd/server/gateway"
	"isrvd/server/service/account"
	"isrvd/server/service/node"
)

// run 按环境变量指定的模式运行，阻塞到收到退出信号。
func run() error {
	switch config.Cluster().Mode {
	case app.ModeCenter:
		return runCenter()
	case app.ModeAgent:
		return runAgent()
	default:
		app.StartApp(nil, app.ModeServer)
		return nil
	}
}

// runCenter 中控：isrvd 在回环端口上服务，网关占用配置的 listenAddr 作为对外入口
func runCenter() error {
	snapshot := config.Current()

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
			AllowedOrigins:   func() []string { return config.Current().Server.AllowedOrigins },
			QueryTokenRoutes: app.QueryTokenRoutes(),
			Audit:            app.Audit(),
			Authenticate:     app.GatewayAuth,
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
func runAgent() error {
	private, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("监听回环端口失败: %w", err)
	}
	agent := node.NewAgent(node.AgentOptions{
		CenterURL: config.Cluster().CenterURL,
		NodeToken: config.Cluster().NodeToken,
		Upstream:  &url.URL{Scheme: "http", Host: private.Addr().String()},
		Token:     account.ServiceToken,
		Version:   config.Version,
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		logman.Info("agent start", "center", config.Cluster().CenterURL)
		agent.Run(ctx)
	}()

	app.StartApp(private, app.ModeAgent) // 阻塞到收到退出信号

	cancel()
	<-done
	return nil
}
