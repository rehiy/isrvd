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

部署参数由进程环境提供，`ISRVD_MODE` 默认值为 `server`。各节点可以共享 etcd 中的业务配置，部署角色与令牌由各自环境独立提供。

## 工作方式

- 每个进程内部仍是完整的 isrvd：`center` 与 `agent` 模式下，isrvd 只监听 `127.0.0.1` 的随机端口。`center` 另起网关占用配置里的 `listenAddr` 作为对外入口；`agent` 不监听任何对外端口，配置里的 `listenAddr` 不生效。
- 浏览器访问 `/n/<节点ID>/`，页面仍由中控提供；其中的 Docker、Swarm、Compose、文件、本机进程、Shell 终端、计划任务、APISIX、Caddy、系统概览接口经隧道转发到该节点，包括流式日志、终端和文件上传下载。
- 账号、系统配置、AI 助手、SSH 远程管理始终由中控自己处理，即使在 `/n/<节点ID>/` 下访问也一样。
- 节点切换与节点管理是 webview 的一部分，**按需启用**：只有 `center` 模式（`/api/overview/bootstrap` 的 `probe.node` 为 `true`）且当前用户有 `GET /api/node/nodes` 权限时，才会出现头部的节点切换器；侧边栏「节点管理」还要求当前处于本机（中控）视角（位于「用户管理」下方，一个页面）。单机模式、`agent` 模式和无节点列表权限的成员看不到任何节点界面，也不会发出节点接口请求，直接访问 `#/node` 会回到概览。
- 切换节点是一次整页跳转：地址变为 `/n/<节点ID>/`，页面停留在当前功能页（带资源参数的详情页回到概览，从中控管理页面切换到子节点时也回到概览）。子节点视角隐藏「节点管理」「用户管理」「系统配置」「操作审计」与「SSH 远程管理」，直接访问这些页面会回到概览；头部节点切换器与个人账户设置保留。此限制仅影响前端页面，相关接口作用于中控，并执行中控的认证与权限校验。
- 认证与权限复用中控当前的账户服务：网关在进程内验证调用者凭据并检查请求对应的路由权限。成员权限、密码或 JWT 密钥变更在配置发布后对新请求立即生效；已建立的长连接不会逐条消息重新认证。应用未就绪或正在重载时，节点管理与转发返回 `502`，避免前端误判为登录过期。节点视角的页面启动仍聚合中控身份与节点服务能力，中控上游不可达时同样返回 `502`。
- 中控网关在请求带 `Upgrade: websocket` 或命中声明了 `QueryToken` 的 `GET` 路由时可从 `?token=` 读取令牌；无论凭据来自哪里，节点管理和转发仍须通过中控身份认证与路由权限校验。普通请求不要把令牌放进 URL，避免进入访问日志、浏览器历史记录或 Referer。

## 部署

**中控**：设置环境变量 `ISRVD_MODE=center`，使用 `server.listenAddr`，然后重启。

**受控机**：先在中控「接入节点」生成 token，复制弹窗中的 Docker 或裸机安装命令，在受控机以 root 身份执行。两种命令均通过 `bash <(curl -sL https://isrvd.rehiy.com/install/)` 调用安装脚本，分别使用 `install --docker` 和 `install`，并导出接入所需的环境变量。已有服务则修改启动环境并重启。切换后只能经中控访问；地址填错时需修改受控机环境变量并重启。

token 无时效限制，首次连接时激活为节点；之后沿用相同 token，直到吊销或删除。**同一 token 只允许一个在线或正在握手的客户端**；重复连接返回 `409`，不会顶替已有会话，断线后可用同一 token 重连。

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

安装脚本的 Docker `update` 会保留现有容器的 `ISRVD_MODE`、`ISRVD_CENTER_URL`、`ISRVD_NODE_TOKEN`。更新时将旧容器临时更名为 `isrvd-update-backup`，新容器通过启动检查后才删除旧容器；创建或启动检查失败时恢复旧容器，保留其原有镜像、环境、挂载和端口。若更新中断后遗留备份容器，需先处理该容器再重试。启动检查只确认容器仍在运行，不代表中控连接已成功。

**systemd**：通过 `systemctl edit isrvd` 在 `[Service]` 中设置 `Environment="ISRVD_MODE=center"`；受控机还需设置 `ISRVD_CENTER_URL`、`ISRVD_NODE_TOKEN`。执行 `systemctl daemon-reload` 与 `systemctl restart isrvd`。安装默认以单机模式启动；在执行脚本 `install` 前导出 `ISRVD_MODE`（受控机同时导出 `ISRVD_CENTER_URL`、`ISRVD_NODE_TOKEN`），脚本会将其写入仅 root 可读的 `installer.conf` drop-in，Docker 安装则传入容器环境。agent 模式不发布 Docker 的 8080 端口（Caddy/APISIX 业务端口仍按镜像发布）。`update` 保留已有角色，不读取安装器环境来切换角色。

