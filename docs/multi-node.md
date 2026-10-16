# 集中管理（多服务器）

同一个 `isrvd` 二进制，通过环境变量 `ISRVD_MODE` 选择角色。部署参数不存入 YAML 或 etcd，也不由后台配置 API 管理；修改环境变量后重启生效：

| 模式 | 作用 |
|------|------|
| `server`（默认） | 单机 |
| `center` | 中控：提供 Web 界面、账号与权限，接收受管机接入，并把请求按节点转发 |
| `agent` | 受管机：不开放任何对外端口，主动连接中控，由中控转发来的请求在本机执行 |

```text
浏览器 ──> center ──┬── 页面、账号、权限、审计 ──> 本进程内的 isrvd
                    └── /n/<节点ID>/api/… ──隧道──> agent（本进程内的 isrvd）
                                                  ▲ 受管机主动出站，中控从不主动外连
```

中控设置 `ISRVD_MODE=center`；受控机启动示例：

```bash
export ISRVD_MODE=agent
export ISRVD_CENTER_URL=https://center.example.com
export ISRVD_NODE_TOKEN='<中控生成的 token>'
isrvd
```

旧 YAML 的 `cluster` 分区不再读取，后续保存配置时会移除。升级前请将部署参数迁移到进程环境；未设置模式时按单机启动。各节点可以共享 etcd 配置，部署角色与令牌由各自环境独立提供。

## 工作方式

- 每个进程内部仍是完整的 isrvd：`center` 与 `agent` 模式下，isrvd 只监听 `127.0.0.1` 的随机端口。`center` 另起网关占用配置里的 `listenAddr` 作为对外入口；`agent` 不监听任何对外端口，配置里的 `listenAddr` 不生效。
- 浏览器访问 `/n/<节点ID>/`，页面仍由中控提供；其中的 Docker、Swarm、Compose、文件、本机进程、Shell 终端、计划任务、APISIX、Caddy、系统概览接口经隧道转发到该节点，包括流式日志、终端和文件上传下载。
- 账号、系统配置、AI 助手、SSH 远程管理始终由中控自己处理，即使在 `/n/<节点ID>/` 下访问也一样。
- 节点切换与节点管理是 webview 的一部分，**按需启用**：只有 `center` 模式（`/api/overview/bootstrap` 的 `probe.node` 为 `true`）且当前用户是创始人时，才会出现头部的节点切换器；侧边栏「节点管理」还要求当前处于本机（中控）视角（位于「用户管理」下方，一个页面）。单机模式、`agent` 模式和非创始人看不到任何节点界面，也不会发出节点接口请求，直接访问 `#/node` 会回到概览。
- 切换节点是一次整页跳转：地址变为 `/n/<节点ID>/`，页面停留在原来的功能页（带资源参数的详情页回到概览，从中控管理页面切换到子节点时也回到概览）。子节点视角隐藏「节点管理」「用户管理」「系统配置」「操作审计」与「SSH 远程管理」，直接访问这些页面会回到概览；头部节点切换器与个人账户设置保留。此限制仅影响前端页面，相关接口仍按原规则作用于中控，并保持原有认证与权限校验。
- 认证与权限复用中控当前的账户服务：网关在进程内验证调用者凭据并检查创始人身份。成员权限、密码或 JWT 密钥变更在配置发布后对新请求立即生效；已建立的长连接不会逐条消息重新认证。应用未就绪或正在重载时，节点管理与转发返回 `502`，避免前端误判为登录过期。节点视角的页面启动仍聚合中控身份与节点服务能力，中控上游不可达时同样返回 `502`。
- 中控网关在请求带 `Upgrade: websocket` 或命中声明了 `QueryToken` 的 `GET` 路由时可从 `?token=` 读取令牌；无论凭据来自哪里，节点管理和转发仍须通过中控创始人身份校验。普通请求不要把令牌放进 URL，避免进入访问日志、浏览器历史记录或 Referer。

## 部署

**中控**：设置环境变量 `ISRVD_MODE=center`，沿用原来的 `listenAddr`，然后重启。

**受控机**：先在中控「接入节点」生成 token，将弹窗中的环境变量配置到受控机启动环境，然后重启。切换后只能经中控访问；地址填错时需修改受控机环境变量并重启。

