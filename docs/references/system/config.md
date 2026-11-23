# 系统配置与审计日志 API

## 配置存储

`ISRVD_CONFIG` 指定配置位置，未设置时读取 `./config.yml`（兼容旧变量 `CONFIG_PATH`）。

```bash
ISRVD_CONFIG=/data/conf/isrvd.yml ./isrvd
ISRVD_CONFIG="etcd://user:pass@127.0.0.1:2379/isrvd/config?fallback=/data/conf/isrvd.yml" ./isrvd
# 多节点 + 凭据通过环境变量传入
ETCD_USERNAME=user ETCD_PASSWORD=pass ISRVD_CONFIG="etcd://h1:2379,h2:2379,h3:2379/isrvd/config?scheme=https&timeout=5s" ./isrvd
```

说明：

- etcd value 使用同款 YAML；`ISRVD_CONFIG` 中的 path 是完整 etcd key，必须显式提供；系统配置推荐 key 为 `/isrvd/config`
- 支持多个节点，用逗号分隔 `host:port`
- 可选查询参数：`scheme`（默认 `http`，TLS 时用 `https`）、`timeout`（Go 时长格式，默认 `5s`）、`fallback`（本地 YAML 文件路径，仅在 etcd key 不存在时用于初始化）
- `ETCD_USERNAME` / `ETCD_PASSWORD` 环境变量优先于 URI 中的账号密码，生产环境建议用环境变量传入凭据；URI 中的特殊字符需 URL encode

### 业务数据存储

所有业务数据文件（计划任务 `cron.yml`、SSH 主机 `webssh-host.yml`、SSH 凭据 `webssh-cred.yml`、中控节点表 `node-list.yml`、旧版 Compose 历史导入文件 `compose-<sha256>.yml`）跟随配置存储后端：

| 配置后端 | 业务数据位置 |
|----------|--------------|
| 本地文件 | `server.rootDirectory/<文件名>` |
| etcd | `<配置 key>/<文件名>`（如 `/isrvd/config/cron.yml`）；key 缺失时导入 `rootDirectory` 下同名文件并写入 etcd，之后读写 etcd |

计划任务执行日志、审计日志、监控历史等日志类数据仍写本地磁盘。修改 `rootDirectory` 并重载后，新建服务使用新的本地目录或 fallback 目录；已存在的 etcd 数据不被覆盖。

### 保存一致性

系统配置、计划任务和 SSH 主机/凭据保存前检查目标存储中的现有内容是否与最近一次成功加载或保存的一致；发现已变更（包括被删除）时拒绝覆盖，请重新加载后重试。加载失败的计划任务存储禁止写入，避免空任务表覆盖原有数据。

本地文件使用同目录临时文件和原子替换，保留已有文件权限，新文件 `0600`。etcd 通过事务比较 key 的存在性和原始值再条件写入，支持跨实例防覆盖；fallback 也只在 key 缺失时原子创建。值比较不检测 ABA，也不提供浏览器旧草稿版本校验或分布式任务执行锁；etcd 超时不代表服务端一定未提交。etcd 空字符串值视为已存在，不触发 fallback。

etcd 配置监听在首次连接、重连后重新读取最新状态，读取失败会重试，存储快照与当前运行配置一致时跳过重载；只补偿连接间隙的最终状态，不重放历史事件。cron、webssh 的外部数据更新仍需重载服务。

启动和重载时，APISIX、Caddy 与 Swarm 的初始化探活并行执行，共用 5 秒超时；Swarm 仍需先创建 Docker 客户端。探活失败的模块标记为不可用，其余服务继续初始化。已配置地址的 APISIX 和 Caddy 初始化失败后，每 15 秒重新探测，成功后恢复业务、指标与故障探测。

## 配置段说明

配置文件（`config.yml` 或 etcd value）的顶层配置段：

