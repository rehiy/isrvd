# 集中管理（多服务器）

同一个 `isrvd` 二进制，通过环境变量 `ISRVD_MODE` 选择角色。部署参数不存入 YAML 或 etcd，也不由配置 API 管理；修改后重启生效：

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

各节点可共享 etcd 中的业务配置，角色与令牌由各自环境独立提供。

## 工作方式

- 每个进程内部仍是完整的 isrvd：`center` 与 `agent` 模式下，isrvd 只监听 `127.0.0.1` 的随机端口。`center` 另起网关占用配置里的 `listenAddr` 作为对外入口；`agent` 不监听任何对外端口，`listenAddr` 不生效。
- 浏览器访问 `/n/<节点ID>/`，页面仍由中控提供；Docker、Swarm、Compose、文件、本机进程、Shell 终端、计划任务、APISIX、Caddy、系统概览接口经隧道转发到该节点，包括流式日志、终端和文件上传下载。
- 新建节点通过 `libgo` 的 `strutil.Rand(10)` 生成 10 位字母数字随机 ID（`0-9A-Za-z`，区分大小写，如 `aB3dE6gH9j`），保存时检查重复，碰撞最多重试 16 次，耗尽则创建失败；已有节点 ID 不迁移，原有 UUID 和 8 位短 ID 的链接继续有效。节点 ID 不是认证凭据，接入仍使用独立的 token。
- 账号、系统配置、AI 助手、SSH 与 SFTP 远程管理始终由中控自己处理，即使在 `/n/<节点ID>/` 下访问也一样。
- 节点切换与节点管理**按需启用**：只有 `center` 模式（`/api/overview/bootstrap` 的 `probe.node` 为 `true`）且当前用户有 `GET /api/node/nodes` 权限时，才出现头部节点切换器；侧边栏「集中管控」（位于「用户管理」下方）还要求处于中控视角，包含「节点管理」（`#/node/access`）与「端口转发」（`#/node/portmaps`）。其他情况看不到任何节点界面，直接访问 `#/node` 会回到概览。
- 切换节点是整页跳转到 `/n/<节点ID>/`，停留在当前功能页（带资源参数的详情页、从中控管理页切换时回到概览）。子节点视角隐藏「节点管理」「用户管理」「系统配置」「操作审计」与「SSH 远程管理」，直接访问回到概览；这只是前端限制，相关接口仍作用于中控并执行中控的认证与权限校验。
- 认证与权限复用中控的账户服务：网关在进程内验证凭据并检查路由权限。权限、密码或 JWT 密钥变更对新请求立即生效，已建立的长连接不逐条重新认证。应用未就绪或正在重载时，节点管理与转发返回 `502`，避免前端误判为登录过期；中控上游不可达时节点视角的启动数据同样返回 `502`。
- 请求带 `Upgrade: websocket` 或命中声明 `QueryToken` 的 `GET` 路由时可从 `?token=` 读取令牌；普通请求不要把令牌放进 URL。无论凭据来自哪里，仍须通过中控认证与路由权限校验。

## 部署

**受管机接入流程**：在「集中管控 → 节点管理」点击「接入节点」，填写名称并创建，保存仅创建时显示的 token（状态为「待接入」）→ 复制弹窗中的安装命令在受管机以 root 执行 → 受管机携带 token 启动，首次连接时中控校验并激活，状态变为「在线」。

**token 规则**：创建节点时生成 86 位字母数字随机 token，无时效限制，`node-list.yml` 只保存其 SHA-256 哈希；状态 `pending`（待接入）→ `active`（已激活）→ `revoked`（已吊销）。吊销或删除立即使其失效并断开连接；JWT 密钥变更不影响 token；遗失 token 时删除节点重新创建。**同一 token 只允许一个在线或正在握手的客户端**，重复连接返回 `409`，不会顶替已有会话。

**受管机被中控拒绝时**（节点被吊销或删除，或中控数据被重置、恢复了旧备份），受管机每分钟重试并记录错误日志；中控数据恢复后自动重连，需重新接入时生成新 token 并重启受管机。

**环境变量**：

| 环境变量 | 适用 | 说明 |
|----------|------|------|
| `ISRVD_MODE` | 全部 | `server`（默认）/ `center` / `agent` |
| `ISRVD_CENTER_URL` | agent，必填 | HTTP(S) 中控地址，不含路径、查询参数、片段或账号密码 |
| `ISRVD_NODE_TOKEN` | agent，必填 | 中控生成的 token，长度 64–96 且不含空白；仅从进程环境读取，不通过配置 API 返回或修改 |

