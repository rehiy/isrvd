# SSH 主机管理 API

> WebSSH 模块支持通过浏览器直接连接远程 SSH 主机，提供可复用认证凭据管理、主机配置管理（支持凭据复用、密码和私钥认证）和 WebSocket 终端会话。
> 主机配置独立存储于 `{rootDirectory}/webssh-host.yml`，认证凭据独立存储于 `{rootDirectory}/webssh-cred.yml`，均不写入主配置文件；配置位于 etcd 时改存 `<配置 key>/<文件名>`，首次读取时自动迁移本地同名文件。
> `password`、`privateKey` 加密落盘（密钥由 `server.jwtSecret` 派生）。⚠️ 修改 `jwtSecret` 后已加密的认证信息无法解密，WebSSH 服务初始化失败，`/ssh/*`、`/sftp/*` 整体返回 `503`，无法通过 API 重新填写；需恢复原 `jwtSecret`，或手工清除 `webssh-host.yml` / `webssh-cred.yml` 中的 `password` / `privateKey` 后再重新填写。
>
> 权限按路由 `METHOD /api/path` 逐条授予（创始人不受限）。

---

## 认证凭据管理

### 查询凭据列表

```bash
isrvd_get "/ssh/credentials"
```

**响应字段：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 凭据 ID（只读） |
| `name` | string | 凭据名称 |
| `description` | string | 描述 |
| `user` | string | SSH 用户名 |
| `authType` | string | 认证方式（只读）：`"password"` \| `"privateKey"`；未设置认证时不返回该字段 |

> `password` 和 `privateKey` 为敏感字段，不在响应中返回。
>
> 添加/更新凭据时：`name`、`user` 必填；填了 `privateKey` 会清空 `password`，只填 `password` 会清空 `privateKey`；更新时两者都留空则保留原值。

### 获取凭据详情

```bash
isrvd_get "/ssh/credential/<ID>"
```

### 添加凭据

```bash
isrvd_post "/ssh/credential" '{
  "name": "生产环境 root",
  "description": "生产环境通用 root 凭据",
  "user": "root",
  "password": "your-password",
  "privateKey": ""
}'
```

### 更新凭据

```bash
isrvd_put "/ssh/credential/<ID>" '{
  "name": "生产环境 root",
  "description": "更新后的描述",
  "user": "root",
  "password": "new-password",
  "privateKey": ""
}'
```

### 删除凭据

```bash
isrvd_delete "/ssh/credential/<ID>"
```

---

## 主机管理

### 查询主机列表

```bash
isrvd_get "/ssh/hosts"
```

**响应字段：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 主机 ID（只读） |
| `name` | string | 主机名称 |
| `addr` | string | 地址（`host` 或 `host:port`，默认端口 22） |
| `credentialId` | string | 绑定的认证凭据 ID（可选） |
| `credentialName` | string | 绑定的认证凭据名称（只读，可选） |
| `user` | string | SSH 用户名；绑定凭据时为保存时复制的凭据用户名快照，实际连接以凭据当前的 `user` 为准 |
| `description` | string | 描述 |

---

### 获取主机详情

```bash
isrvd_get "/ssh/host/<ID>"
```

---

### 添加主机

```bash
# 复用已保存凭据
isrvd_post "/ssh/host" '{
  "name": "生产服务器",
  "addr": "192.168.1.100:22",
  "credentialId": "<CREDENTIAL_ID>",
  "description": "生产环境主服务器"
}'

# 手动密码认证
isrvd_post "/ssh/host" '{
  "name": "生产服务器",
  "addr": "192.168.1.100:22",
  "user": "root",
  "password": "your-password",
  "description": "生产环境主服务器"
}'

# 手动私钥认证
isrvd_post "/ssh/host" '{
  "name": "开发服务器",
  "addr": "dev.example.com",
  "user": "ubuntu",
  "privateKey": "-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----",
  "description": "开发环境"
}'
```

**请求字段：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | ✓ | 主机名称 |
| `addr` | string | ✓ | 地址（`host` 或 `host:port`） |
| `credentialId` | string | | 绑定的认证凭据 ID；设置后校验凭据存在（不存在返回 400），使用凭据认证并丢弃主机上的 `password` / `privateKey` |
| `user` | string | | SSH 用户名；`credentialId` 为空时使用 |
| `password` | string | | SSH 密码；`credentialId` 为空时与 `privateKey` 二选一 |
| `privateKey` | string | | SSH 私钥内容（PEM 格式）；`credentialId` 为空时优先于密码 |
| `description` | string | | 描述 |

---

### 更新主机

```bash
isrvd_put "/ssh/host/<ID>" '{
  "name": "生产服务器",
  "addr": "192.168.1.100:22",
  "user": "root",
  "description": "已更新描述"
}'
```

> **注意**：`credentialId` 不为空时，主机通过对应凭据认证；`credentialId` 为空时使用手动认证信息。手动认证模式下，`password` 和 `privateKey` 为空时保留原值，无需每次都传入敏感信息。

---

### 删除主机

```bash
isrvd_delete "/ssh/host/<ID>"
```

---

## WebSSH 终端

通过 WebSocket 连接到指定主机的 SSH 终端：

```
GET /api/ssh/to/<ID>  (WebSocket, 支持 ?token= 查询参数)
```

```bash
# 示例：使用 wscat 连接（需先获取 token）
TOKEN=$(isrvd_login "$ISRVD_APIURL" "$ISRVD_USERNAME" "$ISRVD_PASSWORD" | jq -r '.payload.token')
wscat -c "ws://<HOST>/api/ssh/to/<ID>?token=$TOKEN"
```

> 终端 WSS 连接内置约 25s 保活心跳，空闲时不会被中间层（nginx/Caddy、NAT）断开。
> 前端窗口尺寸变化时应发送 resize 控制帧，格式为 `\u0000isrvd:resize:<cols>:<rows>`；服务端会同步远程 SSH PTY 尺寸。
