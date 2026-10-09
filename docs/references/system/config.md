# 系统配置与审计日志 API

## 配置存储

`CONFIG_PATH` 指定配置位置，未设置时读取 `./config.yml`。

```bash
CONFIG_PATH=/data/conf/isrvd.yml ./isrvd
CONFIG_PATH="etcd://user:pass@127.0.0.1:2379/isrvd/config?fallback=/data/conf/isrvd.yml" ./isrvd
# 多节点 + 凭据通过环境变量传入
ETCD_USERNAME=user ETCD_PASSWORD=pass CONFIG_PATH="etcd://h1:2379,h2:2379,h3:2379/isrvd/config?scheme=https&timeout=5s" ./isrvd
```

说明：

- etcd value 使用同款 YAML；`CONFIG_PATH` 中的 path 是完整 etcd key，必须显式提供；系统配置推荐 key 为 `/isrvd/config`
- 支持多个节点，用逗号分隔 `host:port`
- 可选查询参数：`scheme`（默认 `http`，TLS 时用 `https`）、`timeout`（Go 时长格式，默认 `5s`）、`fallback`（本地 YAML 文件路径，仅在 etcd key 不存在时用于初始化）
- `ETCD_USERNAME` / `ETCD_PASSWORD` 环境变量优先于 URI 中的账号密码，生产环境建议用环境变量传入凭据；URI 中的特殊字符需 URL encode

### 业务数据存储

计划任务（`cron.yml`）、SSH 主机（`webssh-host.yml`）与 SSH 凭据（`webssh-cred.yml`）跟随配置存储后端：

| 配置后端 | 业务数据位置 |
|----------|--------------|
| 本地文件 | `server.rootDirectory/<文件名>` |
| etcd | `<配置 key>/<文件名>`（如 `/isrvd/config/cron.yml`）；etcd 中不存在时读取 `rootDirectory` 下同名文件并写入 etcd |

计划任务执行日志、审计日志、监控历史等日志类数据仍写本地磁盘。

业务数据存储由 cstore 从配置存储派生：本地后端使用当前 `rootDirectory`，etcd 后端复用配置连接，key 为 `<配置 key>/<文件名>`，缺失时从当前 `rootDirectory` 的同名文件迁移。修改 `rootDirectory` 并重载后，新建服务使用新的本地目录或 fallback 目录；已存在的 etcd 数据不被覆盖。

### 保存一致性

系统配置、计划任务和 SSH 主机/凭据通过 cstore 检查最近一次成功加载或保存的原始内容。保存前检查不触发 fallback 迁移；发现内容已变更（包括被删除）时拒绝覆盖；请重新加载后重试。加载失败的计划任务存储禁止写入，避免空任务表覆盖原有数据。

本地文件采用同目录临时文件和原子替换，保留已有文件权限，新文件使用 `0600`；文件后端仅协调同目录 Store 的进程内更新。etcd 通过事务比较 key 的存在性和原始值，再条件写入，支持跨实例防覆盖；fallback 也只在 key 缺失时原子创建，竞争失败后读取已有数据。值比较不检测内容变更后又恢复原值的 ABA 情况，也不提供浏览器旧草稿版本校验或分布式任务执行锁；etcd 超时不代表服务端一定未提交。

etcd 空字符串值视为已存在，不触发 fallback。使用 libgo v0.21.0 的 `Lookup`、`CompareAndPut` 和 Watch 的 `SYNC` 事件，分别实现存在性查询、条件写入和连接状态补偿。

配置监听在首次连接、重连后重新读取最新状态，读取失败会重试，相同状态不重复触发重载。该机制补偿连接间隙的最终状态，不重放所有历史事件；本地文件仍通过 SIGHUP 手动重载，cron、webssh 的外部数据更新仍需重载服务。

## 配置段说明

配置文件（`config.yml` 或 etcd value）的顶层配置段：

