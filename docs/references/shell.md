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

# 指定 shell（可选，默认为用户默认 shell）
wscat -c "ws://<HOST>/api/shell?token=$TOKEN&shell=/bin/bash"
```

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| `token` | string | JWT 认证令牌（必需） |
| `shell` | string | 指定 shell 路径（可选，如 `/bin/bash`、`/bin/zsh`） |

**说明：**
- 终端工作目录为用户主目录（`member.homeDirectory`）
- 支持 ANSI 转义序列、颜色、光标定位等
- 连接关闭后终端会话自动结束
- 终端 WSS 连接内置约 25s 保活心跳，空闲时不会被中间层（nginx/Caddy、NAT）断开

---

## 前端集成

前端通过 WebSocket 连接，推荐使用 `xterm.js` 渲染终端：

```javascript
const ws = new WebSocket(`ws://${host}/api/shell?token=${token}`);
// 或使用指定的 shell
// const ws = new WebSocket(`ws://${host}/api/shell?token=${token}&shell=/bin/bash`);

ws.onmessage = (event) => {
  // event.data 为终端输出（可能是字符串或 Blob）
  terminal.write(event.data);
};

terminal.onData((data) => {
  ws.send(data);
});
```

---

## 权限要求

- 需要登录（任意认证方式）
- 需要 `shell` 模块权限（非 GET 请求）
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
