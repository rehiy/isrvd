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

# 释放占用的端口
kill_port() {
    local port=$1
    if command -v lsof &>/dev/null; then
        local pids
        pids=$(lsof -ti tcp:"$port" 2>/dev/null)
        [ -n "$pids" ] && kill $pids 2>/dev/null
    elif command -v fuser &>/dev/null; then
        fuser -k "${port}/tcp" 2>/dev/null
    fi
}
kill_port 8080
kill_port 3000

# 启动 Go 服务
if [ ! -f .local.yml ]; then
    cp config.yml .local.yml
fi
# 默认以中控模式启动（8080 为网关入口），便于调试节点管理与节点切换；
# 需要单机模式时：ISRVD_MODE=server ./develop.sh
ISRVD_MODE=${ISRVD_MODE:-center} CONFIG_PATH=.local.yml go run ./server/cmd/server &
GO_PID=$!

# 启动 NPM 服务
cd webview
npm i
npm run dev &
NPM_PID=$!

# 等待任意一个子进程退出（兼容 macOS 默认 bash 3.2，不支持 wait -n）
EXIT_CODE=0
while kill -0 $GO_PID 2>/dev/null && kill -0 $NPM_PID 2>/dev/null; do
    sleep 1
done
if ! kill -0 $GO_PID 2>/dev/null; then
    wait $GO_PID 2>/dev/null
    EXIT_CODE=$?
else
    wait $NPM_PID 2>/dev/null
    EXIT_CODE=$?
fi

if [ $EXIT_CODE -gt 128 ]; then
    echo "收到终止信号，正在停止所有进程..."
else
    echo "检测到子进程退出（退出码: $EXIT_CODE），正在终止其他进程..."
fi
exit $EXIT_CODE