| 配置段 | 说明 |
| -------- | ------ |
| `schema` | 配置结构版本，用于自动迁移 |
| `server` | 监听地址、数据根目录、上传限制、CORS、JWT 密钥/有效期、OpenAPI 开关和调试模式 |
| `password` | 密码登录开关与最小密码长度 |
| `tha` | 代理认证头登录（enabled / headerName / trustedCIDRs；trustedCIDRs 为空时默认填充本机回环地址） |
| `oidc` | OIDC 认证（enabled / issuerUrl / clientId / clientSecret / redirectUrl / usernameClaim / scopes / loginLabel） |
| `passkey` | WebAuthn/Passkey 认证（enabled / rpName / rpId / rpOrigins / timeout） |
| `copilot` | AI 助手模型接入（model / baseUrl / apiKey） |
| `apisix` | APISIX Admin API 地址和密钥 |
| `caddy` | Caddy Admin API 地址 |
| `docker` | Docker 守护进程地址（支持 `tcp://` + TLS 证书连接远程 Docker / Swarm）、容器数据目录、镜像仓库账号 |
| `monitor` | 系统与容器监控的采集间隔（5/15/30/60 秒，其他值禁用自动采集） |
| `notify` | Webhook 通道、CPU/内存/磁盘规则，以及容器异常、任务失败和网关证书到期告警；默认关闭应用故障告警 |
| `marketplace` | 应用市场地址 |
| `links` | 自定义快捷链接（名称、URL、图标） |
| `members` | 用户账号、家目录、Founder 标记、路由权限、Passkey 与 TOTP 信息 |

## 配置重载

isrvd 支持运行时重载配置和服务连接，无需重启进程。

### 重载方式

| 方式 | 说明 |
|------|------|
| etcd 配置变更 | 自动触发，无需手动操作（本进程自身写入不会重复触发） |
| `PUT /api/system/config` | 保存后自动触发一次完整重载，本地文件与 etcd 模式均适用 |
| `kill -HUP <pid>` | 手动触发，适用于本地文件配置场景 |

```bash
kill -HUP $(pgrep isrvd)
```

### 触发行为

1. 重新从配置源加载配置；读取或校验失败时拒绝本次重载，继续使用旧配置
2. 取消进行中请求的 context（长连接、SSE、WebSocket 会被中断）
3. 重新初始化 registry 客户端连接（APISIX/Caddy/Docker）
4. 重新初始化各业务服务
5. 服务恢复可用后，对应 API 立即生效

重载持有锁期间，新请求不排队，直接返回 `503`，`message` 为 `服务正在重载`，客户端应稍后重试。

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

> 该接口返回完整系统配置（含基础设施地址等敏感拓扑信息），需要 `GET /api/system/config` 路由权限，普通登录用户无权访问。前端启动所需的最小配置（如应用市场地址 `marketplaceUrl`）已下放至 `GET /api/overview/bootstrap` 的 `config` 段。

查看与修改为两项独立授权：

| 路由权限 | 能力 |
|----------|------|
| `GET /api/system/config` | 查看配置分组（只读）。侧边栏「系统配置」及子菜单可见，表单仅展示不可编辑 |
| `PUT /api/system/config` | 保存配置。工具栏显示「保存配置」按钮，表单可编辑 |

- 仅授 `PUT` 未授 `GET` 时侧边栏同样可见，但表单保存需依赖 `GET` 读取当前值，建议两项一并授予。
- 仅授 `GET` 时页面顶部显示「当前账号仅有查看权限，配置项不可修改」，表单控件整体禁用，且不会触发未保存改动提醒。

| 字段 | 类型 | 说明 |
|------|------|------|
| server | object | `{listenAddr, rootDirectory, maxUploadSize, allowedOrigins, jwtExpiration, debug, openapi}`（jwtSecret 不返回，写入时位于 JWT 配置项；`openapi`=是否对外提供 `/openapi/` 文档，默认 false） |
| password | object | `{disabled, minLength}`（密码登录配置；`minLength` 默认 6） |
| passkey | object | `{enabled, rpName, rpId, rpOrigins, timeout}` |
| oidc | object | `{enabled, issuerUrl, clientId, redirectUrl, usernameClaim, scopes, loginLabel}`（clientSecret 不返回） |
| tha | object | `{enabled, headerName, trustedCIDRs}`（代理 Header 登录配置） |
| copilot | object | `{model, baseUrl}`（apiKey 不返回） |
| notify | object | `{webhooks, rules, events}`；资源与应用故障告警共用已配置 Webhook |
| apisix | object | `{adminUrl}`（adminKey 不返回） |
| caddy | object | `{adminUrl}` |
| docker | object | `{host, tls, containerRoot, registries}`（registry password、`tls.key` 不返回） |
| monitor | object | `{interval}`（采集间隔秒数，非法值表示禁用） |
| marketplace | object | `{url}` |
| links | object[] | `{label, url, icon}` |

