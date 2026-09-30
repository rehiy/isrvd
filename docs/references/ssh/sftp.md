# SFTP 文件管理 API

> 基于 SSH 主机配置，通过 SFTP 协议进行远程文件管理。所有接口复用主机认证信息，无需额外配置。

---

## 列出目录

```bash
isrvd_get "/sftp/<ID>/ls?path=/home/user"
```

响应 `payload`：`{path, files: SFTPFileInfo[]}`；`path` 为解析后的实际路径（传空或 `~` 时为远程 home 目录，取不到时为 `/`）。

**`files[]` 字段：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `name` | string | 文件/目录名称 |
| `size` | int64 | 文件大小（字节）；目录为文件系统返回的原始值（通常为 4096） |
| `mode` | string | 权限字符串（如 `-rw-r--r--`） |
| `modTime` | int64 | 修改时间（Unix 时间戳） |
| `isDir` | bool | 是否为目录（软链接目录也为 true） |
| `isLink` | bool | 是否为软链接 |
| `linkTarget` | string | 软链接指向的目标路径（仅软链接时存在） |

---

## 读取文件

```bash
isrvd_get "/sftp/<ID>/read?path=/path/to/file"
```

响应 `payload`：`{"content":"<FILE_CONTENT>"}`。文件超过 4 MiB 返回 413。

---

## 写入文件

```bash
isrvd_post "/sftp/<ID>/write" '{"path":"/path/to/file","content":"<FILE_CONTENT>"}'
```

`path`、`content` 均必填（空字符串 `content` 返回 400）；`content` 超过 4 MiB 返回 413。

---

## 下载文件

```bash
isrvd_get "/sftp/<ID>/download?path=/path/to/file"
```

返回原始文件流（未设置 `Content-Disposition: attachment`）；浏览器直连下载可携带 `token` 查询参数认证。

---

## 上传文件

```bash
isrvd_upload "/sftp/<ID>/upload?path=/remote/dir" "file" "/local/file.txt"
```

- 目标目录 `path` 必须通过 **URL Query** 传递，为空返回 400
- 可重复提交多个 `file` 字段；可选表单字段 `relativePath` 与 `file` 按顺序对应，含子目录时自动在远端创建
- 上传请求总大小受 `server.maxUploadSize` 限制

---

## 创建目录

```bash
isrvd_post "/sftp/<ID>/mkdir" '{"path":"/remote/new/dir"}'
```

---

## 删除文件或目录

```bash
isrvd_delete "/sftp/<ID>/rm?path=/remote/file/or/dir"
isrvd_delete "/sftp/<ID>/rm?path=/remote/dir&recursive=true"
```

---

## 重命名

```bash
isrvd_post "/sftp/<ID>/rename" '{"oldPath":"/remote/old","newPath":"/remote/new"}'
```

---

## 修改权限

```bash
isrvd_post "/sftp/<ID>/chmod" '{"path":"/remote/file","mode":"0644"}'
```

---

## 修改所有者

```bash
isrvd_post "/sftp/<ID>/chown" '{"path":"/remote/file","uid":1000,"gid":1000}'
```

`path`、`uid`、`gid` 均必填；由于必填校验，`uid` / `gid` 为 `0`（root）时返回 400。

---

## 计算目录大小

```bash
isrvd_get "/sftp/<ID>/dir-size?path=/remote/dir"
```

**响应字段：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `path` | string | 目录路径 |
| `size` | int64 | 目录总大小（字节） |
