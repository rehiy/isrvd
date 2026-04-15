#!/bin/bash
#

CLEANING=0
cleanup() {
    [ $CLEANING -eq 1 ] && return
    CLEANING=1
    echo "正在停止所有子进程..."
    # 先发 SIGTERM 给子进程
    [ -n "$GO_PID" ] && kill $GO_PID 2>/dev/null
    [ -n "$NPM_PID" ] && kill $NPM_PID 2>/dev/null
    # 尝试等待子进程优雅退出
    sleep 1
    # 强制杀掉仍存活的子进程
    [ -n "$GO_PID" ] && kill -9 $GO_PID 2>/dev/null
    [ -n "$NPM_PID" ] && kill -9 $NPM_PID 2>/dev/null
    wait 2>/dev/null
}

trap cleanup EXIT INT TERM

# 启动 Go 服务
if [ ! -f .local.yml ]; then
    cp config.yml .local.yml
fi
CONFIG_PATH=.local.yml go run cmd/server/main.go &
GO_PID=$!

# 启动 NPM 服务
cd webview
npm i
npm run dev &
NPM_PID=$!

# 等待任意一个子进程退出
wait -n $GO_PID $NPM_PID
EXIT_CODE=$?

if [ $EXIT_CODE -gt 128 ]; then
    echo "收到终止信号，正在停止所有进程..."
else
    echo "检测到子进程退出（退出码: $EXIT_CODE），正在终止其他进程..."
fi
exit $EXIT_CODE