> 配置文件 / etcd 中的 `schema.version` 由系统维护（用于版本化迁移），不通过该接口返回或修改。

## 更新配置

> 先通过 `isrvd_get "/system/config"` 获取当前值，按需修改后提交，不要硬编码配置内容。
> 该接口需要 `PUT /api/system/config` 路由权限；成员仅有 `GET` 权限时前端表单为只读，无法保存。

```bash
isrvd_put "/system/config" '<CURRENT_CONFIG_WITH_CHANGES>'
```

支持按分区提交：请求中为 `null` 或未提交的分区跳过更新。保存成功后自动触发配置重载；重载期间提交可能返回"配置正在重载，请稍后重试"。

配置说明：

- `clientSecret`、`jwtSecret`、`apiKey`、`adminKey`、`docker.registries[].password` 等敏感字段不会通过 GET 返回；PUT 时为空（含纯空白）表示保留原值，非空值会去除首尾空白后保存；`docker.registries[].password` 按 `url` + `username` 匹配旧仓库后保留原值。启动加载配置时，如果 `server.jwtSecret` 为空或仍为示例值 `your-jwt-secret`，系统会使用密码学安全随机源生成 256 位密钥并写回配置；手动配置的密钥不能少于 32 个字符。
- 连接远程 Docker / Swarm：`docker.host` 设为 `tcp://host:2376`，并设置 `docker.tls`：`enabled`（启用 TLS）、`skipVerify`（跳过服务端证书校验，仅限测试）、`ca`（校验服务端证书的 CA，PEM 文本；留空使用系统根证书）、`cert` 与 `key`（客户端证书与私钥，PEM 文本，daemon 启用 `--tlsverify` 时需要，必须成对）。证书以 PEM 文本保存在配置中，使用 etcd 配置后端时随配置同步，无需在各节点放置证书文件。`tls.enabled` 为 `true` 时 `host` 必须是 `tcp://` 地址，保存时会校验 PEM 与证书私钥是否匹配，不通过则拒绝保存。`tls.key` 留空保留原值；清空 `tls.cert` 会同时清除私钥；请求未携带 `tls` 时沿用原值。Swarm 复用 Docker 连接，目标节点需为 manager。
- 远程 Docker 的判定与限制：`docker.host`（留空时取环境变量 `DOCKER_HOST`）为 `tcp://` 且主机不是回环地址（`127.0.0.0/8`、`::1`、`localhost`）即视为远程，与是否启用 TLS 无关；`tcp://docker-proxy:2375` 这类无法确认是否同机的别名也按远程处理。远程时：
  - 「自身容器」只按 isrvd 所在容器的 overlay 存储路径（`upperdir`/`workdir`）识别，不再按 IP、主机名匹配（不同主机的默认网桥都是 `172.17.0.x`，会把远程容器误判为自身）；存储驱动不是 overlay 时不识别自身容器，自身保护不生效。
  - bind 挂载源与 `containerRoot` 是远程主机上的路径，isrvd 不检查其是否存在；通过界面创建容器时，相对路径会按 `containerRoot/<容器名>/` 展开并在远程主机上创建。
  - Compose（Docker 模式）的项目文件、`.env` 与部署历史保存在 isrvd 所在主机；相对 bind 路径会展开为 isrvd 本机的绝对路径后原样交给远程 daemon，该路径在远程主机上不存在时容器启动会失败（`bind source path does not exist`），`./nginx.conf` 这类需要随项目一起存在的文件同样无法使用，请改用远程主机上真实存在的绝对路径或命名卷。
