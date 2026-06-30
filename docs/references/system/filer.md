# 文件管理 API

> 读取类接口为 GET 方法，写入与变更类接口遵循 RESTful 规范

## ⚠️ 重要：filer 路径 ≠ 宿主机路径

filer 管理的是 **isrvd 容器内部**挂载的卷，这些路径只在 isrvd 容器内有效。

如果要把这些目录 volume mount 给其它容器，**不能直接用 filer 路径作为 hostPath**，必须先找出它们在宿主机上的真实路径：

```bash
# 查看 isrvd 容器自身的 volume 挂载，找到 filer 路径对应的宿主机路径
# 1. 从 isrvd_get "/docker/containers" 结果中找到 isrvd 容器的 id
# 2. 用 isrvd_get "/docker/container/<ISRVD_ID>" 查看 mounts 字段
```

**正确的静态文件更新流程**（不需要重建容器）：
1. 先查出 filer 可用目录：`isrvd_get "/filer/files?path=/"`
2. 用 filer 写文件：`isrvd_put "/filer/file" '{"path":"<FILER_PATH>/<FILE>","content":"..."}'`
3. 确认 web 容器是否已经用正确的 hostPath 挂载了该目录（只在首次部署时配置一次）
4. 文件写入后立即生效，无需重启

**禁止用 base64**：不要用 `base64 -d` 方式在 Dockerfile 或 RUN 命令里写入文件内容。应使用 `isrvd_upload` 或直接 `isrvd_put "/filer/file"` 写入。

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

响应中的 `files` 字段包含：

| 字段 | 类型 | 说明 |
|------|------|------|
| `name` | string | 文件名 |
| `path` | string | filer 内路径 |
| `size` | number | 文件大小（字节） |
| `isDir` | boolean | 是否为目录；软链接指向目录时为 `true` |
| `isLink` | boolean | 是否为软链接 |
| `linkTarget` | string | 软链接目标；仅软链接时返回 |
| `mode` | string | 权限字符串 |
| `modeO` | string | 八进制权限 |
| `modTime` | string | 最后修改时间 |

## 创建目录

```bash
isrvd_post "/filer/dir" '{"path":"<DIR>"}'
```

## 创建文件

```bash
isrvd_post "/filer/file" '{"path":"<FILE>","content":"<CONTENT>"}'
```

## 读取文件

```bash
isrvd_get "/filer/file?path=<FILE>"
```

返回：`{"content": "文件内容..."}`

## 保存文件

```bash
isrvd_put "/filer/file" '{"path":"<FILE>","content":"<CONTENT>"}'
```

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

## 修改权限

```bash
isrvd_put "/filer/chmod" '{"path":"<FILE>","mode":"0644"}'
```

## 上传文件

```bash
isrvd_upload "/filer/upload" "file" "<LOCAL_FILE>" "path=<FILER_DIR>"
```

## 下载文件

```bash
isrvd_get "/filer/download?path=<FILE>"
```

返回 attachment 文件流，支持 HTTP Range。

## 预览文件

```bash
isrvd_get "/filer/download?path=<FILE>&inline=1"
```

返回 inline 文件流，支持 HTTP Range，适用于图片、音频、视频、PDF 等浏览器可直接预览的文件；不支持的扩展名返回 415。

## 压缩

```bash
isrvd_post "/filer/zip" '{"path":"<DIR_OR_FILE>"}'
```

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
