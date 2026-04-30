@echo off
chcp 65001 > nul
echo Starting isrvd dev environment...

set GOAMD64=v1

:: 启动后端
start "isrvd backend" cmd /k "cd /d %~dp0 && set GOAMD64=v1 && go run cmd/server/main.go"

:: 稍等后端初始化
timeout /t 2 /nobreak > nul

:: 启动前端
start "isrvd frontend" cmd /k "cd /d %~dp0webview && npm run dev"

echo.
echo Backend  -^> http://localhost:8080
echo Frontend -^> http://localhost:3000
echo.
echo Close the terminal windows to stop.
