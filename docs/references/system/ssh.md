# WebSSH API

模块名：`ssh`。所有 API 均需 `ssh` 模块权限。

---

## 数据结构

### Credential（SSH 凭据）

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 唯一标识（UUID，只读） |
| `name` | string | 凭据名称 |
| `type` | string | 认证类型：`password` / `privateKey` |
| `username` | string | SSH 用户名 |
| `password` | string | 密码（`type=password` 时必填，存储时加密） |
| `privateKey` | string | 私钥内容（`type=privateKey` 时必填） |
| `passphrase` | string | 私钥密码（可选） |
| `description` | string | 描述（可选） |

### Host（SSH 主机）

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 唯一标识（UUID，只读） |
| `name` | string | 主机名称 |
| `host` | string | 主机地址（IP 或域名） |
| `port` | number | SSH 端口（默认 22） |
| `credentialId` | string | 关联凭据 ID |
| `description` | string | 描述（可选） |

---

## 路由列表

### 凭据管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/ssh/credentials` | 查询 SSH 凭据列表 |
| GET | `/api/ssh/credential/:id` | 获取 SSH 凭据详情 |
| POST | `/api/ssh/credential` | 添加 SSH 凭据 |
| PUT | `/api/ssh/credential/:id` | 更新 SSH 凭据 |
| DELETE | `/api/ssh/credential/:id` | 删除 SSH 凭据 |

### 主机管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/ssh/hosts` | 查询 SSH 主机列表 |
| GET | `/api/ssh/host/:id` | 获取 SSH 主机详情 |
| POST | `/api/ssh/host` | 添加 SSH 主机 |
| PUT | `/api/ssh/host/:id` | 更新 SSH 主机 |
| DELETE | `/api/ssh/host/:id` | 删除 SSH 主机 |

### SSH 终端

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/ssh/to/:id` | 打开 SSH 终端（WebSocket） |

### SFTP 文件管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/sftp/:id/ls` | SFTP 列出目录 |
| GET | `/api/sftp/:id/download` | SFTP 下载文件 |
| POST | `/api/sftp/:id/upload` | SFTP 上传文件 |
| DELETE | `/api/sftp/:id/rm` | SFTP 删除文件或目录 |
| POST | `/api/sftp/:id/mkdir` | SFTP 创建目录 |
| POST | `/api/sftp/:id/rename` | SFTP 重命名 |
| POST | `/api/sftp/:id/chmod` | SFTP 修改权限 |
| POST | `/api/sftp/:id/chown` | SFTP 修改所有者 |
| GET | `/api/sftp/:id/read` | SFTP 读取文件 |
| POST | `/api/sftp/:id/write` | SFTP 写入文件 |
| GET | `/api/sftp/:id/dir-size` | SFTP 计算目录大小 |

---

## 接口详情

### 查询凭据列表

**GET** `/api/ssh/credentials`

响应 `payload`：`[<Credential>, ...]`

> 注意：响应中不包含 `password`、`privateKey`、`passphrase` 等敏感字段。

---

### 获取凭据详情

**GET** `/api/ssh/credential/:id`

响应 `payload`：`<Credential>`

> 注意：响应中不包含 `password`、`privateKey`、`passphrase` 等敏感字段。

---

### 添加凭据

**POST** `/api/ssh/credential`

请求体：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | ✓ | 凭据名称 |
| `type` | string | ✓ | `password` 或 `privateKey` |
| `username` | string | ✓ | SSH 用户名 |
| `password` | string | 条件必填 | `type=password` 时必填 |
| `privateKey` | string | 条件必填 | `type=privateKey` 时必填 |
| `passphrase` | string | - | 私钥密码（可选） |
| `description` | string | - | 描述 |

响应 `payload`：`<Credential>`

---

### 更新凭据

**PUT** `/api/ssh/credential/:id`

请求体同添加凭据，响应 `payload`：`<Credential>`

---

### 删除凭据

**DELETE** `/api/ssh/credential/:id`

无请求体，删除成功返回 200。

---

### 查询主机列表

**GET** `/api/ssh/hosts`

响应 `payload`：`[<Host>, ...]`

---

### 获取主机详情

**GET** `/api/ssh/host/:id`

