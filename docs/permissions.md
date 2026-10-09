# 路由权限参考

权限基于路由进行细粒度控制。默认的 `AccessPerm` 路由要求成员持有对应的完整路由权限，Founder 不受此限制；`AccessAuth` 路由只要登录即可访问，`AccessAnon` 路由允许匿名访问。

**权限格式**：`<METHOD> /api/<模块>/<路由>`（如 `GET /api/docker/containers`、`POST /api/compose/docker`）

**前端权限判断**：使用 `portal.hasPerm('<METHOD> /api/<路由>')` 控制按钮/操作的显示

> 留空 = 无 `AccessPerm` 路由权限；具体可用路由可在登录后通过 `GET /api/account/routes` 获取。下表列出主要权限点示例。
>
> 集中管理的节点接口（`/api/node/*`、`/n/<节点ID>/…`）由中控网关提供，不在上述路由表中，**仅创始人可用**，无法授予普通成员。

## 权限点参考

| 模块 | 路由权限点示例 | 说明 |
| ------ | --------------- | ------ |
| `overview` | `GET /api/overview/version` | 系统概览（版本信息） |
| `overview` | `GET /api/overview/monitor` | 系统概览（监控数据） |
| `overview` | `POST /api/overview/upgrade` | 系统概览（在线升级） |
| `system` | `GET /api/system/config` | 系统设置（获取配置） |
| `system` | `PUT /api/system/config` | 系统设置（保存配置） |
| `system` | `GET /api/system/audit/logs` | 系统设置（审计日志） |
| `account` | `GET /api/account/members` | 成员管理（列出） |
| `account` | `POST /api/account/member` | 成员管理（创建） |
| `account` | `PUT /api/account/member/:username` | 成员管理（更新） |
| `account` | `DELETE /api/account/member/:username` | 成员管理（删除） |
| `account` | `POST /api/account/token` | 成员管理（创建 API 令牌） |
| `account` | `GET /api/account/2fa/status` | 二次验证（查询状态） |
| `account` | `POST /api/account/2fa/totp/begin` | 二次验证（开始绑定 TOTP） |
| `account` | `POST /api/account/2fa/totp/enable` | 二次验证（启用 TOTP） |
| `account` | `POST /api/account/2fa/totp/disable` | 二次验证（禁用 TOTP） |
| `account` | `POST /api/account/passkey/register/begin` | Passkey（开始绑定） |
| `account` | `POST /api/account/passkey/register/finish` | Passkey（完成绑定） |
| `account` | `GET /api/account/passkey/credentials` | Passkey（查询凭证列表） |
| `account` | `PUT /api/account/passkey/credential/:id` | Passkey（重命名凭证） |
| `account` | `DELETE /api/account/passkey/credential/:id` | Passkey（删除凭证） |
| `filer` | `GET /api/filer/files` | 文件管理（列出） |
| `filer` | `GET /api/filer/file` | 文件管理（读取） |
| `filer` | `POST /api/filer/file` | 文件管理（创建文件） |
| `filer` | `PUT /api/filer/file` | 文件管理（修改） |
| `filer` | `DELETE /api/filer/file` | 文件管理（删除） |
| `filer` | `POST /api/filer/dir` | 文件管理（创建目录） |
| `filer` | `POST /api/filer/upload` | 文件管理（上传） |
| `filer` | `POST /api/filer/rename` | 文件管理（重命名/移动，后端校验目标路径边界） |
| `filer` | `PUT /api/filer/chmod` | 文件管理（修改权限） |
| `filer` | `POST /api/filer/zip` | 文件管理（压缩） |
| `filer` | `POST /api/filer/unzip` | 文件管理（解压） |
| `shell` | `GET /api/shell` | Web 终端 |
| `local` | `GET /api/local/processes` | 本机进程（列出进程） |
| `local` | `POST /api/local/process/:pid/kill` | 本机进程（终止进程，强制审计） |
| `ssh` | `GET /api/ssh/hosts` | SSH 远程管理（列出主机） |
| `ssh` | `GET /api/ssh/credentials` | SSH 远程管理（列出凭据） |
| `ssh` | `POST /api/ssh/host` | SSH 远程管理（添加主机） |
| `ssh` | `GET /api/ssh/to/:id` | SSH 远程管理（连接终端） |
| `ssh` | `GET /api/sftp/:id/ls` | SSH 远程管理（SFTP 列出目录） |
| `copilot` | `POST /api/copilot/agui` | AI 助手（AG-UI 协议对话） |
| `cron` | `GET /api/cron/jobs` | 计划任务（列出） |
| `cron` | `POST /api/cron/jobs` | 计划任务（创建） |
| `cron` | `POST /api/cron/jobs/:id/run` | 计划任务（立即执行） |
| `cron` | `GET /api/cron/jobs/:id/logs` | 计划任务（查看日志） |
| `apisix` | `GET /api/apisix/routes` | APISIX 管理（路由列出） |
| `apisix` | `GET /api/apisix/consumers` | APISIX 管理（消费者列出） |
| `apisix` | `GET /api/apisix/upstreams` | APISIX 管理（上游列出） |
| `apisix` | `GET /api/apisix/ssls` | APISIX 管理（证书列出） |
| `apisix` | `GET /api/apisix/plugin-configs` | APISIX 管理（插件配置列出） |
| `apisix` | `GET /api/apisix/whitelist` | APISIX 管理（访问授权列出） |
| `docker` | `GET /api/docker/info` | Docker 管理（服务信息） |
| `docker` | `GET /api/docker/containers` | Docker 管理（容器列出） |
| `docker` | `GET /api/docker/images` | Docker 管理（镜像列出） |
| `docker` | `GET /api/docker/networks` | Docker 管理（网络列出） |
| `docker` | `GET /api/docker/volumes` | Docker 管理（卷列出） |
| `docker` | `GET /api/docker/registries` | Docker 管理（镜像仓库列出） |
| `swarm` | `GET /api/swarm/info` | Swarm 管理（集群信息） |
| `swarm` | `GET /api/swarm/nodes` | Swarm 管理（节点列出） |
| `swarm` | `GET /api/swarm/services` | Swarm 管理（服务列出） |
| `swarm` | `GET /api/swarm/tasks` | Swarm 管理（任务列出） |
| `compose` | `GET /api/compose/docker/:name` | Compose 管理（读取配置） |
| `compose` | `POST /api/compose/docker` | Compose 管理（部署） |
| `compose` | `PUT /api/compose/docker/:name` | Compose 管理（重部署） |
| `compose` | `GET /api/compose/swarm/:name` | Compose 管理（读取配置） |
| `compose` | `POST /api/compose/swarm` | Compose 管理（部署） |
| `compose` | `PUT /api/compose/swarm/:name` | Compose 管理（重部署） |
| `caddy` | `GET /api/caddy/info` | Caddy 管理（概览） |
| `caddy` | `GET /api/caddy/config` | Caddy 管理（读取原始配置） |
| `caddy` | `POST /api/caddy/config` | Caddy 管理（整体替换配置） |
| `caddy` | `GET /api/caddy/global` | Caddy 管理（读取全局选项） |
| `caddy` | `PUT /api/caddy/global` | Caddy 管理（更新全局选项） |
| `caddy` | `GET /api/caddy/servers` | Caddy 管理（HTTP 服务列出） |
| `caddy` | `GET /api/caddy/routes` | Caddy 管理（路由列出） |
| `caddy` | `GET /api/caddy/basic-auth` | Caddy 管理（Basic Auth 路由列出） |
| `caddy` | `GET /api/caddy/certs` | Caddy 管理（证书列出） |
| `caddy` | `POST /api/caddy/cert` | Caddy 管理（创建证书） |
| `caddy` | `PUT /api/caddy/cert/:key` | Caddy 管理（更新证书） |
| `caddy` | `DELETE /api/caddy/cert/:key` | Caddy 管理（删除证书） |