| 配置段 | 说明 |
| -------- | ------ |
| `schema` | 配置结构版本，用于格式转换 |
| `server` | 浏览器标签页标题、监听地址、数据根目录、上传限制、CORS、JWT 密钥/有效期、持久化数据加密密钥、OpenAPI 开关和调试模式 |
| `password` | 密码登录开关与最小密码长度 |
| `tha` | 代理认证头登录（enabled / headerName / trustedCIDRs；trustedCIDRs 为空时默认填充本机回环地址） |
| `oidc` | OIDC 认证（enabled / issuerUrl / clientId / clientSecret / redirectUrl / usernameClaim / scopes / loginLabel） |
| `passkey` | WebAuthn/Passkey 认证（enabled / rpName / rpId / rpOrigins / timeout） |
| `copilot` | AI 助手模型接入（model / baseUrl / apiKey） |
| `apisix` | APISIX Admin API 地址和密钥 |
| `caddy` | Caddy Admin API 地址 |
| `docker` | Docker 守护进程地址（支持 `tcp://` + TLS 证书连接远程 Docker / Swarm）、容器数据目录、镜像仓库账号 |
| `monitor` | 系统与容器监控的采集间隔（5/15/30/60 秒；未配置或其他值禁用自动采集） |
| `metrics` | 本机与容器 Prometheus 出口开关（`hostEnabled` / `containersEnabled`；缺省 true，false 关闭出口，不影响其他监控或告警） |
| `notify` | Webhook 通道、CPU/内存/磁盘规则，以及容器异常、任务失败、网关证书到期和受管节点离线告警；默认关闭应用故障告警 |
| `marketplace` | 应用市场地址 |
| `portmap` | 中控端口转发（`range` 监听端口范围 / `bind` 监听地址；仅 center 模式使用，修改后重启生效；与 Docker 容器端口映射无关） |
| `links` | 自定义快捷链接（名称、URL、图标） |
| `members` | 用户账号、家目录、Founder 标记、路由权限、Passkey 与 TOTP 信息 |

## 配置重载

isrvd 支持运行时重载配置和服务连接，无需重启进程。

### 重载方式

| 方式 | 说明 |
|------|------|
| etcd 配置变更 | 自动触发，无需手动操作（本进程自身写入不会重复触发）；key 被删除时只告警，继续使用旧配置 |
| `PUT /api/system/config` | 保存后的配置有变化时自动重载，本地文件与 etcd 模式均适用 |
| `kill -HUP <pid>` | 手动触发，适用于本地文件配置场景 |

```bash
kill -HUP $(pgrep isrvd)
```

### 触发行为

1. 取消进行中请求的 context（长连接、SSE、WebSocket 会被中断，即使随后读取配置失败、继续沿用旧配置，已断开的连接也不会恢复）
2. 重新从配置源加载配置；读取或校验失败时拒绝本次重载，继续使用旧配置
3. 重新初始化 registry 客户端连接（APISIX/Caddy/Docker）
4. 重新初始化各业务服务
5. 服务恢复可用后，对应 API 立即生效

重载持有锁期间，新请求不排队，直接返回 `503`，`message` 为 `服务正在重载`，客户端应稍后重试。`allowedOrigins`、`rootDirectory` 随重载生效；`server.listenAddr`、`server.debug`、`portmap` 与部署环境变量需重启。

### 服务不可用时的行为

服务初始化失败时，对应模块的路由仍然注册，但请求时返回 `503 Service Unavailable`：

```json
{
  "success": false,
  "message": "查询 APISIX 路由列表服务不可用"
}
```

## 获取配置

```bash
isrvd_get "/system/config"
```

> 返回完整系统配置（含基础设施地址等敏感拓扑信息），需要 `GET /api/system/config` 路由权限。前端启动所需的最小配置（如 `marketplaceUrl`）在 `GET /api/overview/bootstrap` 的 `config` 段。

默认读取当前运行配置。用 `isrvd_get "/system/config?stored=true"` 读取已保存的配置，数据位于返回结果的 `config` 字段（即 HTTP 响应的 `payload.config`）。尚未保存配置时返回运行配置作为表单初值。两种读取使用相同的权限和脱敏规则。

配置页的「重新加载」读取已保存的配置，有未保存修改时先确认再替换草稿。保存成功后也从已保存配置更新表单。若重载期间暂时无法回读，页面会保留本次提交的内容并提供重试。旧版受管机只返回运行配置，配置页保留提交内容，待用户重新加载时再读取。