```bash
export ISRVD_MODE=agent
export ISRVD_CENTER_URL=https://center.example.com
export ISRVD_NODE_TOKEN="<创建节点时生成的 token>"
bash isrvd.sh install --docker
```

| 环境变量 | 适用 | 说明 |
|----------|------|------|
| `ISRVD_MODE` | 全部 | `server`（默认）/ `center` / `agent` |
| `ISRVD_CENTER_URL` | agent，必填 | HTTP(S) 中控地址，不含路径、查询参数、片段或账号密码 |
| `ISRVD_NODE_TOKEN` | agent，必填 | 中控直接生成的 token；仅从进程环境读取，不通过配置 API 返回或修改 |

`ISRVD_CONFIG` 用于指定配置文件或 etcd 位置。保存 token 的环境文件应限制权限为 `0600`，不要提交或公开。

接入流程：在「节点管理」页点击「接入节点」填写必填备注并创建节点，保存仅在创建时显示的 token（列表中状态为「待接入」）→ 受管机携带令牌启动 → 首次连接时中控校验并激活该令牌，节点建立常连，状态变为「在线」。节点列表用 `pending`、`active`、`revoked` 区分待接入、已激活和已吊销。

**受管机被中控拒绝时**（节点被吊销或删除，或中控数据被重置、恢复了旧备份）受管机每分钟重试并记录错误日志。中控数据恢复后自动重连；确实要重新接入时，生成新的令牌并重启受管机。

数据位置：中控的节点表存储在 `node-list.yml` 中，与计划任务、SSH 主机等业务数据一样，保存在 `rootDirectory` 下，使用 etcd 时随配置一起同步。受控机的 token 由环境变量 `ISRVD_NODE_TOKEN` 提供，不写入主配置。

## 本地调试

使用 `ISRVD_MODE=center ./develop.sh` 调试节点管理；`.local.yml` 仅保存业务配置。前端开发服务器（`3000`）同时代理 `/api/`、`/openapi/` 和节点视角的 `/n/`。

要在本机调试一个受管机，另开终端用另一份配置（不同的 `rootDirectory`）启动：

```bash
ISRVD_MODE=agent ISRVD_CENTER_URL=https://center.example.com ISRVD_NODE_TOKEN="<token>" ISRVD_CONFIG=./agent.yml go run ./server/cmd/server
```

`/n/<节点ID>/` 下的页面由后端内嵌的前端产物提供，修改前端后需执行 `npm run build`，才能在节点视角下看到变化。

## 权限与边界

- **节点管理和业务操作按路由权限授权，创始人默认拥有全部权限。** 节点管理权限在中控的 `GET /account/routes` 中以 `node` 模块列出，可在成员管理中分配。转发业务请求先按中控成员的实际业务权限检查（去掉 `/n/<节点ID>` 前缀，例如 `GET /api/docker/containers`），再以受管机本地服务令牌执行；未授权请求返回 `403`。业务权限对所有受管节点生效，目前不按节点 ID 单独授权；节点列表权限只允许发现与切换节点，不额外赋予业务操作权限。
- 受管机用**用户名排序后的第一个创始人**签发访问令牌，文件管理与终端使用该成员的家目录。受管机上必须至少有一个创始人，且不能启用代理 Header 登录（`tha.enabled`），否则启动失败。
- 中控模式允许代理 Header 登录（`center + tha.enabled`）。网关按连接对端 IP 和 `tha.trustedCIDRs` 校验身份头，不采信转发头；可信代理必须覆盖客户端传入的身份头。THA 与 JWT 均使用当前配置判断权限，普通成员仍不能管理节点。中控的私有回环服务只接收网关写入的内部身份，外部同名头会先被删除；THA 头名重载不会重新信任原始请求头。本机进程属于可信边界，用户身份不下发给受控节点。
- 受管机转发 WebSocket 时没有 `Origin`，`agent` 模式会放行空 `Origin`；跨站来源已由中控网关按 `allowedOrigins` 校验（未配置时仅允许同源）。
- 创建节点时生成 86 字符的纯字母数字随机 token，无时效限制，仅在创建响应和本次接入命令中显示；`node-list.yml` 从创建起只保存 SHA-256 哈希。首次认证通过将状态从 `pending` 改为 `active`，此后沿用同一 token；吊销或删除立即使其失效。JWT 密钥变更不影响 token。遗失 token 时删除节点并重新创建。
- 网关把请求转发给节点前会剥离 `Authorization`、`Cookie`、`token` 查询参数。
- 节点管理操作（创建、吊销、删除节点等）、节点上的写操作与 WebSocket 都会记入中控的审计页面，规则与 isrvd 一致：非 GET 与全部 WebSocket 记录，WebSocket 在会话结束时记录；操作人是中控登录用户，URI 带 `/n/<节点ID>/` 前缀，请求体按 isrvd 的规则脱敏。受管机本机的审计页面里同一操作记录为其本地创始人。
- 所有审计与节点 token 认证失败限流统一使用连接对端 IP，不采信客户端的 `X-Forwarded-For` 或 `X-Real-IP`；向中控内部服务转发时重新生成来源 IP 头，不透传外部来源链。中控上游与受管机本机的审计因此只记录回环或隧道来源，操作的真实客户端 IP 以中控网关的审计记录为准。部署在反向代理后，审计记录代理 IP，同一代理后的客户端共享节点认证失败限流额度。
- AI 助手在节点页面下仍作用于中控本机。
- 配置重载（SIGHUP、保存系统配置或 etcd 变更）后，`allowedOrigins` 与 `rootDirectory` 的变化对中控即时生效（节点表随 `rootDirectory` 切换到新位置，与计划任务等业务数据一致）；部署环境变量与 `listenAddr` 仍需重启。
- 中控为单实例设计；重启时所有节点会短暂断开，受管机按 1 秒起步的指数退避加抖动自动重连。
- 受管机与中控的版本需一致；协议版本不一致时节点显示「版本不兼容」并拒绝转发。

