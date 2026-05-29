# WebSSH 主机管理与终端

## 概述

WebSSH 模块支持通过浏览器直接连接远程 SSH 主机，提供主机配置管理（支持密码和私钥认证）和 WebSocket 终端会话。

主机配置独立存储于 `{rootDirectory}/webssh.yml`，不写入主配置文件。

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

---

## SFTP 文件管理

基于 SSH 主机配置，通过 SFTP 协议进行远程文件管理。所有接口复用主机认证信息，无需额外配置。

### 列出目录

```bash
isrvd_get "/ssh/sftp/<ID>/ls?path=/home/user"
```

**响应字段（SFTPFileInfo[]）：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `name` | string | 文件/目录名称 |
| `size` | int64 | 文件大小（字节），目录为 0 |
| `mode` | string | 权限字符串（如 `-rw-r--r--`） |
| `modTime` | int64 | 修改时间（Unix 时间戳） |
| `isDir` | bool | 是否为目录 |

---

### 下载文件

```
GET /api/ssh/sftp/<ID>/download?path=/path/to/file  (支持 ?token= 查询参数)
```

---

### 上传文件

```bash
isrvd_upload "/ssh/sftp/<ID>/upload" "file" "/local/file.txt" "path=/remote/dir"
```

**Query 参数：**

| 参数 | 必填 | 说明 |
|------|------|------|
| `path` | ✓ | 目标目录路径 |

**Form 字段：**

| 字段 | 说明 |
|------|------|
| `file` | 上传的文件（multipart） |

---

### 删除文件或目录

```bash
# 删除文件
isrvd_delete "/ssh/sftp/<ID>/rm?path=/path/to/file"
# 删除空目录
isrvd_delete "/ssh/sftp/<ID>/rm?path=/path/to/dir"
```

> **注意**：目录必须为空才能删除。

---

### 创建目录

```bash
isrvd_post "/ssh/sftp/<ID>/mkdir" '{"path": "/path/to/newdir"}'
```

支持多级目录（等同于 `mkdir -p`）。

---

### 重命名/移动

```bash
isrvd_post "/ssh/sftp/<ID>/rename" '{
  "oldPath": "/path/to/old",
  "newPath": "/path/to/new"
}'
```

---

## 存储说明

主机配置存储于 `{rootDirectory}/webssh.yml`，格式示例：

```yaml
- id: 01j...
  name: 生产服务器
  addr: 192.168.1.100:22
  user: root
  password: your-password
  description: 生产环境主服务器
- id: 01j...
  name: 开发服务器
  addr: dev.example.com
  user: ubuntu
  privateKey: |
    -----BEGIN PRIVATE KEY-----
    ...
    -----END PRIVATE KEY-----
  description: 开发环境
```

> 文件权限为 `0600`，密码和私钥以明文存储，请确保文件系统权限安全。
