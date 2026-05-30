# SSH 主机管理 API

> WebSSH 模块支持通过浏览器直接连接远程 SSH 主机，提供主机配置管理（支持密码和私钥认证）和 WebSocket 终端会话。
> 主机配置独立存储于 `{rootDirectory}/webssh.yml`，不写入主配置文件。

---

## 主机管理

### 查询主机列表

```bash
isrvd_get "/ssh/hosts"
```

**响应字段（HostView）：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 主机 ID（只读） |
| `name` | string | 主机名称 |
| `addr` | string | 地址（`host` 或 `host:port`，默认端口 22） |
| `user` | string | SSH 用户名 |
| `passwordSet` | bool | 是否已设置密码（只读，密码不回显） |
| `privateKey` | string | SSH 私钥内容（PEM 格式） |
| `description` | string | 描述 |

---

### 获取主机详情

```bash
isrvd_get "/ssh/host/<ID>"
```

---

### 添加主机

```bash
# 密码认证
isrvd_post "/ssh/host" '{
  "name": "生产服务器",
  "addr": "192.168.1.100:22",
  "user": "root",
  "password": "your-password",
  "description": "生产环境主服务器"
}'

# 私钥认证
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
| `user` | string | ✓ | SSH 用户名 |
| `password` | string | | 密码（与 `privateKey` 二选一） |
| `privateKey` | string | | 私钥 PEM 内容（与 `password` 二选一） |
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

> **注意**：`password` 和 `privateKey` 为空时保留原值，无需每次都传入敏感信息。

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