`ISRVD_CONFIG` 指定配置文件或 etcd 位置。保存 token 的环境文件应限制权限为 `0600`，不要提交或公开。

**裸机启动示例**（受管机）：

```bash
export ISRVD_MODE=agent
export ISRVD_CENTER_URL=https://center.example.com
export ISRVD_NODE_TOKEN='<中控生成的 token>'
isrvd
```

**systemd**：`systemctl edit isrvd` 在 `[Service]` 中设置 `Environment="ISRVD_MODE=center"`（受管机另设 `ISRVD_CENTER_URL`、`ISRVD_NODE_TOKEN`），再执行 `systemctl daemon-reload` 与 `systemctl restart isrvd`。

**安装脚本**：系统服务使用 `/install/` 或本地 `isrvd-system.sh`，全容器化使用 `/install/?script=docker` 或本地 `isrvd-docker.sh`。在 `install` 前导出上述环境变量；默认角色为单机，系统服务写入仅 root 可读的 systemd drop-in（`installer.conf`），Docker 脚本传入容器环境。Docker 的 agent 模式不发布 8080（Caddy/APISIX 业务端口仍按镜像发布）。更新及卸载使用对应脚本，`update` 保留现有角色，不读取安装器环境来切换角色。

系统服务选择组件时可执行 `bash isrvd-system.sh install --components caddy --data /mnt/isrvd-data`；适用于单机、中控和受管机，组件数据目录默认 `/srv/data`，isrvd 主配置仍为 `/usr/local/isrvd/config.yml`。目录须为非根绝对路径且不能与安装目录重叠，组件子路径不能通过符号链接重定向。系统服务 `update` 从所属组件的实际 bind 挂载恢复目录，不接受 `--data` 迁移数据；挂载路径不一致时在停服前中止。`uninstall` 保留实际组件数据目录，删除主配置前请备份。

```bash
export ISRVD_MODE=agent
export ISRVD_CENTER_URL=https://center.example.com
export ISRVD_NODE_TOKEN="<创建节点时生成的 token>"
bash isrvd-docker.sh install
```

Docker `update` 将旧容器临时更名为 `isrvd-update-backup`，新容器通过启动检查（仅确认容器仍在运行，不代表已连上中控）后才删除旧容器；创建或启动检查失败时恢复旧容器。更新中断后遗留备份容器时，需先处理再重试。

**Docker 手动运行**：提前准备完整的 `/srv/data/conf/isrvd.yml`，中控和受管机使用各自的数据目录：

```bash
# 中控（8080 为对外入口）
docker run -d --name isrvd-center -e ISRVD_MODE=center --network sdnet -p 8080:8080 \
  -v /srv/data:/data -v /var/run/docker.sock:/var/run/docker.sock rehiy/isrvd:slim

# 受管机（不需要 -p）
docker run -d --name isrvd-agent --network sdnet \
  -e ISRVD_MODE=agent -e ISRVD_CENTER_URL=https://center.example.com \
  -e ISRVD_NODE_TOKEN="<中控生成的 token>" \
  -v /srv/data:/data -v /var/run/docker.sock:/var/run/docker.sock rehiy/isrvd:slim
```

**数据位置**：中控的节点表存于 `node-list.yml`，与计划任务等业务数据一样位于 `rootDirectory`，使用 etcd 时随配置同步。受管机的 token 由环境变量提供，不写入主配置。

## 本地调试

`./develop.sh` 默认以 `ISRVD_MODE=center` 启动后端，可直接调试节点管理；`.local.yml` 仅保存业务配置。前端开发服务器（`3000`）同时代理 `/api/`、`/openapi/` 和 `/n/`。

另开终端用另一份配置（不同的 `rootDirectory`）可在本机调试受管机：

```bash
ISRVD_MODE=agent ISRVD_CENTER_URL=https://center.example.com ISRVD_NODE_TOKEN="<token>" ISRVD_CONFIG=./agent.yml go run ./server/cmd/server
```

`/n/<节点ID>/` 下的页面由后端内嵌的前端产物提供，修改前端后需执行 `npm run build` 才能在节点视角下看到变化。

## 端口转发（类似 frp）

在「集中管控 → 端口转发」点击「添加转发」，或调用 `PUT /api/node/{id}/portmap`，让中控监听一个端口，把 TCP/UDP 流量经隧道转发到节点侧任意 `host:port`（`host` 缺省为节点本机 `127.0.0.1`）。规则由中控管理并存入 `node-list.yml`，受管机无需额外配置。

```text
客户端 ──TCP/UDP──> center:20022 ──隧道流──> agent ──> 127.0.0.1:22
```

