# Docker 镜像 API

## 列出镜像

```bash
isrvd_get "/docker/images"
isrvd_get "/docker/images?all=true"
```

| 参数 | 类型 | 说明 |
|------|------|------|
| all | boolean | 默认 `false`，此时不返回没有 tag 的镜像 |

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 镜像 ID |
| shortId | string | 短 ID |
| repoTags | string[] | 标签列表 |
| repoDigests | string[] | Digest 列表 |
| size | number | 大小（字节） |
| created | number | 创建时间戳 |

## 搜索镜像

```bash
isrvd_get "/docker/images/search?name=<KEYWORD>"
```

`name` 必填（为空返回 500"搜索关键词不能为空"），最多返回 25 条：

| 字段 | 类型 | 说明 |
|------|------|------|
| name | string | 镜像名 |
| description | string | 描述 |
| isOfficial | boolean | 是否官方镜像 |
| starCount | number | 星标数 |

## 查看镜像详情

```bash
isrvd_get "/docker/image/<IMAGE_ID>"
```

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 镜像完整 ID |
| shortId | string | 短 ID |
| repoTags | string[] | 标签 |
| repoDigests | string[] | Digest 列表 |
| size | number | 大小（字节） |
| created | string | 创建时间 |
| author | string | 作者 |
| architecture | string | 架构 |
| os | string | 操作系统 |
| cmd | string[] | 默认命令 |
| entrypoint | string[] | 入口命令 |
| env | string[] | 环境变量 |
| workingDir | string | 工作目录 |
| exposedPorts | string[] | 暴露端口 |
| labels | object | 标签 |
| layers | number | 层数 |
| layerDetails | object[] | `{digest, createdBy, created, size, empty}`；`created` 为 RFC3339 UTC 时间，空层的 `digest` 为空字符串 |

## 构建镜像

```bash
isrvd_post "/docker/image/build" '{"dockerfile":"<DOCKERFILE_CONTENT>","tag":"<IMAGE>:<TAG>"}'
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| dockerfile | string | ✅ | Dockerfile 内容（缺失返回 400） |
| tag | string | ✅ | 构建产物标签（为空返回 500"镜像标签不能为空"） |

返回 `{tag, message}`，`message` 为构建输出的最后一行。

## 镜像打标签

```bash
isrvd_post "/docker/image/<IMAGE_ID>/tag" '{"repoTag":"<REPO>/<NAME>:<TAG>"}'
```

`repoTag` 必填。

## 拉取镜像

```bash
isrvd_post "/docker/image/pull" '{"image":"<IMAGE>"}'
isrvd_post "/docker/image/pull" '{"image":"<IMAGE>","registryUrl":"<REGISTRY_URL>","namespace":"<NS>"}'
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| image | string | ✅ | 镜像名，未带 tag 时补 `:latest` |
| registryUrl | string | | 来源仓库；为空时从 Docker Hub（或 daemon 配置的 mirror）拉取，此时忽略 `namespace` |
| namespace | string | | 仓库命名空间 |

返回 `{image, message}`，`image` 为实际拉取的完整引用。

## 推送镜像

```bash
isrvd_post "/docker/image/push" '{"image":"<IMAGE>","registryUrl":"<REGISTRY_URL>","namespace":"<NS>"}'
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| image | string | ✅ | 本地镜像（repo:tag） |
| registryUrl | string | ✅ | 目标仓库地址（需已在 `/docker/registries` 中配置） |
| namespace | string | | 目标命名空间 |

推送时取 `image` 最后一个 `/` 之后的部分，重新打标签为 `<registryHost>[/<namespace>]/<name>` 后推送；目标引用无 `:` 时补 `:latest`。返回 `{image, target, message}`。

## 镜像删除

```bash
isrvd_post "/docker/image/<IMAGE_ID>/action" '{"action":"remove"}'
```

仅支持 `remove`：强制删除（含被已停止容器引用或有多个 tag 的镜像），并清理无用父层。

## 清理未使用镜像

```bash
# 清理所有未被容器引用的镜像（含有 tag 但闲置的）—— UI 默认行为
isrvd_post "/docker/image/prune" '{"all":true}'

# 仅清理悬空层（未打 tag 的中间/失标层）
isrvd_post "/docker/image/prune" '{"all":false}'

# 按时间过滤：仅清理 24 小时前创建的镜像
isrvd_post "/docker/image/prune" '{"all":true,"until":"24h"}'
```

请求字段（请求体可省略，此时等同 `{"all":false}`）：

| 字段 | 类型 | 说明 |
|------|------|------|
| all | bool | `true`（UI 默认）清理所有未被容器引用的镜像；`false`（接口默认）仅清理悬空层 |
| until | string | 仅清理在该时间之前创建的镜像（Docker filters 语法，如 `24h`、`2024-01-01T00:00:00`） |

响应字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| imagesDeleted | object[] | 删除条目，元素含 `untagged` 与/或 `deleted` |
| imagesDeleted[].untagged | string | 被解除的 tag 引用 |
| imagesDeleted[].deleted | string | 被删除的镜像层 ID（`sha256:...`） |
| spaceReclaimed | number | 回收磁盘空间（字节） |
