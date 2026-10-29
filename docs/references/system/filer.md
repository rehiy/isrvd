# 文件管理 API

## 路径说明

`path` 一律相对当前成员 `homeDirectory`（成员不存在时为 `rootDirectory/share`），越界（含符号链接逃逸）返回 403"路径越界，拒绝访问"。这些路径只在 isrvd 所在环境有效：isrvd 运行在容器中时，把目录 volume mount 给其它容器不能直接用 filer 路径作为 hostPath，需先通过 `isrvd_get "/docker/container/<ISRVD_ID>"` 的 `mounts` 字段找到宿主机真实路径。

静态文件更新无需重建容器：用 `isrvd_put "/filer/file"` 或 `isrvd_upload` 写入，若 web 容器已正确挂载该目录，写入后立即生效。

## 计算目录大小

```bash
isrvd_get "/filer/dir-size?path=<DIR>"
```

返回：`{"path": "<DIR>", "size": <BYTES>}`

---

## 列出文件

```bash
isrvd_get "/filer/files?path=<DIR>"
```

`path` 必填（缺失返回 400），目录不存在返回 404。响应 `{path, files}`，目录在前、按名称排序，`files` 元素：

| 字段 | 类型 | 说明 |
|------|------|------|
| `name` | string | 文件名 |
| `path` | string | 相对主目录的路径 |
| `size` | number | 文件大小（字节） |
| `isDir` | boolean | 是否为目录；软链接指向目录时为 `true` |
| `isLink` | boolean | 是否为软链接 |
| `linkTarget` | string | 软链接目标；仅软链接时返回 |
| `mode` | string | 权限字符串，仅含权限位，如 `-rwxr-xr-x`（目录无 `d`） |
| `modeO` | string | 八进制权限，如 `755` |
| `modTime` | string | 最后修改时间（RFC3339） |

## 创建目录

```bash
isrvd_post "/filer/dir" '{"path":"<DIR>"}'
```

不递归创建：父目录必须存在；目录已存在时返回 500。

## 创建文件

```bash
isrvd_post "/filer/file" '{"path":"<FILE>","content":"<CONTENT>"}'
```

自动创建缺失的父目录；文件已存在则覆盖。

## 读取文件

```bash
isrvd_get "/filer/file?path=<FILE>"
```

返回：`{"path": "<FILE>", "content": "文件内容..."}`。文件超过 8 MiB 返回 413（请使用下载接口）；二进制文件（前 8000 字节含 NUL 字节或非打印字符超过 10%）返回 415；路径为目录返回 400，不存在返回 404，无权限返回 403，其他读取错误返回 500。

## 保存文件

```bash
isrvd_put "/filer/file" '{"path":"<FILE>","content":"<CONTENT>"}'
```

> 创建 / 保存文件时 `path` 必填，`content` 可为空字符串；`content` 超过 8 MiB、请求体超过 32 MiB 返回 413。

## 重命名 / 移动

```bash
# 重命名：target 为新名称或相对当前文件所在目录的路径
isrvd_post "/filer/rename" '{"path":"<OLD_PATH>","target":"<NEW_NAME>"}'

# 移动：target 可为相对当前文件所在目录的路径，也可为绝对 filer 路径
isrvd_post "/filer/rename" '{"path":"<OLD_PATH>","target":"<TARGET_DIR>/<NAME>"}'
isrvd_post "/filer/rename" '{"path":"<OLD_PATH>","target":"/<TARGET_DIR>/<NAME>"}'
```

`target` 由后端解析并校验用户目录边界；目标父目录不存在时会自动创建，越界路径会被拒绝。

## 删除

```bash
isrvd_delete "/filer/file?path=<FILE>"
```

目录递归删除；`path=/` 即主目录本身。

## 修改权限

```bash
isrvd_put "/filer/chmod" '{"path":"<FILE>","mode":"0644"}'
```

`mode` 须为八进制字符串，否则返回 400。

## 上传文件

```bash
isrvd_upload "/filer/upload" "file" "<LOCAL_FILE>" "path=<FILER_DIR>"
```

- 只处理单个 `file` 字段；`path` 为**表单字段**（目标目录），文件写入 `<path>/<原文件名>`，已存在时覆盖（流式写入）
- 目标父目录不会自动创建，不存在时返回 500
- `Content-Length` 超过 `server.maxUploadSize` 时返回 400"文件大小超过限制"

## 下载文件

```bash
isrvd_get "/filer/download?path=<FILE>"
```

返回 attachment 文件流，支持 HTTP Range 与 `?token=` 认证；文件不存在返回 404。

## 预览文件

```bash
isrvd_get "/filer/download?path=<FILE>&inline=1"
```

返回 inline 文件流，支持 HTTP Range。仅支持扩展名：jpg、jpeg、png、gif、bmp、svg、webp、ico、tiff、tif、mp3、wav、ogg、m4a、flac、aac、mp4、webm、mov、m4v、mkv、pdf，其他返回 415（先于文件存在检查）。

## 压缩

```bash
isrvd_post "/filer/zip" '{"path":"<DIR_OR_FILE>"}'
```

输出为 `<path>.zip`，位于源路径同目录。

## 解压

### 基础解压（默认解压到 zip 文件同级目录）

```bash
isrvd_post "/filer/unzip" '{"path":"<FILE>.zip"}'
```

### 解压到指定目录

```bash
isrvd_post "/filer/unzip" '{"path":"<FILE>.zip","targetDir":"<DIR_NAME>"}'
```

`targetDir` 仅允许输入标准目录名（不能包含 `/` 等路径分隔符），会自动在 zip 文件的当前目录下创建。

解压后，`<FILE>.zip` 文件会移回原位置。