- 先在中控配置的 `portmap` 分区设置，修改后需重启中控生效（也可经 `GET|PUT /api/system/config` 读写，格式无效返回 `400`）：

  ```yaml
  portmap:
    range: 20000-20999  # 可分配的监听端口范围（或单个端口）；留空则功能关闭
    bind: 127.0.0.1     # 监听地址（IP），默认只绑回环；对外开放须显式设为 0.0.0.0 等地址
  ```

  这里的「端口映射」指中控把端口经隧道映射到节点侧地址，与 Docker 容器端口映射无关。
- 同一 `proto` 的监听端口在所有节点间唯一。保存规则时先确认端口可监听，失败则不保存；中控重启后自动恢复监听，端口被占用或超出当前 `portmap.range` 的规则只记录告警、不恢复。
- 监听与节点是否在线无关：离线时连接被立即关闭，上线后自动恢复；规则变更、节点吊销或删除时，对应监听与已建立的连接立即断开。
- 隧道里转发流与 HTTP 流共用同一条连接，以首字节区分；UDP 按「2 字节长度 + 数据报」分帧。协商成功且受管机 `features` 不含 `portmap` 时，保存规则会被拒绝（清空规则不受限）；节点离线或无法协商时放行。
- **转发的连接不经过 isrvd 的认证、权限与审计，鉴权完全由目标应用负责**。该权限等同于访问节点网络内任意地址，请仅授予创始人，且勿转发无认证的服务。目标应用看到的来源地址是受管机本机。
- 并发上限：每条 TCP 规则 512 个连接，每条 UDP 规则 256 个来源（空闲 60 秒回收）。
- Docker 部署：中控容器需发布监听范围（如 `-p 20000-20999:20000-20999 -p 20000-20999:20000-20999/udp`），并设置 `portmap.bind: 0.0.0.0`。使用 `isrvd-docker.sh` 时先在 `/srv/data/conf/isrvd.yml` 写好 `portmap.range` 与 `portmap.bind` 再执行 `update`，脚本按该范围发布 TCP/UDP 端口。受管机在容器内时，`127.0.0.1` 指向容器自身，访问宿主机服务请使用宿主机地址或 host 网络。

## 权限与边界

- **节点管理和业务操作按路由权限授权，创始人默认拥有全部权限。** 节点管理权限在 `GET /account/routes` 中以 `node` 模块列出。转发请求先按中控成员的业务权限检查（去掉 `/n/<节点ID>` 前缀，如 `GET /api/docker/containers`），再以受管机本地服务令牌执行；未授权返回 `403`。业务权限对所有节点生效，不按节点单独授权；节点列表权限只允许发现与切换节点。
- 受管机用**用户名排序后的第一个创始人**签发访问令牌，文件管理与终端使用其家目录。受管机不能启用代理 Header 登录（`tha.enabled`），否则启动失败、保存配置返回 `400`。受管机没有创始人时节点仍可上线，但因无法签发本机令牌，所有转发请求返回 `401`。
- 中控允许代理 Header 登录（`center + tha.enabled`）：网关按连接对端 IP 和 `tha.trustedCIDRs` 校验身份头，不采信转发头，可信代理必须覆盖客户端传入的身份头；中控的私有回环服务只接收网关写入的内部身份，外部同名头会先被删除。普通成员仍不能管理节点，用户身份不下发给受管机。
- 受管机转发 WebSocket 时没有 `Origin`，`agent` 模式放行空 `Origin`；跨站来源已由中控网关按 `allowedOrigins` 校验（未配置时仅允许同源）。
- 网关转发前剥离 `Authorization`、`Cookie`、`token` 查询参数、THA 身份头与内部网关身份头；受管机侧再换成本机令牌。
- 节点管理操作、节点上的写操作与 WebSocket 都记入中控审计页面：规则同 isrvd（非 GET 与全部 WebSocket 记录，WebSocket 在会话结束时记录），操作人是中控登录用户，URI 带 `/n/<节点ID>/` 前缀，请求体脱敏。受管机本机审计中同一操作记录为其本地创始人。
- 审计与 token 认证失败限流统一使用连接对端 IP，不采信 `X-Forwarded-For` 或 `X-Real-IP`；转发到中控内部服务时重新生成来源 IP 头。因此中控上游与受管机本机审计只记录回环或隧道来源，真实客户端 IP 以网关审计为准；部署在反向代理后，审计记录代理 IP，同一代理后的客户端共享限流额度。
- AI 助手在节点页面下仍作用于中控本机。
- 配置重载后，`allowedOrigins` 与 `rootDirectory` 的变化对中控即时生效（节点表随 `rootDirectory` 切换）；部署环境变量、`listenAddr` 与 `portmap` 需重启。
- 中控为单实例设计；重启时节点短暂断开，受管机自动重连：指数退避每次翻倍，上限 60 秒，等待取 [d/2, d] 的随机值，连接稳定超过 30 秒后重置；被拒绝（401/403）时每 60 秒重试。
- 兼容性以能力协商为准，**不要求版本号一致**。中控连接后读取受管机隧道内的 `GET /agent/info`（返回 `mode`、`version`、`protocol`、`hostname`、`os`、`arch`、`features`）：基础 HTTP 转发默认放行，旧版受管机没有该接口或协商失败时同样放行；只有受管机协议版本低于中控的 `MinProtocolVersion` 时（当前均为 `1`，仅在基础传输格式不兼容变更时调高），节点显示「协议版本过低」并拒绝转发。可选功能（如端口转发）按 `features` 逐项判断，缺失时仅该功能不可用，节点列表以图标标出「部分功能不可用」。

