# 受管节点 API

> 集中管理的节点接口，仅在 `ISRVD_MODE=center`（中控）下可用，由进程内的网关提供，不由 isrvd 注册 HTTP handler；权限元信息在中控 `GET /account/routes` 中列出，OpenAPI 目录不包含这些网关接口。其他模式下这些路径返回 `404`。
> 能力是否启用可通过 `GET /overview/bootstrap` 响应中的 `probe.node` 判断。
>
> **按路由权限调用**：节点管理检查对应的 `METHOD /api/node/...` 权限，业务转发检查去掉 `/n/<节点ID>` 前缀后的业务路由权限（参数使用 `:id` 等规范路径）。创始人默认拥有全部权限；普通成员可在成员管理中授权。权限不足返回 `403`，未登录返回 `401`，中控应用未就绪或正在重载时返回 `502`。身份与权限由进程内当前账户服务验证，权限变更对新请求立即生效，已建立的长连接不会逐条消息重新认证。权限对所有节点生效，目前不支持按节点 ID 单独授权。
>
> 部署、接入流程与安全边界见 [集中管理](../../multi-node.md)。

---

## 节点管理

### 查询节点列表

```bash
isrvd_get "/node/nodes"
```

返回全部节点，通过 `status` 区分待接入、已激活和已吊销。

**响应字段（节点）：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 节点 ID（UUID，只读） |
| `name` | string | 节点名称 |
| `status` | string | `pending` 待接入 \| `active` 已激活 \| `revoked` 已吊销 |
| `hostname` | string | 注册时上报的主机名 |
| `os` | string | 注册时上报的操作系统 |
| `arch` | string | 注册时上报的 CPU 架构 |
| `createdAt` | string | 节点创建时间（只读） |
| `portmaps` | array | 端口转发规则，未配置时不返回；字段见下文「配置端口转发」 |
| `online` | bool | 当前是否在线 |
| `latency` | number | 隧道往返延迟（毫秒），离线为 `0` |
| `agentVersion` | string | 受管机程序版本，离线时不返回 |
| `compatible` | bool | 协议版本是否与中控一致；不一致时拒绝转发 |
| `connectedAt` | string | 本次连接时间，离线时不返回 |
| `remoteAddr` | string | 受管机连接来源地址，离线时不返回 |

> 节点令牌仅保存哈希；令牌只在创建节点时返回一次，列表接口不返回。`online`、`latency`、`agentVersion`、`compatible`、`connectedAt`、`remoteAddr` 是运行时状态，不持久化。

### 创建节点并生成 token

```bash
isrvd_post "/node" '{"name":"杭州机房 web-01"}'
```

| 请求字段 | 类型 | 说明 |
|------|------|------|
| `name` | string | 备注，必填；去除首尾空白后须为 1–64 个字符，作为节点名称 |

| 响应字段 | 类型 | 说明 |
|------|------|------|
| `node` | object | 上表中的节点对象，初始 `status` 为 `pending` |
| `token` | string | 86 字符的纯字母数字随机 token，无时效限制，仅在创建时返回 |

中控在 `node-list.yml` 中只保存 token 的 SHA-256 哈希（`tokenHash`），不保存明文或可解密密文。列表和更新响应均不返回 token；遗失后删除节点并重新创建。JWT 密钥变更不影响节点 token。

创建后在受管机以 root 身份任选一种方式安装：

```bash
export ISRVD_MODE=agent
export ISRVD_CENTER_URL=https://center.example.com
export ISRVD_NODE_TOKEN="<创建时返回的 token>"
# 裸机安装（systemd）
bash <(curl -sL https://isrvd.rehiy.com/install/) install
# 或使用 Docker 安装
bash <(curl -sL https://isrvd.rehiy.com/install/) install --docker
```

安装脚本会保存这些环境变量。已有服务需更新启动环境并重启。首次认证通过后节点状态变为 `active`，重启和重连继续使用同一 token。

### 重命名节点

```bash
isrvd_put "/node/<ID>" '{"name":"杭州机房 web-01"}'
```

`name` 必填，1–64 个字符。

### 吊销节点

```bash
isrvd_post "/node/<ID>/revoke"
```

令牌立即失效并断开在线连接，之后经隧道的请求返回 `503 节点离线`；该节点的端口转发规则一并清除。已吊销的节点需删除后用新令牌重新接入。

> 受管机被拒绝后每分钟重试一次并记录错误日志；需要重新接入时，创建新节点并更新受管机 token 后重启即可。

### 删除节点

```bash
isrvd_delete "/node/<ID>"
```

删除记录并断开在线连接，原 token 立即失效，同时停止该节点的端口转发监听。待接入节点同样可重命名、吊销或删除；保存失败时保留原记录并返回错误，可重试操作。

### 配置端口转发

在中控监听端口，经隧道转发到节点侧的任意 `host:port`（类似 frp），支持 TCP 与 UDP；也可在 Web 界面「集中管控 → 端口转发」中管理。需先在中控配置文件设置 `portmap.range`（可分配的监听端口范围，如 `20000-20999`；留空则功能关闭），修改后重启生效，详见 [集中管理](../../multi-node.md)。

