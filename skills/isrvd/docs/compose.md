# Compose 部署 API

> 所有接口前缀: `/api/compose`  
> 只读操作需要 `compose:r` 权限，写操作需要 `compose:rw` 权限  
> Compose 功能依赖 Docker 引擎可用

---

## §1 Docker Compose 部署（单机）

### §1.1 部署

```
POST /api/compose/docker/deploy
Content-Type: multipart/form-data
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| content | string | ✅ | 完整的 docker-compose.yml 文本 |
| projectName | string | ✅ | 项目名（匹配 `^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`） |
| initURL | string | | 附加运行文件 zip 下载地址 |
| initFile | file | | 上传的附加运行文件 zip |

> ⚠️ 此接口使用 `multipart/form-data`（因为支持文件上传），不是 JSON。

返回 `DeployResult`：

```json
{
  "target": "docker",
  "items": ["容器名 (shortId)", "..."],
  "installDir": "/path/to/install"
}
```

### §1.2 获取已部署内容

```
GET /api/compose/docker/:name
```

返回该项目的 docker-compose.yml 内容。

### §1.3 重建部署

```
PUT /api/compose/docker/:name
Body: { "content": "新的 docker-compose.yml 文本" }
```

---

## §2 Swarm Stack 部署（集群）

### §2.1 部署

```
POST /api/compose/swarm/deploy
Content-Type: application/json
Body: {
  "content": "完整的 docker-compose.yml 文本",
  "projectName": "项目名"
}
```

> ⚠️ 与 Docker Compose 不同，Swarm 部署使用 JSON 格式。

### §2.2 获取已部署内容

```
GET /api/compose/swarm/:name
```

### §2.3 重建部署

```
PUT /api/compose/swarm/:name
Body: { "content": "新的 docker-compose.yml 文本" }
```

---

## §3 projectName 命名规则

- 必须匹配正则: `^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`
- 以字母或数字开头
- 只能包含字母、数字、下划线、点、连字符

---

## §4 更新已有部署的镜像

### Docker Compose 方式

```bash
# 1. 获取当前 compose 内容
isrvd_get "/compose/docker/my-app"

# 2. 修改 compose 中的镜像版本（在本地编辑）

# 3. 重建部署
isrvd_put "/compose/docker/my-app" '{"content":"修改后的 compose 内容"}'
```

### Swarm Stack 方式

```bash
# 方式一：直接重新部署（拉取最新镜像）
isrvd_post "/swarm/service/SERVICE_ID/redeploy"

# 方式二：修改 compose 并重建
isrvd_get "/compose/swarm/my-stack"
# 修改镜像版本后...
isrvd_put "/compose/swarm/my-stack" '{"content":"修改后的 compose 内容"}'
```

---

## 常见工作流

### 首次部署多容器应用

```bash
# Docker Compose（单机）
isrvd_upload "/compose/docker/deploy" "initFile" "./init-files.zip" \
  "content=@docker-compose.yml" \
  "projectName=my-app"

# 或者不带附加文件
isrvd_upload "/compose/docker/deploy" "content" "" \
  "content=version: '3.8'
services:
  web:
    image: nginx:latest
    ports:
      - '80:80'
  db:
    image: mysql:8
    environment:
      MYSQL_ROOT_PASSWORD: secret" \
  "projectName=my-app"
```

### 使用 deploy.sh 快捷部署

```bash
# 单机 Compose 部署
./scripts/deploy.sh compose my-app ./docker-compose.yml

# Swarm Stack 部署
./scripts/deploy.sh swarm my-stack ./docker-compose.yml
```
