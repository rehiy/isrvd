# Compose API

Compose 接口用于单机 Docker Compose 与 Swarm Stack 的部署、读取配置、重部署，以及按服务更新镜像并重建。Docker Compose 读取与重部署支持按 `com.docker.compose.project` 标签聚合外部 `docker compose up` 启动的既有项目。

## 字段说明

### ComposeDeploy

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `content` | string | 是 | 完整 compose yaml 文本 |
| `envContent` | string | 否 | `.env` 内容（`KEY=VALUE`），部署时合并进变量插值环境并写盘 |
| `initURL` | string | 否 | 附加运行文件 zip 下载地址 |
| `initFile` | file | 否 | 附加运行文件 zip；与 `initURL` 互斥且文件优先 |
| `forcePull` | boolean | 否 | `true` 时强制拉取最新镜像（即使本地已存在），默认 `false` |

> Docker 与 Swarm Compose 部署都支持 `initURL` / `initFile`；解压目标为 `docker.containerRoot/<NAME>/`。Swarm 场景下建议 `containerRoot` 指向各节点共享的存储（如 NFS），以便所有节点都能访问解压出的文件。

部署时先浅解析顶层 `name`：支持使用进程环境和请求中 `envContent` 做项目名插值，并按 compose-go 规则转为小写规范名；未声明 `name` 时使用 compose 内容短哈希。随后在任何落盘前做不读取 `env_file` 的结构预检，只在 `.env` 和附加运行文件就位后执行完整 compose 加载。

### ComposeRedeploy

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `content` | string | 按需 | 完整 compose yaml 文本（全量重建） |
| `envContent` | string | 按需 | `.env` 内容（`KEY=VALUE`）；可与 `content` 一起提交，也可单独提交仅更新 `.env` 后重建 |
| `serviceName` | string | 按需 | 要更新镜像的 compose 服务名（按服务更新） |
| `image` | string | 按需 | 新镜像名，`serviceName` 非空时必填 |
| `forcePull` | boolean | 否 | `true` 时强制拉取最新镜像，默认 `false` |

> `content`、`envContent`、`serviceName` 三者至少提交一项；`serviceName` 与 `content` 互斥，`serviceName` 非空时 `image` 必填。

重部署会在正式加载和创建容器/服务前写入新 `.env`，确保 `env_file` 和变量插值使用新值；后续失败时会精确恢复原 `.env` 内容，包括空文件或原本不存在的状态，再回滚旧实例。

### ComposeDeployResult

| 字段 | 类型 | 说明 |
|---|---|---|
| `projectName` | string | 实际使用的项目名 |
| `items` | string[] | 创建或重建的容器/服务列表 |
| `installDir` | string | 项目落盘目录（Docker Compose 与 Swarm 均返回） |

### DockerComposeContentResult

| 字段 | 类型 | 说明 |
|---|---|---|
| `content` | string | compose.yml 文本 |
| `envContent` | string | `.env` 文本（`KEY=VALUE`）；无落盘文件时不返回 |
| `projectName` | string | 实际解析到的项目名 |
| `fileModTime` | number | `docker.containerRoot/<PROJECT>/compose.yml` 修改时间（Unix 秒）；无落盘文件时不返回 |
| `source` | string | 内容来源：`file` 表示落盘文件，`runtime` 表示从运行态反推 |

## Docker Compose

### 部署

Docker Compose 部署支持 JSON；上传附加运行文件时使用 multipart form。

仅提交 compose 内容：

```bash
isrvd_post "/compose/docker" "$(jq -n --arg content "$(cat docker-compose.yml)" '{content:$content}')"
```

上传本地附加文件：

```bash
isrvd_upload "/compose/docker" "initFile" "./init.zip" "content=$(cat docker-compose.yml)"
```

使用远程附加文件：

```bash
isrvd_post "/compose/docker" '{"content":"<COMPOSE_YAML>","initURL":"<HTTPS_ZIP_URL>"}'
```

> `initURL` 仅允许 `http/https` 公网地址，不允许指向本机、内网或链路本地地址。

### 读取 compose 文件

```bash
isrvd_get "/compose/docker/<NAME>"
```

`<NAME>` 可以是 iSrvd 项目名，也可以是带 `com.docker.compose.project` 标签的容器名。读取顺序：

1. 优先读取 `docker.containerRoot/<PROJECT>/compose.yml`，响应中的 `fileModTime` 为该文件修改时间
2. 文件不存在时，按 `com.docker.compose.project=<PROJECT>` 聚合同项目容器并反推多服务 compose
3. 无 compose project 标签时，退回单容器 inspect 反推

强制从运行态反推（跳过已有 `compose.yml`，但仍返回已有文件的 `fileModTime` 供前端提示）：

```bash
isrvd_get "/compose/docker/<NAME>?force=true"
```

```bash
# 读取 iSrvd 已落盘的项目，或读取外部 Compose 项目标签聚合后的配置
isrvd_get "/compose/docker/<PROJECT>"

# 也可以传入该 Compose 项目下任意一个容器名，后端会解析到真实 project
isrvd_get "/compose/docker/<CONTAINER_NAME>"
```