## 网关接口

以下接口由 `center` 模式的网关提供（不在 isrvd 的 HTTP handler 注册表中，也不出现在 OpenAPI 目录里；成员权限列表会列出节点管理接口）。响应格式与 isrvd 一致：`{"success":bool,"message":string,"payload":any}`。节点管理接口需要对应的路由权限，使用 `Authorization` 头认证。详细的请求与响应字段见 [受管节点](references/node/nodes.md)。

| 方法与路径 | 认证 | 说明 |
|------------|------|------|
| `POST /api/node/item` | 对应路由权限 | 创建待接入节点，body `{"name":"杭州机房 web-01"}`；仅本次响应包含 `token` |
| `GET /api/node/nodes` | 对应路由权限 | 节点列表，含 `status`（`pending`/`active`/`revoked`）、`online`、`latency`、`agentVersion`、`compatible` 等 |
| `PUT /api/node/item/{id}` | 对应路由权限 | 重命名，body `{"name":"…"}`（1–64 个字符） |
| `POST /api/node/item/{id}/revoke` | 对应路由权限 | 吊销：令牌立即失效并断开连接，需删除后用新令牌重新接入 |
| `DELETE /api/node/item/{id}` | 对应路由权限 | 删除并断开连接 |
| `GET /api/node/connect` | 节点令牌 | 受管机建立 WebSocket 隧道，首次连接时激活待接入的令牌；令牌只经 `Authorization: Bearer` 头传递，带 `Origin` 的浏览器请求会被拒绝 |
| `/n/{id}/…` | 对应路由权限 | 以某个节点的视角访问，见上文「工作方式」 |

## 代码职责与文件组织

节点记录、认证和在线会话分别组织，仍聚合在 `server/service/node` 包中：

| 位置 | 职责 |
|------|------|
| `node/node.go` | `Node` 持久化实体、`NodeInfo` 接口响应、创建/重命名请求及校验 |
| `node/service.go` | 服务生命周期、节点创建/查询/重命名/吊销/删除 |
| `node/store.go` | `node-list.yml` 的读取、并发修改、提交和数据目录切换；修改成功后返回已提交的快照 |
| `node/auth.go` | 握手认证与限流、token 生成和哈希；认证、吊销、删除共用服务锁，避免重复接入和吊销竞态 |
| `node/session.go` | 会话登记/释放、在线状态投影、隧道传输和延迟采样；协议协商继承连接的 context |
| `node/agent.go` | 两端共用的协商协议与受管机常连客户端 |
| `node/proxy.go`、`node/permission.go` | 节点业务转发及共享权限目录 |

表中 `node/` 均指 `server/service/node/`。部署模式、中心地址及 token 的环境变量由 `server/config/cluster.go` 统一读取、校验。存储层只接收 `Node`，在线状态通过 `NodeInfo` 投影到响应中，不进入 YAML；接口的 JSON 字段继续保持扁平结构。

网关 `server/gateway` 按 HTTP 职责组织：`gateway.go` 负责装配和路由，`node.go` 处理节点管理与 WebSocket 接入，`proxy.go` 负责节点转发与启动数据聚合，`middleware.go` 复用中控认证/权限/审计，`response.go` 统一输出 JSON。网关不构造账户或审计服务，始终通过 `Options` 使用应用当前实例。

前端 `views/node/nodes.vue` 负责节点列表、搜索、轮询和弹窗；`widget/node-actions.vue` 统一桌面与移动端的重命名、吊销、删除按钮及权限检查；创建和编辑表单由各自的 modal 维护。
