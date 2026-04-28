---
name: isrvd-ops
description: 通过 isrvd API 进行容器部署、服务管理、镜像操作、路由配置等运维操作。当用户要求"部署服务"、"管理容器"、"拉取/推送镜像"、"配置路由"、"管理 Swarm"、"Compose 部署"等运维任务时使用此 Skill。
---

# isrvd 运维操作 Skill

isrvd 是一个集成了 Docker、Swarm、APISIX、Compose 和文件管理的运维平台。

> **渐进式披露**：本文件只包含概述和决策树。各模块的详细 API 文档在 `docs/` 子目录中，按需读取。
> **Script Harness**：`scripts/` 目录提供可直接执行的 shell 脚本，封装了认证、调用和常见工作流。

---

## 快速开始

### 1. 环境准备

```bash
# 检测 isrvd 服务状态并获取认证 token
source ./scripts/api.sh
isrvd_init "http://localhost:8080" "admin" "password"
```

### 2. 常用快捷脚本

| 脚本 | 用途 | 示例 |
|------|------|------|
| `scripts/api.sh` | 通用 API 封装（认证、GET/POST/PUT/DELETE） | `source scripts/api.sh && isrvd_get "/docker/containers"` |
| `scripts/deploy.sh` | 一键部署（Compose/容器/Swarm 服务） | `./scripts/deploy.sh compose my-app ./docker-compose.yml` |
| `scripts/health-check.sh` | 健康检查与状态报告 | `./scripts/health-check.sh` |

---

## 决策树：用户想做什么？

```
用户需求
├── 部署/创建
│   ├── 单个容器 → 读 docs/docker.md §2.2 → 用 scripts/deploy.sh container
│   ├── 多容器应用(单机) → 读 docs/compose.md §1 → 用 scripts/deploy.sh compose
│   ├── 集群服务 → 读 docs/swarm.md §3 或 docs/compose.md §2
│   └── 配置路由 → 读 docs/apisix.md §1-§3
│
├── 更新/变更
│   ├── 更新容器镜像 → docs/docker.md §3.7 拉取 + docs/compose.md §4 重建
│   ├── 扩缩容 Swarm 服务 → docs/swarm.md §4.1
│   ├── 重新部署 Swarm 服务 → docs/swarm.md §4.2
│   ├── 修改路由规则 → docs/apisix.md §4
│   └── 修改系统配置 → docs/system.md §2
│
├── 查询/监控
│   ├── 查看容器/镜像/网络/卷 → docs/docker.md §1-§6
│   ├── 查看集群/服务/任务 → docs/swarm.md §1-§2
│   ├── 查看路由/上游/插件 → docs/apisix.md §1
│   ├── 系统状态/健康检查 → docs/system.md §1 → 用 scripts/health-check.sh
│   └── 查看日志 → docs/docker.md §2.4 或 docs/swarm.md §4.3
│
├── 删除/清理
│   ├── 删除容器/镜像/网络/卷 → docs/docker.md 各模块 action=remove
│   ├── 删除 Swarm 服务 → docs/swarm.md §4.1
│   └── 删除路由/消费者 → docs/apisix.md §5-§6
│
└── 管理
    ├── 镜像仓库管理 → docs/docker.md §4
    ├── 成员/权限管理 → docs/system.md §3
    └── 文件管理 → docs/system.md §4
```

---

## 通用约定（所有模块共享）

### 认证

- **JWT**：`Authorization: Bearer <token>`（通过 `POST /api/auth/login` 获取）
- **代理 Header**：反向代理注入用户名（配置项 `proxyHeaderName`）

### 统一响应格式

```json
{ "success": true|false, "message": "描述", "payload": <数据或null> }
```

### 权限模块

| 模块 | 说明 | 权限级别 |
|------|------|----------|
| `docker` | 容器/镜像/网络/卷/仓库 | `r` / `rw` |
| `swarm` | 集群/服务/节点/任务 | `r` / `rw` |
| `compose` | Compose 部署 | `r` / `rw` |
| `apisix` | 路由/消费者/上游/插件 | `r` / `rw` |
| `filer` | 文件管理 | `r` / `rw` |
| `system` | 系统配置/成员 | `r` / `rw` |
| `agent` | LLM 代理 | `r` / `rw` |

---

## 详细文档索引

| 文档 | 覆盖模块 | 何时读取 |
|------|----------|----------|
| [docs/docker.md](docs/docker.md) | 容器、镜像、网络、数据卷、镜像仓库 | 需要管理 Docker 资源时 |
| [docs/swarm.md](docs/swarm.md) | Swarm 集群、节点、服务、任务 | 需要管理集群服务时 |
| [docs/compose.md](docs/compose.md) | Docker Compose、Swarm Stack 部署 | 需要通过 compose 文件部署时 |
| [docs/apisix.md](docs/apisix.md) | 路由、消费者、上游、插件、白名单 | 需要配置 API 网关时 |
| [docs/system.md](docs/system.md) | 系统状态、配置、成员、文件管理 | 需要系统管理操作时 |

## 脚本索引

| 脚本 | 用途 | 何时使用 |
|------|------|----------|
| [scripts/api.sh](scripts/api.sh) | 通用 API 调用封装 | 所有 API 调用的基础，其他脚本依赖此文件 |
| [scripts/deploy.sh](scripts/deploy.sh) | 部署快捷脚本 | 需要快速部署容器/Compose/Swarm 服务时 |
| [scripts/health-check.sh](scripts/health-check.sh) | 健康检查与状态报告 | 需要检查系统和服务状态时 |
