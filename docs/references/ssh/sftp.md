# SFTP 文件管理 API

> 基于 SSH 主机配置，通过 SFTP 协议进行远程文件管理，复用主机认证信息。除 `dir-size` 失败返回 500 外，其余错误（主机不存在、SSH 连接失败、路径错误、缺少 `path`）统一返回 400。

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
| `mode` | string | 权限字符串（如 `-rw-r--r--`、`drwxr-xr-x`、符号链接 `Lrwxrwxrwx`） |
| `modTime` | int64 | 修改时间（Unix 时间戳） |
| `isDir` | bool | 是否为目录（软链接目录也为 true） |
| `isLink` | bool | 是否为软链接 |
| `linkTarget` | string | 软链接指向的目标路径（仅软链接时存在） |

---

## 读取文件

```bash
isrvd_get "/sftp/<ID>/read?path=/path/to/file"
```

响应 `payload`：`{"content":"<FILE_CONTENT>"}`。文件超过 8 MiB 返回 413；二进制文件（前 8000 字节含 NUL 字节或非打印字符超过 10%）返回 415；其余非 UTF-8 字节被替换为 U+FFFD。

---

## 写入文件

```bash
isrvd_post "/sftp/<ID>/write" '{"path":"/path/to/file","content":"<FILE_CONTENT>"}'
```

`path` 必填，`content` 可为空字符串；`content` 超过 8 MiB 返回 413。

页面的新建文件操作会传 `?createOnly=true`。同名时返回 409，原文件保持原样。新建使用 SFTP 的普通重命名操作发布暂存文件。

编辑与上传共用同目录暂存。文件写完并关闭后才重命名到目标。覆盖已有文件要求远端目录可写且服务器支持 `posix-rename@openssh.com`，并检查原文件写权限，保留原权限位、所有者和组。覆盖会替换文件 inode，其他硬链接仍指向旧 inode。失败或取消时，后端清理本次创建的暂存文件。远端断开且无法重新连接时，可能留下以 `.isrvd-upload-` 开头的暂存文件。如果在重命名期间断开连接，客户端需刷新列表确认远端结果。

---

## 下载文件

```bash
isrvd_get "/sftp/<ID>/download?path=/path/to/file"
```

返回原始文件流和文件长度。浏览器直连下载可携带 `token` 查询参数认证。传输开始前的读取错误返回 JSON，发送部分文件后发生的错误由客户端通过长度不符识别。每个 SFTP 操作使用独立连接，取消操作会关闭该连接。连接建立后，远端连续 30 秒无数据时中止传输。

---

## 上传文件

```bash
isrvd_upload "/sftp/<ID>/upload?path=/remote/dir" "file" "/local/file.txt"
```

- 目标目录 `path` 必须通过 **URL Query** 传递，为空返回 400
- 可重复提交多个 `file` 字段；可选表单字段 `relativePath` 与 `file` 按顺序对应，含子目录时自动在远端创建
- 上传请求总大小受 `server.maxUploadSize` 限制
- 页面显示「服务器写入中」表示浏览器已发送表单。服务器完成远端文件提交后，页面才显示成功
- 批量上传逐个提交文件。后续文件失败时，已经成功提交的文件仍保留

---

## 创建目录

```bash
isrvd_post "/sftp/<ID>/mkdir" '{"path":"/remote/new/dir"}'
```

递归创建，已存在不报错。

---

## 删除文件或目录

```bash
isrvd_delete "/sftp/<ID>/rm?path=/remote/file/or/dir"
isrvd_delete "/sftp/<ID>/rm?path=/remote/dir&recursive=true"
```

未指定 `recursive=true`（仅字面量 `true` 有效）时只能删除文件或空目录。

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

`mode` 须为 3 或 4 位八进制字符串（如 `644`、`0644`），否则返回 400。

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
| `size` | int64 | 目录总大小（字节）；不计目录自身，跳过符号链接 |
