# Shell 终端 API

## 概述

Shell 模块提供 Web 终端功能，允许通过浏览器直接访问 isrvd 服务器本地的 Shell。与 WebSSH 不同，Shell 终端直接运行在 isrvd 服务器上，无需配置远程主机。

---

## 打开 Shell 终端

通过 WebSocket 连接到服务器本地 Shell：

```bash
# 使用 wscat 连接（需先获取 token）
TOKEN=$(isrvd_login "$ISRVD_APIURL" "$ISRVD_USERNAME" "$ISRVD_PASSWORD" | jq -r '.payload.token')
wscat -c "ws://<HOST>/api/shell?token=$TOKEN"

# 指定 shell（可选，仅支持白名单短名）
wscat -c "ws://<HOST>/api/shell?token=$TOKEN&shell=bash"
```

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| `token` | string | JWT 认证令牌；JWT 模式必需，代理 Header（THA）模式不需要 |
| `shell` | string | 可选，仅接受白名单短名：`bash`、`sh`、`zsh`、`powershell`、`pwsh`、`cmd`；传入路径（如 `/bin/bash`）、非白名单或不可用时回退为系统默认 shell（Linux 依次尝试 bash/sh/zsh，macOS 依次尝试 zsh/bash/sh，Windows 依次尝试 powershell/pwsh/cmd） |

**说明：**
- 终端工作目录为用户主目录（`member.homeDirectory`）
- 支持 ANSI 转义序列、颜色、光标定位等
- 连接关闭后终端会话自动结束
- 终端 WSS 连接内置约 25s 保活心跳，空闲时不会被中间层（nginx/Caddy、NAT）断开
- 前端窗口尺寸变化时发送 resize 控制帧 `\u0000isrvd:resize:<cols>:<rows>`（cols 10–1000，rows 3–1000，格式错误或越界被静默丢弃）
- PTY 启动失败时降级为 Pipe 模式并输出提示，Pipe 模式不支持 resize

---

## 浏览器复制与粘贴

本机 Shell、WebSSH 和 Docker 容器终端共用以下快捷键规则（需先点击终端获取焦点）：

| 快捷键 | 行为 |
|--------|------|
| Windows / Linux `Ctrl+C`，已选中文字 | 复制选中内容，不向终端发送中断信号；选区保持不变 |
| Windows / Linux `Ctrl+C`，未选中文字 | 向终端发送原有中断信号，通常用于停止前台命令 |
| Windows / Linux `Ctrl+V`、`Ctrl+Shift+V` | 粘贴剪贴板文字，优先于终端原有的 `^V` 功能（因此 vim 的 `Ctrl+V` 可视块模式不可用；可先在终端执行 `stty -ixon` 关闭流控，再用 `Ctrl+Q` 代替） |
| macOS `⌘C` / `⌘V` | 使用浏览器原生复制 / 粘贴，`Ctrl+C` 等 Control 组合键继续交给终端 |

复制后若需要中断命令，先点击终端取消选区，再按 `Ctrl+C`。复制和粘贴只支持文字，使用浏览器原生剪贴板事件，不会上传文件。

---

## 权限要求

- 需要登录（任意认证方式）
- 需要授予 `GET /api/shell` 路由权限（创始人除外）；当前用户在成员列表中不存在时返回 403
- 实际 shell 权限取决于运行 isrvd 的系统用户权限

---

## 与 WebSSH 的区别

| 特性 | Shell | WebSSH |
|------|-------|--------|
| 连接目标 | isrvd 服务器本地 | 远程 SSH 主机 |
| 配置要求 | 无（开箱即用） | 需配置 SSH 主机 |
| 认证方式 | isrvd JWT | SSH 密码/私钥 |
| 适用场景 | 服务器本地管理 | 远程服务器管理 |
| WebSocket 路径 | `/api/shell` | `/api/ssh/to/<ID>` |
| 文件管理 | `/api/filer/*`（本地） | `/api/sftp/:id/*`（远程） |
