# Docker 镜像 API

## 列出镜像

```bash
isrvd_get "/docker/images"
```

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 镜像 ID |
| shortId | string | 短 ID |
| repoTags | string[] | 标签列表 |
| repoDigests | string[] | Digest 列表 |
| size | number | 大小（字节） |
| created | number | 创建时间戳 |

## 查看镜像详情

```bash
isrvd_get "/docker/image/<IMAGE_ID>"
```

| 字段 | 类型 | 说明 |
|------|------|------|
| repoTags | string[] | 标签 |
| size | number | 大小 |
| created | string | 创建时间 |
| architecture | string | 架构 |
| os | string | 操作系统 |
| cmd | string[] | 默认命令 |
| entrypoint | string[] | 入口命令 |
| env | string[] | 环境变量 |
| exposedPorts | string[] | 暴露端口 |
| labels | object | 标签 |
| layers | number | 层数 |
| layerDetails | object[] | `{digest, createdBy, created, size, empty}` |

## 搜索镜像

```bash
isrvd_get "/docker/images/search?name=<KEYWORD>"
```

## 构建镜像

```bash
isrvd_post "/docker/image/build" '{"dockerfile":"<DOCKERFILE_CONTENT>","tag":"<IMAGE>:<TAG>"}'
```

## 镜像打标签

```bash
isrvd_post "/docker/image/<IMAGE_ID>/tag" '{"repoTag":"<REPO>/<NAME>:<TAG>"}'
```

## 镜像删除

```bash
isrvd_post "/docker/image/<IMAGE_ID>/action" '{"action":"remove"}'
```

## 清理未使用镜像

```bash
# 清理所有未被容器引用的镜像（含有 tag 但闲置的）—— UI 默认行为
isrvd_post "/docker/image/prune" '{"all":true}'

# 仅清理悬空层（未打 tag 的中间/失标层）
isrvd_post "/docker/image/prune" '{"all":false}'

# 按时间过滤：仅清理 24 小时前创建的镜像
isrvd_post "/docker/image/prune" '{"all":true,"until":"24h"}'
```

请求字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| all | bool | `true`（UI 默认）清理所有未被容器引用的镜像；`false` 仅清理悬空层 |
| until | string | 仅清理在该时间之前创建的镜像（Docker filters 语法，如 `24h`、`2024-01-01T00:00:00`） |

响应字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| imagesDeleted | object[] | 删除条目，元素含 `untagged` 与/或 `deleted` |
| imagesDeleted[].untagged | string | 被解除的 tag 引用 |
| imagesDeleted[].deleted | string | 被删除的镜像层 ID（`sha256:...`） |
| spaceReclaimed | number | 回收磁盘空间（字节） |

## 拉取镜像

```bash
isrvd_post "/docker/image/pull" '{"image":"<IMAGE>"}'
isrvd_post "/docker/image/pull" '{"image":"<IMAGE>","registryUrl":"<REGISTRY_URL>","namespace":"<NS>"}'
```

> `registryUrl` 为空则从 Docker Hub 拉取

## 推送镜像

```bash
isrvd_post "/docker/image/push" '{"image":"<IMAGE>","registryUrl":"<REGISTRY_URL>","namespace":"<NS>"}'
```