token 在激活前有有效期，首次连接时激活为节点；之后沿用相同 token，直到吊销或删除。**同一 token 只允许一个在线或正在握手的客户端**；重复连接返回 `409`，不会顶替已有会话，断线后可用同一 token 重连。

**Docker**：提前准备完整的 `/srv/data/conf/isrvd.yml`，通过 `-e` 传入部署参数。中控和受控机应使用各自的数据目录：

```bash
# 中控（8080 为对外入口）
docker run -d --name isrvd-center -e ISRVD_MODE=center --network sdnet -p 8080:8080 \
  -v /srv/data:/data -v /var/run/docker.sock:/var/run/docker.sock rehiy/isrvd:slim

# 受控机（不需要 -p）
docker run -d --name isrvd-agent --network sdnet \
  -e ISRVD_MODE=agent -e ISRVD_CENTER_URL=https://center.example.com \
  -e ISRVD_NODE_TOKEN="<中控生成的 token>" \
  -v /srv/data:/data -v /var/run/docker.sock:/var/run/docker.sock rehiy/isrvd:slim
```

安装脚本的 Docker `update` 会保留现有容器的 `ISRVD_MODE`、`ISRVD_CENTER_URL`、`ISRVD_NODE_TOKEN`，更新失败回滚时同样保留。

**systemd**：通过 `systemctl edit isrvd` 在 `[Service]` 中设置 `Environment="ISRVD_MODE=center"`；受控机还需设置 `ISRVD_CENTER_URL`、`ISRVD_NODE_TOKEN`。执行 `systemctl daemon-reload` 与 `systemctl restart isrvd`。安装默认以单机模式启动。

| 环境变量 | 适用 | 说明 |
|----------|------|------|
| `ISRVD_MODE` | 全部 | `server`（默认）/ `center` / `agent` |
| `ISRVD_CENTER_URL` | agent，必填 | HTTP(S) 中控地址，不含路径、查询参数、片段或账号密码 |
| `ISRVD_NODE_TOKEN` | agent，必填 | 中控直接生成的 token；仅从进程环境读取，不通过配置 API 返回或修改 |

`ISRVD_CONFIG` 用于指定配置文件或 etcd 位置。保存 token 的环境文件应限制权限为 `0600`，不要提交或公开。

接入流程：在「节点管理」页点击「接入节点」生成令牌（列表中显示为「待接入」）→ 受管机携带令牌启动 → 首次连接时中控校验并激活该令牌，节点建立常连，状态变为「在线」。节点与待接入的令牌在同一个列表里，用状态区分。

**受管机被中控拒绝时**（节点被吊销、删除、令牌已过期未激活，或中控数据被重置、恢复了旧备份）受管机每分钟重试并记录错误日志。中控数据恢复后自动重连；确实要重新接入时，生成新的令牌并重启受管机。

数据位置：中控的节点表与注册码（`node-list.yml`、`node-code.yml`）与计划任务、SSH 主机等业务数据一样，保存在 `rootDirectory` 下，使用 etcd 时随配置一起同步。受控机的 token 由环境变量 `ISRVD_NODE_TOKEN` 提供，不写入主配置。

## 本地调试

使用 `ISRVD_MODE=center ./develop.sh` 调试节点管理；`.local.yml` 仅保存业务配置。前端开发服务器（`3000`）同时代理 `/api/`、`/openapi/` 和节点视角的 `/n/`。

要在本机调试一个受管机，另开终端用另一份配置（不同的 `rootDirectory`）启动：

```bash
ISRVD_MODE=agent ISRVD_CENTER_URL=https://center.example.com ISRVD_NODE_TOKEN="<token>" ISRVD_CONFIG=./agent.yml go run ./server/cmd/server
```

`/n/<节点ID>/` 下的页面由后端内嵌的前端产物提供，修改前端后需执行 `npm run build`，才能在节点视角下看到变化。

## 权限与边界