- `password.disabled` 设为 `true` 后，密码登录接口（`POST /api/account/login`）将直接拒绝请求，前端也会隐藏密码登录表单，仅保留 Passkey、OIDC 或代理 Header（THA）登录方式。后端会拒绝关闭全部登录方式的配置。
- `password.minLength` 密码最小长度，默认 6；创建成员和修改密码时后端同步校验，前端提示文案也会动态更新。
- 启用 OIDC 时，`oidc.issuerUrl` 和 `oidc.redirectUrl` 必须显式配置为合法的 HTTP(S) 绝对地址，`oidc.clientId` 不能为空。
- `oidc.usernameClaim` 默认 `sub`；如改用 `email`，需确保 IdP 已验证邮箱且本地 `members.username` 与邮箱完全一致。
- `oidc.loginLabel` 自定义 OIDC 登录按钮显示名称；留空则使用默认文案"使用 OIDC 登录"。
- 代理 Header 登录的 `tha.headerName` 默认 `X-Username`；该 Header 的值会作为登录用户名，且必须存在于 `members.username`。`tha.trustedCIDRs` 限制允许传入 Header 的代理来源 IP/CIDR（如 `["10.0.0.0/8"]`）；未配置时默认为本机回环地址（`127.0.0.1/32`、`::1/128`），代理不在本机时需显式配置。
- `monitor.interval` 合法值为 `5/15/30/60`（秒），其他值（含 `0`、负数）均视为禁用自动采集；保存后自动重载生效；禁用会停止资源阈值告警，不影响独立的应用故障检测。
- `notify.webhooks` 的每项包含 `name`、`url`、`template`；`template` 留空时由后端生成标准事件 JSON，Web 管理界面会显示其结构预览。管理界面也可套用钉钉、飞书、企业微信、Slack、Discord、Microsoft Teams、Google Chat 和 Telegram Bot 请求体预设；选择预设后仍可编辑模板，Telegram Bot 的 `CHAT_ID` 占位符需替换为实际值。JSON 模板可使用 `json` 函数安全编码动态字段，例如 `{{.Title | json}}`（函数输出已包含 JSON 引号）。`notify.rules` 的每项包含 `metric`（`cpu`、`memory` 或 `disk`）、`threshold` 和 `duration`。Webhook 地址视为管理员可信配置，仅校验格式（必须为带主机名的 `http/https` 地址），不限制内网或回环目标，会跟随重定向，单次请求超时 10 秒。

## 应用故障告警

在「系统配置 → 监控告警」中配置通知通道，并开启所需故障类型。旧配置没有 `events` 时三类告警均默认关闭。

下面的例子保留当前通知通道和资源规则，只修改应用故障配置（需要 jq）：