```bash
isrvd_put "/node/<ID>/portmap" '{"portmaps":[{"name":"SSH","proto":"tcp","listen":20022,"target":"127.0.0.1:22"},{"proto":"udp","listen":25353,"target":"127.0.0.1:53"}]}'
```

请求体整体替换该节点的规则；传 `{"portmaps":[]}` 清空。保存前会先确认新增端口可监听，失败则不保存任何变更。

| 字段 | 类型 | 说明 |
|------|------|------|
| `name` | string | 备注，可选，最多 64 个字符 |
| `proto` | string | `tcp` 或 `udp` |
| `listen` | number | 中控监听端口，须在配置 `portmap.range` 范围内；同一 `proto` 下全局唯一（跨节点） |
| `target` | string | 节点侧目标 `host:port`；`host` 缺省为 `127.0.0.1`，可为 IP 或主机名 |

- 每个节点最多 32 条规则；已吊销的节点不能配置。
- 监听地址由中控配置 `portmap.bind` 决定，默认 `127.0.0.1`；对外开放须显式设为 `0.0.0.0` 等地址。
- 监听与节点是否在线无关：节点离线时连接被立即关闭，上线后自动恢复。规则变更、节点吊销或删除时，对应监听与已建立的连接立即断开。
- 受管机程序须支持端口转发（在线节点不支持时返回 `400`）；规则写入 `node-list.yml`，中控重启后自动恢复监听，端口被占用的规则只记录告警。
- **转发的连接不经过 isrvd 认证，也不进入审计页面**，鉴权完全由目标应用负责；该权限等同于访问节点网络内任意地址，请仅授予创始人，且勿转发无认证的服务。UDP 与 TCP 到达目标时，来源地址均为受管机本机。
- 并发上限：每条 TCP 规则 512 个连接，每条 UDP 规则 256 个来源地址（空闲 60 秒回收）。

---

## 以节点视角访问

`/n/<节点ID>/` 之下的路径等价于直接访问中控，只是业务接口会经隧道转发到该节点。使用脚本时，把 API 根路径换成带节点前缀的地址即可：

```bash
# API_ROOT 形如 https://center.example.com/n/<节点ID>/api
python3 ./scripts/api.py token "$API_ROOT" "$TOKEN"
python3 ./scripts/api.py get "/docker/containers"
```

| 类别 | 模块 | 处理位置 |
|------|------|----------|
| 经隧道转发到节点 | `overview`、`filer`、`shell`、`local`、`docker`、`swarm`、`compose`、`cron`、`apisix`、`caddy` | 受管机 |
| 始终由中控处理 | `account`、`system`、`copilot`、`ssh`、`node` | 中控 |
| 页面与静态资源 | 其余路径 | 中控 |

- `GET /n/<ID>/api/overview/bootstrap` 的认证信息取自中控，`probe` 中的 `docker`、`swarm`、`compose`、`apisix`、`caddy` 取自节点，`node`、`copilot` 取自中控；节点不可达时这几项全部为 `false`。
- 节点不存在返回 `404`；离线或已吊销返回 `503`；协议版本不一致、隧道故障返回 `502`；节点响应超时返回 `504`。
- 节点管理操作、节点上的写操作与 WebSocket 会记入中控的审计页面（`GET /system/audit/logs`），URI 带 `/n/<节点ID>/` 前缀，请求体按 isrvd 的规则脱敏；空成功响应记为 `200`，WebSocket 升级成功记为 `101`；受管机本机的审计页面里同一操作记录为其本地创始人。

### `?token=` 的适用范围

与 isrvd 原生规则一致：只有 WebSocket 升级请求，以及 isrvd 声明了 `QueryToken` 的 `GET` 路由（文件下载、容器文件下载、容器/Swarm 实时日志）可以用 `?token=` 认证。**其余请求包括所有写操作与节点管理接口，必须使用 `Authorization` 头**，否则返回 `401`。这样可避免令牌进入访问日志、浏览器历史和 `Referer`。

---

## 受管机接入接口

以下接口由受管机自动调用，不需要手工操作；这里列出便于排查与对接。路径相对 API 根（`/api`），不要求登录，由节点令牌与按来源 IP 限流保护。

### 建立隧道

`GET /node/connect` 是 WebSocket 升级请求，无法用 `isrvd_*` 调用。令牌只经 `Authorization: Bearer <token>` 头传递，不放在 URL 中；中控按完整 token 的 SHA-256 哈希查找节点，待接入节点在首次认证通过时更新为 `active`；认证失败按连接对端 IP 限流（10 次/分钟），忽略 `X-Forwarded-For` 等转发头；通过反向代理接入时，同一代理后的客户端共享限流额度；带 `Origin` 的浏览器请求一律拒绝。同一 token 已在线或握手中时返回 `409`，不会断开旧连接；升级失败会释放握手占用，断线后相同 token 可再次连接。
