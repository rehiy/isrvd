# 集中管理（多服务器）

同一个 `isrvd` 二进制，通过 `--mode` 选择角色，不需要额外的程序：

| 模式 | 作用 |
|------|------|
| `server`（默认） | 单机，与以往完全一致 |
| `center` | 中控：提供 Web 界面、账号与权限，接收受管机接入，并把请求按节点转发 |
| `agent` | 受管机：不开放任何对外端口，主动连接中控，由中控转发来的请求在本机执行 |

```text
浏览器 ──> center ──┬── 页面、账号、权限、审计 ──> 本进程内的 isrvd
                    └── /n/<节点ID>/api/… ──隧道──> agent（本进程内的 isrvd）
                                                  ▲ 受管机主动出站，中控从不主动外连
```

```bash
isrvd --mode center
isrvd --mode agent --center-url https://center.example.com --enroll-code <注册码>
```

## 工作方式

- 每个进程内部仍是完整的 isrvd：`center` 与 `agent` 模式下，isrvd 只监听 `127.0.0.1` 的随机端口。`center` 另起网关占用配置里的 `listenAddr` 作为对外入口；`agent` 不监听任何对外端口，配置里的 `listenAddr` 不生效。
- 浏览器访问 `/n/<节点ID>/`，页面仍由中控提供；其中的 Docker、Swarm、Compose、文件、本机进程、Shell 终端、计划任务、APISIX、Caddy、系统概览接口经隧道转发到该节点，包括流式日志、终端和文件上传下载。
- 账号、系统配置、AI 助手、SSH 远程管理始终由中控自己处理，即使在 `/n/<节点ID>/` 下访问也一样。
- 节点切换与节点管理是 webview 的一部分，**按需启用**：只有 `center` 模式（`/api/overview/bootstrap` 的 `probe.node` 为 `true`）且当前用户是创始人时，才会出现头部的节点切换器和侧边栏「节点管理」（位于「用户管理」下方，一个页面）。单机模式、`agent` 模式和非创始人看不到任何节点界面，也不会发出节点接口请求，直接访问 `#/node` 会回到概览。网关不再注入脚本或托管页面。
- 切换节点是一次整页跳转：地址变为 `/n/<节点ID>/`，页面停留在原来的功能页（带资源参数的详情页回到概览）。在节点视角下，「节点管理」页面和接口仍作用于中控本身。
- 认证与权限全部委托给中控 isrvd：网关用调用者的凭据请求 `/api/overview/bootstrap` 判断身份。结果缓存 10 秒，因此成员被降权或移除后最多 10 秒生效。中控 isrvd 暂不可用时网关返回 `502`，而不是 `401`，避免前端把一次上游抖动当成登录过期而登出。
- `?token=` 只在与 isrvd 相同的范围内有效：WebSocket，以及声明了 `QueryToken` 的 `GET` 路由（文件下载、实时日志）。其余请求，包括所有写操作与节点管理接口，必须用 `Authorization` 头，否则返回 `401`。

## 部署

**中控**：无需额外配置，加上 `--mode center` 启动即可，沿用原来的 `config.yml` 与 `listenAddr`。

**受管机**：先在中控界面生成注册码，再启动（生成后会直接给出完整命令，可一键复制）：

```bash
isrvd --mode agent --center-url https://center.example.com --enroll-code <注册码>
```

首次注册成功后，节点令牌会加密保存到本机 `<rootDirectory>/node-agent.yml`，之后重启可省略 `--enroll-code`；注册码一直留在启动参数里也没有问题，本地已有可用凭据时不会再使用它。

参数会在读取配置之前校验：参数错误或 `--help` 不会读取、更不会改写配置文件。

**Docker**：镜像入口不带参数，用环境变量选择模式即可，不需要改镜像或配置文件：

