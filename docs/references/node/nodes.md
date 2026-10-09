# 受管节点 API

> 集中管理的节点接口，仅在 `isrvd --mode center`（中控）下可用，由进程内的网关提供，不在 isrvd 的 `/api/*` 路由表中，也不会出现在 `GET /account/routes` 与 OpenAPI 目录里。其他模式下这些路径返回 `404`。
> 能力是否启用可通过 `GET /overview/bootstrap` 响应中的 `probe.node` 判断。
>
> **仅创始人可调用**：节点上的请求以受管机本地的创始人身份执行，与调用者无关，所以无法把节点权限授予普通成员。非创始人返回 `403`，未登录返回 `401`，中控 isrvd 暂不可用返回 `502`。
>
> 部署、接入流程与安全边界见 [集中管理](../../multi-node.md)；注册码接口见 [注册码](codes.md)。

---

## 节点管理

### 查询节点列表

```bash
isrvd_get "/node/nodes"
```

**响应字段（节点）：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 节点 ID（UUID，只读） |
| `name` | string | 节点名称 |
| `status` | string | `pending` 待审批 \| `approved` 已审批 \| `revoked` 已吊销 |
| `hostname` | string | 注册时上报的主机名 |
| `os` | string | 注册时上报的操作系统 |
| `arch` | string | 注册时上报的 CPU 架构 |
| `createdAt` | string | 注册时间 |
| `online` | bool | 当前是否在线 |
| `latency` | number | 隧道往返延迟（毫秒），离线为 `0` |
| `agentVersion` | string | 受管机程序版本，离线时不返回 |
| `compatible` | bool | 协议版本是否与中控一致；不一致时拒绝转发 |
| `connectedAt` | string | 本次连接时间，离线时不返回 |
| `remoteAddr` | string | 受管机连接来源地址，离线时不返回 |

> 节点令牌与领取密钥只保存哈希，任何接口都不返回。`online`、`latency`、`agentVersion`、`compatible`、`connectedAt`、`remoteAddr` 是运行时状态，不持久化。

### 重命名节点

```bash
isrvd_put "/node/item/<ID>" '{"name":"杭州机房 web-01"}'
```

`name` 必填，1–64 个字符。

### 审批节点

```bash
isrvd_post "/node/item/<ID>/approve"
```

仅 `pending` 状态的节点可审批。审批后受管机在下一个轮询周期（约 10 秒）领取令牌并建立连接。

### 吊销节点

```bash
isrvd_post "/node/item/<ID>/revoke"
```

令牌立即失效并断开在线连接，之后经隧道的请求返回 `503 节点离线`。已吊销的节点需删除后重新注册才能接入。

> 受管机被拒绝后**不会删除本地凭据**，而是每分钟重试一次并记录错误日志；需要重新接入时，用新的注册码重启受管机即可。

### 删除节点

```bash
isrvd_delete "/node/item/<ID>"
```

删除记录并断开在线连接。

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
- 节点管理操作、节点上的写操作与 WebSocket 会记入中控的审计页面（`GET /system/audit/logs`），URI 带 `/n/<节点ID>/` 前缀，请求体按 isrvd 的规则脱敏；受管机本机的审计页面里同一操作记录为其本地创始人。

### `?token=` 的适用范围

与 isrvd 原生规则一致：只有 WebSocket 升级请求，以及 isrvd 声明了 `QueryToken` 的 `GET` 路由（文件下载、容器文件下载、容器/Swarm 实时日志）可以用 `?token=` 认证。**其余请求包括所有写操作与节点管理接口，必须使用 `Authorization` 头**，否则返回 `401`。这样可避免令牌进入访问日志、浏览器历史和 `Referer`。

---

## 受管机接入接口

以下接口由受管机自动调用，不需要手工操作；这里列出便于排查与对接。路径相对 API 根（`/api`），均不要求登录，由一次性凭据、节点令牌与按来源 IP 限流保护。

### 提交注册

```bash
isrvd_post "/node/enroll" '{
  "code": "<注册码>",
  "name": "web-01",
  "hostname": "web-01",
  "os": "linux",
  "arch": "amd64"
}'
```

`code` 必填；`name` 缺省取主机名。响应：`nodeId`、`claimSecret`（一次性领取密钥）、`status`（`pending` 或 `approved`）。按来源 IP 限流（60 次/分钟）；注册码使用一次即失效，无效或过期返回 `403`。

### 领取节点令牌

```bash
isrvd_post "/node/enroll/claim" '{"nodeId": "<节点ID>", "claimSecret": "<领取密钥>"}'
```

响应：`status`；审批通过后同时返回 `token`（节点令牌）。审批前返回 `pending`，节点首次成功连接前允许重复领取。同样按来源 IP 限流。

### 建立隧道

`GET /node/connect` 是 WebSocket 升级请求，无法用 `isrvd_*` 调用。令牌只经 `Authorization: Bearer <节点ID>.<密钥>` 头传递，不放在 URL 中；认证失败按来源 IP 限流（10 次/分钟）；带 `Origin` 的浏览器请求一律拒绝。