- **只有创始人可以管理和操作节点。** 节点上的请求以受管机本地的创始人身份执行，与调用者无关，所以不能把节点权限下放给只有部分路由权限的成员。
- 受管机用**用户名排序后的第一个创始人**签发访问令牌，文件管理与终端使用该成员的家目录。受管机上必须至少有一个创始人，且不能启用代理 Header 登录（`tha.enabled`），否则启动失败。
- 中控模式允许代理 Header 登录（`center + tha.enabled`）。网关按连接对端 IP 和 `tha.trustedCIDRs` 校验身份头，不采信转发头；可信代理必须覆盖客户端传入的身份头。THA 与 JWT 均使用当前配置判断权限，普通成员仍不能管理节点。中控的私有回环服务只接收网关写入的内部身份，外部同名头会先被删除；THA 头名重载不会重新信任原始请求头。本机进程属于可信边界，用户身份不下发给受控节点。
- 受管机转发 WebSocket 时没有 `Origin`，`agent` 模式会放行空 `Origin`；跨站来源已由中控网关按 `allowedOrigins` 校验（未配置时仅允许同源）。
- 节点令牌只保存哈希；注册码一次性使用、默认 1 小时过期（最长 7 天），用 `server.jwtSecret` 派生的密钥加密保存在 `node-code.yml` 中，创始人在其待接入期间可随时从列表查看接入命令（修改 `jwtSecret` 后未使用的注册码会被清除）；首次连接成功即激活为节点并核销待接入记录，之后令牌可按节点吊销。
- 网关把请求转发给节点前会剥离 `Authorization`、`Cookie`、`token` 查询参数。
- 节点管理操作（生成注册码、吊销、删除等）、节点上的写操作与 WebSocket 都会记入中控的审计页面，规则与 isrvd 一致：非 GET 与全部 WebSocket 记录，WebSocket 在会话结束时记录；操作人是中控登录用户，URI 带 `/n/<节点ID>/` 前缀，请求体按 isrvd 的规则脱敏。受管机本机的审计页面里同一操作记录为其本地创始人。
- 网关审计和节点 token 认证失败限流统一使用连接对端 IP，不采信客户端的 `X-Forwarded-For` 或 `X-Real-IP`；向中控内部服务转发时重新生成来源 IP 头，不透传外部来源链。部署在反向代理后，审计记录代理 IP，同一代理后的客户端共享节点认证失败限流额度。
- AI 助手在节点页面下仍作用于中控本机。
- 配置重载（SIGHUP、保存系统配置或 etcd 变更）后，`allowedOrigins` 与 `rootDirectory` 的变化对中控即时生效（节点表与注册码随 `rootDirectory` 切换到新位置，与计划任务等业务数据一致）；部署环境变量与 `listenAddr` 仍需重启。
- 中控为单实例设计；重启时所有节点会短暂断开，受管机按 1 秒起步的指数退避加抖动自动重连。
- 受管机与中控的版本需一致；协议版本不一致时节点显示「版本不兼容」并拒绝转发。

## 网关接口

以下接口由 `center` 模式的网关提供（不在 isrvd 的 `/api/*` 路由表中，也不出现在 OpenAPI 目录里）。响应格式与 isrvd 一致：`{"success":bool,"message":string,"payload":any}`。需要创始人的接口使用 `Authorization` 头认证。详细的请求与响应字段见 [受管节点](references/node/nodes.md) 与 [注册码](references/node/codes.md)。

| 方法与路径 | 认证 | 说明 |
|------------|------|------|
| `GET /api/node/nodes` | 创始人 | 节点列表，含 `status`（`active`/`revoked`）、`online`、`latency`、`agentVersion`、`compatible` 等 |
| `PUT /api/node/item/{id}` | 创始人 | 重命名，body `{"name":"…"}`（1–64 个字符） |
| `POST /api/node/item/{id}/revoke` | 创始人 | 吊销：令牌立即失效并断开连接，需删除后用新令牌重新接入 |
| `DELETE /api/node/item/{id}` | 创始人 | 删除并断开连接 |
| `GET /api/node/codes` | 创始人 | 尚未使用且未过期的注册码（含明文，用于查看接入命令） |
| `POST /api/node/code` | 创始人 | 生成注册码，body `{"name":"","ttlMinutes":60}`；响应与 `GET /api/node/codes` 都包含 `code` 明文 |
| `DELETE /api/node/code/{id}` | 创始人 | 撤销注册码 |
| `GET /api/node/connect` | 节点令牌 | 受管机建立 WebSocket 隧道，首次连接时激活待接入的令牌；令牌只经 `Authorization: Bearer` 头传递，带 `Origin` 的浏览器请求会被拒绝 |
| `/n/{id}/…` | 创始人 | 以某个节点的视角访问，见上文「工作方式」 |