```bash
# 中控（8080 为对外入口）
docker run -d --name isrvd-center --network sdnet -p 8080:8080 -e ISRVD_MODE=center \
  -v /srv/data:/data -v /var/run/docker.sock:/var/run/docker.sock rehiy/isrvd:slim

# 受管机（不需要 -p）
docker run -d --name isrvd-agent --network sdnet \
  -e ISRVD_MODE=agent -e ISRVD_CENTER_URL=https://center.example.com -e ISRVD_ENROLL_CODE=<注册码> \
  -v /srv/data:/data -v /var/run/docker.sock:/var/run/docker.sock rehiy/isrvd:slim
```

**systemd**（安装脚本生成的单元文件不带模式）：用 `systemctl edit isrvd` 添加 drop-in，写入 `[Service]` 与对应的 `Environment="ISRVD_MODE=agent"` 等变量，再 `systemctl restart isrvd`。

**默认配置文件无需改动**：模式只由命令行或环境变量决定，不在 `config.yml` 中；`center` 沿用 `listenAddr` 作为对外入口，`agent` 用 `jwtSecret` 加密本地凭据。

| 参数 | 环境变量 | 适用 | 说明 |
|------|----------|------|------|
| `--mode` | `ISRVD_MODE` | 全部 | `server`（默认）/ `center` / `agent` |
| `--center-url` | `ISRVD_CENTER_URL` | agent，必填 | 中控地址，不含 `/api` |
| `--enroll-code` | `ISRVD_ENROLL_CODE` | agent | 首次注册的注册码；本地已有凭据后可省略 |
| `--node-name` | `ISRVD_NODE_NAME` | agent | 节点名称，默认取主机名 |
| | `ISRVD_CENTER_TRUST_PROXY` | center | 设为 `1` 时按 `X-Forwarded-For` 识别来源 IP；仅在网关前还有可信反向代理时设置 |

接入流程：在「节点管理」页点击「接入节点」生成注册码（列表中显示为「待接入」）→ 受管机携带注册码启动 → 该行变为节点，状态为「待审批」→ 审批 → 节点领取令牌并建立常连，状态变为「在线」。节点与待接入的注册码在同一个列表里，用状态区分。勾选「自动通过审批」生成的注册码可跳过审批步骤。

**受管机被中控拒绝时**（节点被吊销、删除，或中控数据被重置、恢复了旧备份）不会删除本地凭据：
- 启动时给了尚未用过的注册码：用它重新注册，成功后才替换本地凭据；
- 否则保留凭据，每分钟重试并记录错误日志。中控数据恢复后自动重连；确实要重新接入时，用新的注册码重启受管机。

数据位置：中控的节点表与注册码（`node-list.yml`、`node-code.yml`）与计划任务、SSH 主机等业务数据一样，保存在 `rootDirectory` 下，使用 etcd 时随配置一起同步。受管机的节点令牌保存在本机 `<rootDirectory>/node-agent.yml`，权限 `0600`。

## 本地调试

仓库的 `./develop.sh` 默认以 `center` 模式启动后端（`8080` 为网关入口），前端开发服务器（`3000`）同时代理 `/api/`、`/openapi/` 和节点视角的 `/n/`。需要调试单机模式时用 `ISRVD_MODE=server ./develop.sh`。

要在本机调试一个受管机，另开终端用另一份配置（不同的 `rootDirectory`）启动：

```bash
CONFIG_PATH=./agent.yml go run ./server/cmd/server --mode agent --center-url http://127.0.0.1:8080 --enroll-code <注册码>
```

`/n/<节点ID>/` 下的页面由后端内嵌的前端产物提供，修改前端后需执行 `npm run build`，才能在节点视角下看到变化。

## 权限与边界