```bash
notify_config=$(isrvd_get "/system/config" | jq -c '.notify | .events = {
  containerEnabled: true,
  cronEnabled: true,
  certificateEnabled: true,
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
| restartThreshold | integer | 3 | 窗口内 Docker 自动重启次数阈值；1–10000 |
| restartWindow | integer | 300 | 重启统计窗口（秒）；30–86400 |
| certificateDays | integer | 14 | 证书到期前提醒天数；1–3650 |

数值为 0 时使用默认值；API 提交负数或超出范围的非零数值返回 400，且不会应用本次配置更新。

检测与通知行为：

- 容器每 30 秒轮询一次，首次检测将当前 `RestartCount` 作为基线，后续累计窗口内增量；一次持续异常只通知一次，异常原因变化不重复通知。恢复运行且健康、重启次数低于阈值后发送恢复通知，再次异常才重新告警。配置重载保留进程内去重状态；切换 Docker、Caddy 或 APISIX 地址时仅清除对应数据源状态，清空网关地址后停止查询旧网关。进程重启会重新建立基线。
- 正常退出、暂停及常见停止信号的退出码 137/143 不触发异常退出告警；但 OOM 退出仍告警。为容器添加 `isrvd.notify.ignore=true` 标签可排除检测。查询失败不发送恢复通知，删除或排除容器也不视为恢复。自动删除或在两次轮询之间完成的短时容器可能无法检测。
- 容器列表查询最多等待 20 秒；每个容器详情查询单独限时 5 秒，失败或超时后继续检查后续容器。大量慢容器会延长单轮检测时间，配置重载或退出会取消当前查询。
- 计划任务每次失败执行发送一次 `cron.failed`，只包含任务名称、ID、执行 ID 和耗时；通知不含脚本、输出或原始错误，具体失败原因在执行历史查看。
- 证书每小时检测一次，同一证书每天最多提醒一次；检测到过期时升级为 `critical`，不受每日提醒间隔限制。续期至提前提醒窗口之外发送恢复通知。APISIX 禁用证书不检测；Caddy 自动签发策略没有有效期，实际文件、PEM 或缓存证书可读取时才检测。仍在返回列表中但读取或解析失败的证书保留原告警状态，不视为删除或恢复。
- Caddy 文件及缓存证书按文件路径关联告警；PEM 证书按证书内容中的真实主题、SAN 集合（DNS、IP、邮箱及 URI）和公钥算法关联，CN 为空时也不受 SAN 顺序影响。同身份的多张 PEM 证书按最早到期的一张判断，避免旧证书告警被另一张新证书反复恢复；移除旧证书后按剩余证书判断恢复。无法解析的 Caddy PEM 不按列表下标匹配旧证书，而是保留未匹配的旧 PEM 告警状态；存在身份不明的 PEM 时暂缓所有 Caddy PEM 恢复通知，待均可解析或移除不可读项后再判断，避免健康证书掩盖仍不可读的旧证书。可读取证书仍正常告警和升级；文件、缓存及 APISIX 证书的恢复不受此限制。删除或调整列表顺序不视为证书恢复；通知中的 `certificateId` 仍使用当前 API 列表的 Key。
- 无通知通道时不启动容器和证书探测；关闭监控历史采集不影响这些故障告警。发送沿用同一套 Webhook 地址格式校验。
- SIGTERM/SIGINT 退出时分段计时：先给 HTTP 请求最多 5 秒；再等待进行中的配置重载与计划任务（含重载前旧实例）最多 5 秒，超时后取消任务并额外等待 1 秒清理；最后给已发出的 Webhook 最多 1 秒。超过期限的执行日志与通知不保证完成。

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
| certificate.alert / certificate.recover | provider、certificateId、subject、notAfter、daysRemaining | provider 为 caddy/apisix，notAfter 为 UTC RFC3339；证书过期时为 critical，恢复为 info |
| resource.alert / resource.recover | metric、value、threshold、unit | metric 为 cpu/memory/disk，unit 固定为 `%`；当前值 ≥90 为 critical，否则 warning，恢复为 info |

---

## 审计日志

审计策略由后端路由的 `Audit` 字段控制：`0` 按 Method 审计（非 GET 与 WebSocket 记录），`-1` 忽略，`1` 强制记录。未显式配置时默认为 `0`。文件管理读取类接口（`GET /filer/files`、`/filer/file`、`/filer/download`）为 GET 请求，按默认策略不记录审计日志。

请求体中的密码、SSH 私钥（`privateKey`）、文件/脚本内容（`content`、`envContent`）及其他密钥字段会脱敏为 `[REDACTED]`；multipart、二进制或非 JSON 请求体以占位符代替，超过 64 KiB 截断。请求 URI 中的 `token` 等敏感查询参数统一替换为 `******`，包括 WebSocket 终端请求。脱敏只影响审计记录，不修改实际请求。已有日志不会自动改写。

```bash
isrvd_get "/system/audit/logs?limit=20"
isrvd_get "/system/audit/logs?username=<USERNAME>"
```

| 参数 | 说明 |
|------|------|
| username | 可选，按操作人过滤 |
| limit | 默认 100 |

接口只返回内存中最近 100 条记录（时间倒序）；持久化文件位于 `server.rootDirectory/audit/YYYY-MM-DD.jsonl`。

| 字段 | 类型 | 说明 |
|------|------|------|
| timestamp | string | 时间戳 |
| username | string | 操作用户 |
| method | string | HTTP 方法；WebSocket 请求记为 `WS` |
| uri | string | 请求路径及查询参数，敏感参数值已脱敏 |
| body | string | 请求体，敏感字段已脱敏 |
| ip | string | 来源 IP |
| statusCode | number | 响应状态码 |
| success | boolean | 是否成功 |
| duration | number | 耗时（ms） |