响应 `payload`：`<Host>`

---

### 添加主机

**POST** `/api/ssh/host`

请求体：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | ✓ | 主机名称 |
| `host` | string | ✓ | 主机地址 |
| `port` | number | - | SSH 端口（默认 22） |
| `credentialId` | string | ✓ | 关联凭据 ID |
| `description` | string | - | 描述 |

响应 `payload`：`<Host>`

---

### 更新主机

**PUT** `/api/ssh/host/:id`

请求体同添加主机，响应 `payload`：`<Host>`

---

### 删除主机

**DELETE** `/api/ssh/host/:id`

无请求体，删除成功返回 200。

---

### 打开 SSH 终端

**GET** `/api/ssh/to/:id`

通过 WebSocket 连接到远程 SSH 主机终端。
前端窗口尺寸变化时应发送 resize 控制帧，格式为 `\u0000isrvd:resize:<cols>:<rows>`，服务端会同步远程 SSH PTY 尺寸。

**Query 参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| `token` | string | JWT 认证令牌（必需） |

**前端集成示例：**

```javascript
const ws = new WebSocket(`ws://${host}/api/ssh/to/${hostId}?token=${token}`);

ws.onmessage = (event) => {
  terminal.write(event.data);
};

terminal.onData((data) => {
  ws.send(data);
});

terminal.onResize(({ cols, rows }) => {
  ws.send(`\u0000isrvd:resize:${cols}:${rows}`);
});
```

---

### SFTP 列出目录

**GET** `/api/sftp/:id/ls?path=<DIR>`

Query 参数：

| 参数 | 说明 |
|------|------|
| `path` | 远程目录路径（默认 `~`） |

响应 `payload`：`{files: [...]}`

---

### SFTP 下载文件

**GET** `/api/sftp/:id/download?path=<FILE>`

返回文件流（attachment），需携带 `token` 参数认证。

---

### SFTP 上传文件

**POST** `/api/sftp/:id/upload?path=<DIR>`

Multipart 表单上传，字段名 `file`。支持目录上传（`webkitRelativePath`）。

---

### SFTP 删除

**DELETE** `/api/sftp/:id/rm?path=<FILE>&recursive=true`

Query 参数：

| 参数 | 说明 |
|------|------|
| `path` | 远程文件或目录路径（必需） |
| `recursive` | 是否为递归删除（目录删除时需 `true`） |

---

### SFTP 创建目录

**POST** `/api/sftp/:id/mkdir`

请求体：

```json
{"path": "<REMOTE_DIR>"}
```

---

### SFTP 重命名

**POST** `/api/sftp/:id/rename`

请求体：

```json
{"oldPath": "<OLD_PATH>", "newPath": "<NEW_PATH>"}
```

---

### SFTP 修改权限

**POST** `/api/sftp/:id/chmod`

请求体：

```json
{"path": "<FILE>", "mode": "0755"}
```

---

### SFTP 修改所有者

**POST** `/api/sftp/:id/chown`

请求体：

```json
{"path": "<FILE>", "uid": 1000, "gid": 1000}
```

---

### SFTP 读取文件

**GET** `/api/sftp/:id/read?path=<FILE>`

响应 `payload`：`{content: "<FILE_CONTENT>"}`

---

### SFTP 写入文件

**POST** `/api/sftp/:id/write`

请求体：

```json
{"path": "<FILE>", "content": "<CONTENT>"}
```

---

### SFTP 计算目录大小

**GET** `/api/sftp/:id/dir-size?path=<DIR>`

响应 `payload`：`{path: "<DIR>", size: <BYTES>}`

---

## 与 Shell 的区别

| 特性 | Shell | WebSSH |
|------|-------|--------|
| 连接目标 | isrvd 服务器本地 | 远程 SSH 主机 |
| 配置要求 | 无（开箱即用） | 需配置 SSH 主机和凭据 |
| 认证方式 | isrvd JWT | SSH 密码/私钥 |
| 适用场景 | 服务器本地管理 | 远程服务器管理 |
| WebSocket 路径 | `/api/shell` | `/api/ssh/to/<ID>` |
| 文件管理 | `/api/filer/*`（本地） | `/api/sftp/:id/*`（远程） |
