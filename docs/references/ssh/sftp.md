# SFTP 文件管理 API

> 基于 SSH 主机配置，通过 SFTP 协议进行远程文件管理。所有接口复用主机认证信息，无需额外配置。

---

## 列出目录

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

## 读取文件

```
GET /api/ssh/sftp/<ID>/read?path=/path/to/file
```

返回文件内容（text/plain 或 application/octet-stream）。

---

## 下载文件

```
GET /api/ssh/sftp/<ID>/download?path=/path/to/file  (支持 ?token= 查询参数)
```

返回 attachment 文件流，支持 HTTP Range。

---

## 上传文件

```bash
isrvd_upload "/ssh/sftp/<ID>/upload" "file" "/local/file.txt" "path=/remote/dir"
```

---

## 创建目录

```bash
isrvd_post "/ssh/sftp/<ID>/mkdir" '{"path":"/remote/new/dir"}'
```

---

## 删除文件或目录

```bash
isrvd_post "/ssh/sftp/<ID>/rm" '{"path":"/remote/file/or/dir"}'
```

---

## 重命名

```bash
isrvd_post "/ssh/sftp/<ID>/rename" '{"path":"/remote/old","target":"/remote/new"}'
```

---

## 修改权限

```bash
isrvd_post "/ssh/sftp/<ID>/chmod" '{"path":"/remote/file","mode":"0644"}'
```

---

## 修改所有者

```bash
isrvd_post "/ssh/sftp/<ID>/chown" '{"path":"/remote/file","uid":1000,"gid":1000}'
```

---

## 计算目录大小

```bash
isrvd_get "/ssh/sftp/<ID>/dir-size?path=/remote/dir"
```

**响应字段：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `path` | string | 目录路径 |
| `size` | int64 | 目录总大小（字节） |
| `count` | int | 文件总数 |