查看与修改为两项独立授权：

| 路由权限 | 能力 |
|----------|------|
| `GET /api/system/config` | 查看配置（只读） |
| `PUT /api/system/config` | 保存配置；需依赖 `GET` 读取当前值，建议两项一并授予 |

| 字段 | 类型 | 说明 |
|------|------|------|
| server | object | `{title, listenAddr, rootDirectory, maxUploadSize, allowedOrigins, jwtExpiration, debug, openapi}`（jwtSecret 不返回；`openapi` 默认 false） |
| password | object | `{disabled, minLength}`（密码登录配置；`minLength` 默认 6） |
| passkey | object | `{enabled, rpName, rpId, rpOrigins, timeout}` |
| oidc | object | `{enabled, issuerUrl, clientId, redirectUrl, usernameClaim, scopes, loginLabel}`（clientSecret 不返回） |
| tha | object | `{enabled, headerName, trustedCIDRs}`（代理 Header 登录配置） |
| copilot | object | `{model, baseUrl}`（apiKey 不返回） |
| notify | object | `{webhooks, rules, events}`；资源与应用故障告警共用已配置 Webhook |
| apisix | object | `{adminUrl, metricsUrl}`（adminKey 不返回；`metricsUrl` 可选，含用户信息、查询参数、片段或解析失败时整值返回 `******`；留空时取 `adminUrl` 主机的 `:9091/apisix/prometheus/metrics`） |
| caddy | object | `{adminUrl}` |
| docker | object | `{host, tls, containerRoot, registries}`（registry password、`tls.key` 不返回） |
| monitor | object | `{interval}`（采集间隔秒数；未配置或非法值表示禁用） |
| metrics | object | `{hostEnabled, containersEnabled}`（缺省 true；false 关闭对应 Prometheus 出口，不影响其他监控或告警） |
| marketplace | object | `{url}` |
| portmap | object | `{range, bind}`（中控端口转发，与 Docker 容器端口映射无关；`range` 如 `20000-20999` 或单个端口，留空不启用；`bind` 为监听 IP，默认 `127.0.0.1`；格式无效返回 `400`；修改后重启生效） |
| links | object[] | `{label, url, icon}` |

> `schema.version` 由系统维护，`members` 请用 `/account/member*` 接口，二者均不通过该接口返回或修改。

多节点部署参数通过 `ISRVD_MODE`、`ISRVD_CENTER_URL`、`ISRVD_NODE_TOKEN` 环境变量设置，详见 [集中管理](../../multi-node.md)。受管模式要求 `tha.enabled=false`。

## 更新配置

> 先通过 `isrvd_get "/system/config"` 获取当前值，按需修改后提交，不要硬编码配置内容。需要 `PUT /api/system/config` 路由权限。

```bash
isrvd_put "/system/config" '<CURRENT_CONFIG_WITH_CHANGES>'
```

请求中为 `null` 或未提交的分区跳过更新。网页一次保存本地草稿中有变化的全部分区。保存成功后自动触发配置重载。

`metrics` 分区只含 `hostEnabled`、`containersEnabled`，缺省 true，显式 false 会持久化；旧 `hostDisabled`、`containersDisabled` 仍可读取并取反（新字段优先），加载后自动回写为 enabled 格式，响应只返回新字段。APISIX / Caddy 指标开关是网关自身状态，位于 `PUT /api/apisix/metrics`、`PUT /api/caddy/metrics`，不写入本分区；详见 [监控指标 API](../metrics.md)。

「监控指标」页面（`/system/config/metrics`）管理 `monitor` 采集间隔与 `metrics` 出口开关，两者互不联动且随「保存」提交；`notify` 通道和告警规则位于「告警通知」页面（`/system/config/alert`）。仅有指标权限的用户不读取系统配置，也不能查看或修改采集与告警配置。

状态码：`400` 为通知阈值非法、`portmap` 非法、受管模式启用 THA；其余校验失败（JWT 密钥不足 32 字符、Passkey/OIDC/Docker TLS 校验失败、关闭全部登录方式、配置正在重载等）返回 `500`。