- **只有创始人可以管理和操作节点。** 节点上的请求以受管机本地的创始人身份执行，与调用者无关，所以不能把节点权限下放给只有部分路由权限的成员。
- 受管机用**用户名排序后的第一个创始人**签发访问令牌，文件管理与终端使用该成员的家目录。受管机上必须至少有一个创始人，且不能启用代理 Header 登录（`tha.enabled`），否则启动失败。
- 中控模式下也不应启用代理 Header 登录：网关转发的请求在 isrvd 看来都来自本机，客户端可借此伪造登录头。启动时会给出警告。
- 受管机用 `server.jwtSecret` 加密本地凭据；修改该密钥后旧凭据无法解密，需要用新的注册码重新注册。
- 受管机转发 WebSocket 时没有 `Origin`，`agent` 模式会放行空 `Origin`；跨站来源已由中控网关按 `allowedOrigins` 校验（未配置时仅允许同源）。
- 节点令牌与领取密钥只保存哈希；注册码一次性使用、默认 1 小时过期（最长 7 天），用 `server.jwtSecret` 派生的密钥加密保存在 `node-code.yml` 中，创始人在其待接入期间可随时从列表查看接入命令（修改 `jwtSecret` 后未使用的注册码会被清除）；默认需人工审批；令牌可按节点吊销。
- 网关把请求转发给节点前会剥离 `Authorization`、`Cookie`、`token` 查询参数。
- 节点管理操作（生成注册码、审批、吊销、删除等）、节点上的写操作与 WebSocket 都会记入中控的审计页面，规则与 isrvd 一致：非 GET 与全部 WebSocket 记录，WebSocket 在会话结束时记录；操作人是中控登录用户，URI 带 `/n/<节点ID>/` 前缀，请求体按 isrvd 的规则脱敏。受管机本机的审计页面里同一操作记录为其本地创始人。
- AI 助手在节点页面下仍作用于中控本机。
- 配置重载（SIGHUP、保存系统配置或 etcd 变更）后，`allowedOrigins` 与 `rootDirectory` 的变化对中控即时生效（节点表与注册码随 `rootDirectory` 切换到新位置，与计划任务等业务数据一致）；`listenAddr`、`ISRVD_CENTER_TRUST_PROXY` 仍需重启。
- 中控为单实例设计；重启时所有节点会短暂断开，受管机按 1 秒起步的指数退避加抖动自动重连。
- 受管机与中控的版本需一致；协议版本不一致时节点显示「版本不兼容」并拒绝转发。

## 网关接口

以下接口由 `center` 模式的网关提供（不在 isrvd 的 `/api/*` 路由表中，也不出现在 OpenAPI 目录里）。响应格式与 isrvd 一致：`{"success":bool,"message":string,"payload":any}`。需要创始人的接口使用 `Authorization` 头认证。详细的请求与响应字段见 [受管节点](references/node/nodes.md) 与 [注册码](references/node/codes.md)。

| 方法与路径 | 认证 | 说明 |
|------------|------|------|
| `GET /api/node/nodes` | 创始人 | 节点列表，含 `status`（`pending`/`approved`/`revoked`）、`online`、`latency`、`agentVersion`、`compatible` 等 |
| `PUT /api/node/item/{id}` | 创始人 | 重命名，body `{"name":"…"}`（1–64 个字符） |
| `POST /api/node/item/{id}/approve` | 创始人 | 审批待接入的节点 |
| `POST /api/node/item/{id}/revoke` | 创始人 | 吊销：令牌立即失效并断开连接，需删除后重新注册才能接入 |
| `DELETE /api/node/item/{id}` | 创始人 | 删除并断开连接 |
| `GET /api/node/codes` | 创始人 | 尚未使用且未过期的注册码（含明文，用于查看接入命令） |
| `POST /api/node/code` | 创始人 | 生成注册码，body `{"name":"","ttlMinutes":60,"autoApprove":false}`；响应与 `GET /api/node/codes` 都包含 `code` 明文 |
| `DELETE /api/node/code/{id}` | 创始人 | 撤销注册码 |
| `POST /api/node/enroll` | 注册码 | 受管机提交注册，按来源 IP 限流 |
| `POST /api/node/enroll/claim` | 领取密钥 | 受管机领取令牌，按来源 IP 限流 |
| `GET /api/node/connect` | 节点令牌 | 受管机建立 WebSocket 隧道；令牌只经 `Authorization: Bearer` 头传递，带 `Origin` 的浏览器请求会被拒绝 |
| `/n/{id}/…` | 创始人 | 以某个节点的视角访问，见上文「工作方式」 |