## 网关接口

以下接口由 `center` 网关提供（不在 isrvd 的 HTTP handler 注册表和 OpenAPI 目录中；成员权限列表会列出节点管理接口）。响应格式同 isrvd：`{"success":bool,"message":string,"payload":any}`。请求与响应字段见 [受管节点](references/node/nodes.md)。

| 方法与路径 | 认证 | 说明 |
|------------|------|------|
| `GET /api/node/nodes` | 路由权限 | 节点列表 |
| `POST /api/node` | 路由权限 | 创建待接入节点，body `{"name":"…"}`；仅本次响应包含 `token` |
| `PUT /api/node/{id}` | 路由权限 | 重命名，body `{"name":"…"}`（1–64 个字符） |
| `POST /api/node/{id}/revoke` | 路由权限 | 吊销：令牌立即失效并断开连接，需删除后用新令牌重新接入 |
| `DELETE /api/node/{id}` | 路由权限 | 删除并断开连接 |
| `PUT /api/node/{id}/portmap` | 路由权限 | 整体替换节点的端口转发规则，见「端口转发」 |
| `GET /api/node/connect` | 节点令牌 | 受管机建立 WebSocket 隧道，首次连接激活待接入令牌；令牌只经 `Authorization: Bearer` 头传递，带 `Origin` 的请求被拒绝 |
| `/n/{id}/…` | 路由权限 | 以某个节点的视角访问 |

## 代码职责与文件组织

`server/service/node/`：

| 文件 | 职责 |
|------|------|
| `node.go` | `Node` 持久化实体、`NodeInfo` 响应、创建/重命名请求及校验 |
| `service.go` | 服务生命周期；节点创建（含 token 生成）/查询/重命名/吊销/删除 |
| `store.go` | `node-list.yml` 的读取、并发修改、提交和数据目录切换 |
| `auth.go` | 握手认证与限流、token 哈希比较、握手预占释放；认证、吊销、删除共用服务锁 |
| `session.go` | 会话登记/释放、在线状态投影、隧道传输、延迟采样与协议协商 |
| `agent.go` | 两端共用的协商协议与受管机常连客户端；隧道流按首字节分流为 HTTP 与端口转发 |
| `portmap.go` | 端口转发规则、校验、线路头部与 UDP 分帧、受管机侧中继 |
| `portmap_listener.go` | 中控侧 TCP/UDP 监听器与生命周期 |
| `proxy.go`、`permission.go` | 节点业务转发及共享权限目录 |

其他：

- `server/config/cluster.go` 读取、校验部署环境变量及 `portmap` 配置。
- `server/app/gateway_permission.go` 提供网关授权，并在 center 模式把节点权限并入路由索引；`server/cmd/server/launch.go` 按三种模式装配。
- `server/gateway/`：`gateway.go` 装配与路由，`node.go` 节点管理与 WebSocket 接入，`proxy.go` 节点转发与启动数据聚合，`middleware.go` 复用认证/权限/审计，`response.go` 统一 JSON 输出。
- 前端（`webview/src/`）：`views/node/nodes.vue`（节点管理）、`views/node/portmaps.vue`（端口转发）、`views/node/widget/`（`node-actions`、`node-create-modal`、`node-edit-modal`、`portmap-actions`、`node-portmap-modal`）、`helper/node.ts`、`helper/portmap.ts`、`component/node-switcher.vue`、`stores/node.ts`。后端按节点整体替换规则，`portmapPatch` 在每次增删改前先读取最新规则列表，避免覆盖他人修改。