配置说明：

- `server.dataSecret` 是 SSH 凭据和 Compose 历史的持久加密密钥，仅存于 YAML/etcd 配置，API 不返回且不能修改。首次升级已有有效 `jwtSecret` 的旧配置时沿用其值保证历史密文兼容；首次全新安装生成独立随机密钥。之后修改 `jwtSecret` 不影响历史数据，但旧安装的 `dataSecret` 仍与原 JWT 密钥相同；如果原 JWT 密钥已经泄露，仅轮换 JWT 不能保护旧密文，还需另行完成数据重新加密。必须备份 `dataSecret`，首次升级**前**已因 JWT 密钥变更而无法解密的数据不能自动修复。
- `clientSecret`、`jwtSecret`、`apiKey`、`adminKey`、`docker.registries[].password` 等敏感字段不会通过 GET 返回；PUT 时为空（含纯空白）表示保留原值，非空值会去除首尾空白后保存；`docker.registries[].password` 按 `url` + `username` 匹配旧仓库后保留原值。启动加载配置时，如果 `server.jwtSecret` 为空或仍为示例值 `your-jwt-secret`，系统会使用密码学安全随机源生成 256 位密钥并写回配置；手动配置的密钥不能少于 32 个字符。
- `apisix.metricsUrl` 的脱敏占位符 `******` 回传时保留完整原地址及凭据；提交新地址替换原值，提交空字符串恢复默认抓取地址，不将旧凭据迁移到新目标。审计请求体中的该字段始终脱敏。
- 默认值：`server.listenAddr` `:8080`，`server.rootDirectory` `.`（相对路径转为绝对路径），`server.maxUploadSize` 104857600（0 取默认），`server.jwtExpiration` 86400 秒，`passkey.timeout` 60000 毫秒（0 取默认），`oidc.scopes` 空时为 `[openid profile email]` 且始终包含 `openid`。`docker.containerRoot`、`members[].homeDirectory` 相对 `rootDirectory` 解析，落盘时还原为 `./` 相对路径。
- 连接远程 Docker / Swarm：`docker.host` 设为 `tcp://host:2376`，并设置 `docker.tls`：`enabled`（启用 TLS）、`skipVerify`（跳过服务端证书校验，仅限测试）、`ca`（校验服务端证书的 CA，PEM 文本；留空使用系统根证书）、`cert` 与 `key`（客户端证书与私钥，PEM 文本，daemon 启用 `--tlsverify` 时需要，必须成对）。证书以 PEM 文本保存在配置中，使用 etcd 配置后端时随配置同步。`tls.enabled` 为 `true` 时 `host` 必须是 `tcp://` 地址，保存时会校验 PEM 与证书私钥是否匹配，不通过则拒绝保存。`tls.key` 留空保留原值；清空 `tls.cert` 会同时清除私钥；请求未携带 `tls` 时沿用原值。Swarm 复用 Docker 连接，目标节点需为 manager。
- 远程 Docker 的判定与限制：`docker.host`（留空时取环境变量 `DOCKER_HOST`）为 `tcp://` 且主机不是回环地址（`127.0.0.0/8`、`::1`、`localhost`）即视为远程，与是否启用 TLS 无关；`tcp://docker-proxy:2375` 这类无法确认是否同机的别名也按远程处理。远程时：
  - 「自身容器」只按 isrvd 所在容器的 overlay 存储路径（`upperdir`/`workdir`）识别；存储驱动不是 overlay 时自身保护不生效。
  - bind 挂载源与 `containerRoot` 是远程主机上的路径，isrvd 不检查其是否存在；通过界面创建容器时，相对路径会按 `containerRoot/<容器名>/` 展开并在远程主机上创建。
  - Compose（Docker 模式）的项目文件、`.env` 与部署历史保存在 isrvd 所在主机；相对 bind 路径会展开为 isrvd 本机的绝对路径后原样交给远程 daemon，该路径在远程主机上不存在时容器启动会失败（`bind source path does not exist`），请改用远程主机上真实存在的绝对路径或命名卷。