Docker Compose 中的相对 bind path 会基于容器目录 `docker.containerRoot/<PROJECT>` 解析，部署、全量重部署、按服务更新镜像和失败回滚保持一致。后端统一使用 compose-go loader 解析与标准化 YAML，再直接生成 Docker SDK `container.Config` / `container.HostConfig` 或 `swarm.ServiceSpec` 原始结构；`entrypoint`、`command`、`dns`、`dns_opt`、`dns_search`、`extra_hosts`、`tty`、`stdin_open`、`read_only`、`stop_signal`、`sysctls`、`device_cgroup_rules`、`devices`、`security_opt`、`group_add`、`ipc`、`pid`、`uts`、`cgroup`、`runtime`、`shm_size`、`tmpfs`、`ulimits`、`cap_add`、`cap_drop`、`deploy.resources` 等字段不再经过 iSrvd 自定义中间 DTO。

Docker Compose 部署会将顶层 `gpus` 以及 `deploy.resources.reservations.devices` 映射为 Docker `HostConfig.DeviceRequests`，并保留 `options`。GPU 场景可使用：

```yaml
services:
  worker:
    image: nvidia/cuda:12.4.1-base-ubuntu22.04
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: all
              capabilities: [gpu]
```

当 `compose.yml` 不存在、需要从运行态反推生成时，bind mount 的 `source` 会按该容器目录输出为相对路径（例如 `./data`）；只有路径不在容器目录内时才保留宿主机绝对路径。命名卷输出卷名，不输出 Docker 内部挂载目录，`--device` 会反推到 Compose `devices`，GPU 设备请求会反推到 `deploy.resources.reservations.devices` 并保留 `options`。反推会尽量与镜像 Dockerfile 默认配置做差分，避免把镜像内已有的 `CMD`、`ENTRYPOINT`、`ENV`、`EXPOSE`、`LABEL`、`STOPSIGNAL`、`USER`、`WORKDIR` 重复写入 compose。`ports` 未指定 `published` 时会保持 Docker 随机宿主端口发布语义，不会降级为仅 `expose`。

外部 Compose 项目接管说明：

- 同一 `com.docker.compose.project` 下的多个容器会合并为一个多服务 compose
- 反推内容会保留真实容器名为 `container_name`，便于后续重部署删除旧容器并保持名称稳定
- 重部署成功后会将 compose 写入 `docker.containerRoot/<PROJECT>/compose.yml`
- 暂不支持接管同一 `com.docker.compose.service` 下存在多个容器的 scaled 服务

### 重部署

```bash
isrvd_put "/compose/docker/<NAME>" "$(jq -n --arg content "$(cat docker-compose.yml)" '{content:$content}')"
```

`<NAME>` 可以是项目名，也可以是该项目下任意一个容器名；后端会通过 `com.docker.compose.project` 解析到项目名后整体重建关联容器。

重部署时同时更新 `.env`：

```bash
isrvd_put "/compose/docker/<NAME>" "$(jq -n --arg content "$(cat docker-compose.yml)" --arg envContent "$(cat .env)" '{content:$content,envContent:$envContent}')"
```

仅更新 `.env` 后重建（`content` 省略，沿用现有 compose.yml）：

```bash
isrvd_put "/compose/docker/<NAME>" "$(jq -n --arg envContent "$(cat .env)" '{envContent:$envContent}')"
```

### 按服务更新镜像并重建

```bash
isrvd_put "/compose/docker/<NAME>" '{"serviceName":"<SERVICE_NAME>","image":"<NEW_IMAGE>"}'
```

## Swarm Compose

### 部署

Swarm Compose 部署用法与 Docker Compose 一致，同样支持 JSON 与 multipart form：

```bash
isrvd_post "/compose/swarm" "$(jq -n --arg content "$(cat stack.yml)" '{content:$content}')"
```

上传本地附加文件：

```bash
isrvd_upload "/compose/swarm" "initFile" "./init.zip" "content=$(cat stack.yml)"
```

使用远程附加文件：

```bash
isrvd_post "/compose/swarm" '{"content":"<COMPOSE_YAML>","initURL":"<HTTPS_ZIP_URL>"}'
```

### 读取 compose 文件

```bash
isrvd_get "/compose/swarm/<NAME>"
```

Swarm Compose 与 Docker Compose 共用容器目录 `docker.containerRoot/<NAME>`：相对 bind path 基于该目录解析；从运行态反推时也按此目录输出相对路径。注意 Swarm 是分布式部署，落盘目录仅在主节点；非主节点上的相对 bind path 需各节点自行准备。Swarm Stack 暂不映射 Docker 单机 `--gpus` / `HostConfig.DeviceRequests` 语义；GPU 调度需通过 Docker Swarm generic resources 等集群级配置另行管理。

### 重部署

```bash
isrvd_put "/compose/swarm/<NAME>" "$(jq -n --arg content "$(cat stack.yml)" '{content:$content}')"
```

重部署时同时更新 `.env`：

```bash
isrvd_put "/compose/swarm/<NAME>" "$(jq -n --arg content "$(cat stack.yml)" --arg envContent "$(cat .env)" '{content:$content,envContent:$envContent}')"
```

仅更新 `.env` 后重建（`content` 省略，沿用现有 compose.yml）：

```bash
isrvd_put "/compose/swarm/<NAME>" "$(jq -n --arg envContent "$(cat .env)" '{envContent:$envContent}')"
```

### 按服务更新镜像并重建

```bash
isrvd_put "/compose/swarm/<NAME>" '{"serviceName":"<SERVICE_NAME>","image":"<NEW_IMAGE>"}'
```
