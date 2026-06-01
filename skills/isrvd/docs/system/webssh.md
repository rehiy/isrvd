# WebSSH 主机管理与终端

## 概述

WebSSH 模块支持通过浏览器直接连接远程 SSH 主机，提供可复用认证凭据管理、主机配置管理（支持密码和私钥认证）、WebSocket 终端会话和 SFTP 文件管理。

主机配置独立存储于 `{rootDirectory}/webssh.yml`，认证凭据独立存储于 `{rootDirectory}/webssh-credentials.yml`，均不写入主配置文件。

---

## 认证凭据管理

认证凭据可被多台 SSH 主机复用。凭据详情接口会返回 `password` 和 `privateKey` 字段；列表接口不回显敏感内容，仅返回 `authType` 表示认证类型。

### 查询凭据列表

```bash
isrvd_get "/ssh/credentials"
```

**响应字段（CredentialView）：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 凭据 ID（只读） |
| `name` | string | 凭据名称 |
| `description` | string | 描述 |
| `user` | string | SSH 用户名 |
| `authType` | string | 认证类型：`password` / `privateKey` / 空字符串（只读） |

---

### 获取凭据详情

```bash
isrvd_get "/ssh/credential/<ID>"
```

**响应字段（Credential）：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 凭据 ID（只读） |
| `name` | string | 凭据名称 |
| `description` | string | 描述 |
| `user` | string | SSH 用户名 |
| `password` | string | SSH 密码 |
| `privateKey` | string | SSH 私钥内容（PEM 格式，优先于密码） |

---

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

**请求字段（CredentialUpsertRequest）：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | ✓ | 凭据名称 |
| `description` | string | | 描述 |
| `user` | string | ✓ | SSH 用户名 |
| `password` | string | | SSH 密码（与 `privateKey` 二选一） |
| `privateKey` | string | | SSH 私钥内容（PEM 格式，优先于密码） |

---

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

> **注意**：凭据更新接口会按请求体整体覆盖 `password` 和 `privateKey`；如需保留原值，先通过详情接口读取后再提交。

---

### 删除凭据

```bash
isrvd_delete "/ssh/credential/<ID>"
```

> **注意**：删除凭据不会自动删除已绑定该凭据的主机；这些主机后续连接会因凭据不存在而失败，需要切换到其它凭据或改为手动认证。

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
| `credentialId` | string | 绑定的认证凭据 ID（可选） |
| `credentialName` | string | 绑定的认证凭据名称（只读，可选） |
| `user` | string | SSH 用户名 |
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

**请求字段（HostUpsertRequest）：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | ✓ | 主机名称 |
| `addr` | string | ✓ | 地址（`host` 或 `host:port`） |
| `credentialId` | string | | 绑定的认证凭据 ID；设置后优先使用该凭据认证 |
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
| `isDir` | bool | 是否为目录（软链接目录也为 true） |
| `isLink` | bool | 是否为软链接 |
| `linkTarget` | string | 软链接指向的目标路径（仅软链接时存在） |

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

**Web UI 上传行为：**

- 上传列表支持并发上传，默认并发数为 3。
- 点击取消会中断尚未完成的前端上传请求，并停止该任务后续未开始的目录创建和文件上传队列。
- 已经进入服务端处理的请求会尽快停止；如果服务端已完成接收并开始远端 SFTP 写入，仍可能短暂继续执行。
- 已取消的文件会保留在上传列表中，可点击清理按钮移除；目录清理会递归移除已取消文件，并删除因此变空的目录节点。
- 上传失败的文件会保留在上传列表中，可点击重试按钮重新上传失败项。
- 上传失败后前端会尝试删除远端同名残留文件；如果清理失败，会在错误提示中说明远端可能残留不完整文件。

---

### 删除文件或目录

```bash
# 删除文件
isrvd_delete "/ssh/sftp/<ID>/rm?path=/path/to/file"
# 删除空目录
isrvd_delete "/ssh/sftp/<ID>/rm?path=/path/to/dir"
# 递归删除目录（含所有内容，等同于 rm -rf）
isrvd_delete "/ssh/sftp/<ID>/rm?path=/path/to/dir&recursive=true"
```

**Query 参数：**

| 参数 | 必填 | 说明 |
|------|------|------|
| `path` | ✓ | 目标路径 |
| `recursive` | | `true` 时递归删除目录及其所有内容，默认 `false` |

> **注意**：`recursive=false`（默认）时，目录必须为空才能删除。

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

### 修改权限

```bash
isrvd_post "/ssh/sftp/<ID>/chmod" '{
  "path": "/path/to/file",
  "mode": "0644"
}'
```

**请求字段：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `path` | string | ✓ | 文件或目录路径 |
| `mode` | string | ✓ | 3-4 位八进制权限，如 `755`、`0644` |

---

### 修改所有者

```bash
isrvd_post "/ssh/sftp/<ID>/chown" '{
  "path": "/path/to/file",
  "uid": 1000,
  "gid": 1000
}'
```

**请求字段：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `path` | string | ✓ | 文件或目录路径 |
| `uid` | int | ✓ | 用户 ID |
| `gid` | int | ✓ | 用户组 ID |

---

### 读取文件

```bash
isrvd_get "/ssh/sftp/<ID>/read?path=/path/to/file"
```

**响应字段：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `content` | string | 文件内容 |

---

### 写入文件

```bash
isrvd_post "/ssh/sftp/<ID>/write" '{
  "path": "/path/to/file",
  "content": "文件内容"
}'
```

**请求字段：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `path` | string | ✓ | 文件路径 |
| `content` | string | ✓ | 文件内容 |

---

### 计算目录大小

```bash
isrvd_get "/ssh/sftp/<ID>/dir-size?path=/path/to/dir"
```

**响应字段：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `path` | string | 目录路径 |
| `size` | int64 | 目录总大小（字节） |

---

## 存储说明

认证凭据存储于 `{rootDirectory}/webssh-credentials.yml`，格式示例：

```yaml
- id: 01j...
  name: 生产环境 root
  description: 生产环境通用 root 凭据
  user: root
  password: your-password
- id: 01j...
  name: 开发环境 ubuntu
  description: 开发环境通用私钥
  user: ubuntu
  privateKey: |
    -----BEGIN PRIVATE KEY-----
    ...
    -----END PRIVATE KEY-----
```

主机配置存储于 `{rootDirectory}/webssh.yml`，格式示例：

```yaml
- id: 01j...
  name: 生产服务器
  addr: 192.168.1.100:22
  credentialId: 01j...
  user: root
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

> 两个文件权限均为 `0600`，密码和私钥以明文存储，请确保文件系统权限安全。绑定 `credentialId` 的主机不会在 `webssh.yml` 中冗余保存凭据密码或私钥，连接时从 `webssh-credentials.yml` 解析认证信息。