- `password.disabled` 设为 `true` 后，`POST /api/account/login` 直接拒绝请求，仅保留 Passkey、OIDC 或代理 Header（THA）登录；后端拒绝关闭全部登录方式的配置。
- `password.minLength` 密码最小长度，默认 6；创建成员和修改密码时后端同步校验。
- 启用 OIDC 时，`oidc.issuerUrl` 和 `oidc.redirectUrl` 必须为合法的 HTTP(S) 绝对地址且不含用户信息，`oidc.clientId` 不能为空。启用 Passkey 时，`rpName` 非空，`rpId` 不含 `://`、`/`、`?`、`#`，`rpOrigins` 非空且每项为 HTTP(S) 绝对地址。
- `oidc.usernameClaim` 默认 `sub`；如改用 `email`，需确保 IdP 已验证邮箱且本地 `members.username` 与邮箱完全一致。`oidc.loginLabel` 为登录按钮名称，留空使用默认文案。
- 代理 Header 登录的 `tha.headerName` 默认 `X-Username`；该 Header 的值会作为登录用户名，且必须存在于 `members.username`。`tha.trustedCIDRs` 限制允许传入 Header 的代理来源 IP/CIDR（如 `["10.0.0.0/8"]`）；未配置时默认为本机回环地址（`127.0.0.1/32`、`::1/128`），代理不在本机时需显式配置。中控模式允许 THA，网关按连接对端 IP 校验，拒绝不可信身份头。
- `monitor.interval` 合法值为 `5/15/30/60`（秒），未配置或其他值（含 `0`、负数）均禁用自动采集；禁用会停止资源阈值告警，不影响独立的应用故障检测。
- `notify.webhooks` 的每项包含 `name`、`url`、`template`；`template` 留空时由后端生成标准事件 JSON。JSON 模板可使用 `json` 函数安全编码动态字段，例如 `{{.Title | json}}`（函数输出已包含 JSON 引号）。`notify.rules` 的每项包含 `metric`（`cpu`、`memory` 或 `disk`）、`threshold` 和 `duration`（连续超阈值的采集次数，≤0 按 1）；`threshold ≤ 0` 或未知 `metric` 的规则被忽略，不报 400。Webhook 地址视为管理员可信配置：保存时不校验，发送时无效地址（须为带主机名的 `http/https` 地址）被跳过并记录日志，不限制内网或回环目标，单次请求超时 10 秒，接收端返回非 2xx 时记录投递失败及状态码。
- 通知最多同时处理 128 个发送任务；同事件键、同 URL 串行发送，仅保留一个最新待发状态，替换过时待发状态或达到总容量时记录日志，不阻塞监控采集。任务失败通知不按事件键串行，但同样受总容量限制。排队和发送共用 30 秒有效期；退出宽限期结束后取消在途与待发通知。这是尽力投递，不保证过载时保留所有状态转换。

## 浏览器标签页标题

`server.title` 去除首尾空白后保存，留空使用 `iSrvd`，适用于访问该实例的所有用户。登录后按「实例标题 · 当前页面名称」显示；登录页和未登录的启动接口始终为 `iSrvd`；集中管理的节点视角使用该节点自己的标题，便于区分机器。

```bash
# 保留 server 分区中的其他配置，只修改标题（需 jq）
server_config=$(isrvd_get "/system/config" | jq -c '.server | .title = "内网机器" | {server: .}')
isrvd_put "/system/config" "$server_config"
```

## 应用故障告警

在「系统配置 → 监控告警」中配置通知通道，并开启所需故障类型。旧配置没有 `events` 时，各类应用故障及节点告警均默认关闭。

下面的例子保留当前通知通道和资源规则，只修改应用故障配置（需要 jq）：

```bash
notify_config=$(isrvd_get "/system/config" | jq -c '.notify | .events = {
  containerEnabled: true,
  cronEnabled: true,
  certificateEnabled: true,
  nodeEnabled: true,
  restartThreshold: 3,
  restartWindow: 300,
  certificateDays: 14
} | {notify: .}')
isrvd_put "/system/config" "$notify_config"
```

`notify.events` 请求和响应字段相同：

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| containerEnabled | boolean | false | 容器异常退出、OOM、dead、restarting、unhealthy 及频繁重启 |
| cronEnabled | boolean | false | 计划任务每次执行失败时通知，包括超时 |
| certificateEnabled | boolean | false | APISIX 和 Caddy 可读取证书的到期检测 |
| nodeEnabled | boolean | false | 仅中控模式：受管节点持续离线告警及恢复通知 |
| restartThreshold | integer | 3 | 窗口内 Docker 自动重启次数阈值；1–10000 |
| restartWindow | integer | 300 | 重启统计窗口（秒）；30–86400 |
| certificateDays | integer | 14 | 证书到期前提醒天数；1–3650 |

数值为 0 时使用默认值；API 提交负数或超出范围的非零数值返回 400，且不会应用本次配置更新。

检测与通知行为：

- 容器每 30 秒轮询一次，首次检测将当前 `RestartCount` 作为基线，后续累计窗口内增量；一次持续异常只通知一次，异常原因变化不重复通知。恢复运行且健康、重启次数低于阈值后发送恢复通知，再次异常才重新告警。配置重载保留进程内去重状态；切换 Docker、Caddy 或 APISIX 地址时仅清除对应数据源状态。进程重启会重新建立基线。
- 正常退出、暂停及常见停止信号的退出码 137/143 不触发异常退出告警；但 OOM 退出仍告警。为容器添加 `isrvd.notify.ignore=true` 标签可排除检测。查询失败不发送恢复通知，删除或排除容器也不视为恢复。自动删除或在两次轮询之间完成的短时容器可能无法检测。
- 容器列表查询最多等待 20 秒；每个容器详情查询单独限时 5 秒，失败或超时后继续检查后续容器；配置重载或退出会取消当前查询。
- 计划任务每次失败执行发送一次 `cron.failed`，只包含任务名称、ID、执行 ID 和耗时；通知不含脚本、输出或原始错误，具体失败原因在执行历史查看。
- 证书每小时检测一次，同一证书每天最多提醒一次；检测到过期时升级为 `critical`，不受每日提醒间隔限制。续期至提前提醒窗口之外发送恢复通知。APISIX 禁用证书不检测；Caddy 自动签发策略没有有效期，实际文件、PEM 或缓存证书可读取时才检测。仍在返回列表中但读取或解析失败的证书保留原告警状态，不视为删除或恢复。
- Caddy 文件及缓存证书按文件路径关联告警；PEM 证书按证书内容中的真实主题、SAN 集合和公钥算法关联，同身份的多张 PEM 证书按最早到期的一张判断。存在无法解析的 Caddy PEM 时暂缓所有 Caddy PEM 恢复通知，待均可解析或移除后再判断；文件、缓存及 APISIX 证书的恢复不受此限制。删除或调整列表顺序不视为证书恢复；通知中的 `certificateId` 使用当前 API 列表的 Key。
- 中控受管节点告警独立于监控历史采集：启用 `nodeEnabled` 且配置了 Webhook 后，每 30 秒读取一次节点隧道状态；已激活节点连续离线约 90–120 秒时发送 `node.alert`，恢复在线时发送一次 `node.recover`。待接入、已吊销、已删除节点不产生告警或恢复。重启中控、关闭告警或移除全部通道后重新建立观察基线；重启时原本离线的已激活节点会重新计时。此功能仅观察连接在线，不代表业务服务健康。
- 无通知通道时不启动容器和证书探测，受管节点也不发送告警；关闭监控历史采集不影响这些故障告警。
- 同一资源、节点或证书发往同一 Webhook 地址的告警与恢复事件按发送顺序处理。其他告警和地址可以并行。
- SIGTERM/SIGINT 到达时立即启动 HTTP 关闭，宽限期为 5 秒。配置重载、升级任务和计划任务（含重载前旧实例）进入后续服务关闭阶段，最多等待 5 秒，超时后取消计划任务并额外等待 1 秒清理。最后最多等待 Webhook 1 秒。

所有通知沿用标准事件字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| source | string | 固定为 `isrvd` |
| event | string | 下表中的事件类型 |
| level | string | `warning`、`critical` 或 `info` |
| title | string | 标题 |
| message | string | 故障摘要 |
| timestamp | integer | 发送时间（Unix 秒） |
| data | object | 按事件类型携带的数据 |

| event | data 字段 | 说明 |
|-------|-----------|------|
| container.alert / container.recover | containerId、containerName、reason、exitCode、restartCount、restartWindow | reason 为 restarting/dead/oom/exit/unhealthy/restarts；恢复为空，restartCount 为窗口内检测到的增量 |
| cron.failed | jobId、jobName、runId、duration | duration 为毫秒 |
| node.alert / node.recover | nodeId、nodeName、offlineSeconds | 中控受管节点离线及恢复；离线秒数为本轮检测的估算值，恢复时为 info |
| certificate.alert / certificate.recover | provider、certificateId、subject、notAfter、daysRemaining | provider 为 caddy/apisix，notAfter 为 UTC RFC3339；证书过期时为 critical，恢复为 info |
| resource.alert / resource.recover | metric、value、threshold、unit | metric 为 cpu/memory/disk，unit 固定为 `%`；当前值 ≥90 为 critical，否则 warning，恢复为 info |

---

## 审计日志

审计服务启动或重载时，从当前目录最近 30 天的日志补齐最近 100 条记录，查询按时间倒序返回。

审计策略由后端路由的 `Audit` 字段控制：`0` 按 Method 审计（非 GET 与 WebSocket 记录），`-1` 忽略，`1` 强制记录。未显式配置时默认为 `0`。文件管理读取类接口（`GET /filer/files`、`/filer/file`、`/filer/download`）为 GET 请求，按默认策略不记录审计日志。

请求体中的密码、SSH 私钥（`privateKey`）、文件/脚本内容（`content`、`envContent`）及其他密钥字段会脱敏为 `[REDACTED]`：JSON 与表单体按字段脱敏；multipart、二进制、其他类型分别记为 `[Multipart Omitted]`、`[Binary Omitted]`、`[Body Omitted]`，非法 JSON 记为 `[Invalid JSON Omitted]`；JSON/表单体超过 4 MiB 记为 `[Body Too Large Omitted]`。单个字符串值超过 512 字节时只保留前部并追加 `…[Truncated]`，JSON 数组超过 50 个元素时保留最后 50 个，并在开头加入 `"[Truncated]"` 元素，因此 JSON 输出结构完整且始终合法；结果仍超过 64 KiB 时，JSON 会逐级把字符串上限收紧到 128、32、8 字节，仍超限才记为 `{"[Truncated]":true}`；表单体超过 64 KiB 时截断并追加 ` [Truncated]`。敏感字段仍为 `[REDACTED]`。请求 URI 中的 `token` 等敏感查询参数统一替换为 `******`，包括 WebSocket 终端请求。脱敏只影响审计记录，不修改实际请求，已有日志不会改写。

```bash
isrvd_get "/system/audit/logs?limit=20"
isrvd_get "/system/audit/logs?username=<USERNAME>"
```

| 参数 | 说明 |
|------|------|
| username | 可选，按操作人过滤 |
| limit | 默认 100，最大有效值 100（非数字或 ≤0 按 100） |

接口只返回内存中最近 100 条记录（时间倒序）；持久化文件位于 `server.rootDirectory/audit/YYYY-MM-DD.jsonl`。

| 字段 | 类型 | 说明 |
|------|------|------|
| timestamp | string | 时间戳 |
| username | string | 操作用户 |
| method | string | HTTP 方法；WebSocket 请求记为 `WS` |
| uri | string | 请求路径及查询参数，敏感参数值已脱敏 |
| body | string | 请求体，敏感字段已脱敏 |
| ip | string | 来源 IP（连接对端，不采信 `X-Forwarded-For` 等转发头；反向代理后为代理 IP） |
| statusCode | number | 响应状态码 |
| success | boolean | 是否成功 |
| duration | number | 耗时（ms） |
