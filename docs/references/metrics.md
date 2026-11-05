# 监控指标 API

模块名：`metrics`。始终可用；未配置 Caddy / APISIX Admin 地址时，对应出口返回 404。所有路由需在成员 `permissions` 中逐条授予对应的 `METHOD /api/metrics...` 权限（创始人除外）。

- **指标出口**：`/api/metrics/apisix`、`/api/metrics/caddy` 每次请求直接抓取并原样返回网关指标，不解析、差分、聚合或保存历史；`/api/metrics/host` 从最后保存的主机记录导出指标，`/api/metrics/containers` 从各容器最后保存的记录导出指标。各出口独立授权。
- **范围**：最后保存记录仅用于 `/api/metrics/host`、`/api/metrics/containers` 指标出口；原 `/api/overview/monitor` 和 `/api/docker/container/:id/stats` 监控查看接口、实时采集、历史响应与页面行为不变。
- **开关**：本机与容器指标出口独立控制，默认启用，持久化在 `metrics` 配置分区；禁用只阻止对应出口读取，不影响其他监控、监控日志或告警。Caddy / APISIX 仍通过 Admin API 开关网关指标。
- **系统配置 → 监控指标**（`/system/config/metrics`，旧 `/metrics` 自动重定向）：统一管理监控采集间隔与指标出口，采集和出口开关互不联动；展示出口 URL、复制入口及权限和状态提示。告警规则与 Webhook 通道独立位于「告警通知」（`/system/config/alert`）。本机/容器开关绑定系统配置草稿，需系统配置 GET/PUT 权限，点击「保存」后持久化生效；APISIX/Caddy 开关仍通过原指标 API 立即生效，以网关实际配置为准。仅有指标权限的用户仍可访问该区域，不读取系统配置；网关操作不依赖系统配置写权限。未启用或不可用的模块隐藏来源卡片，关闭的出口保留启用入口。初始化、手动刷新、系统配置保存/重载和网关操作完成后查询状态，不轮询，不请求指标出口或预览指标文本；卸载时取消在途状态查询。桌面与移动端行为一致。抓取、预览、历史与聚合查询由外部监控系统处理。

## 抓取地址

| 网关 | 默认地址 | 说明 |
|------|----------|------|
| Caddy | `<caddy.adminUrl>/metrics` | Caddy Admin 端点 |
| APISIX | `<apisix.adminUrl 的 scheme://host>:9091/apisix/prometheus/metrics` | 可用 `apisix.metricsUrl` 覆盖 |

抓取地址只来自配置，不接受请求参数；抓取不跟随重定向。`apisix.metricsUrl` 含用户信息、查询参数、片段或无法解析时，系统配置查询将整个地址替换为 `******`，保存时回传该占位符保留原地址及凭据；提交完整新地址替换原值，提交空字符串恢复默认地址。配置保存的审计日志始终隐藏 `metricsUrl`。

## 开启指标做了什么

- **Caddy**：为全部 HTTP server 设置 `metrics`。优先写入 `{"per_host": true}`（原始指标带 `host` 标签，供外部监控使用）；仅当配置加载返回 HTTP 400 且明确不识别 `per_host` 字段时回退为 `{}`，兼容 Admin API 的 JSON 错误响应及纯文本响应（包括默认镜像使用的 Caddy 2.8.4），关闭后可再次开启；其他错误直接返回。已存在 `metrics` 的 server 保持不变。关闭时移除。镜像 `docker-caddy` 的默认配置已预置 `"metrics": {}`。
- **APISIX**：在 ID 为 `isrvd-metrics` 的 Global Rule 中启用 `prometheus` 插件（新建时 `prefer_name: true`），保留其他插件。已有规则实际启用了该插件时不重复创建；`_meta.disable: true` 不视为启用，自管规则重新开启时只移除禁用标志，保留原插件参数及其他元数据。关闭时只移除自管规则中的指标插件，规则为空才删除；若其他规则也实际开启了指标，返回 409，需在 APISIX 中关闭，不修改用户规则。

## 原始指标口径

APISIX 与 Caddy 的指标名、标签、单位及计数语义由网关自身定义。isrvd 不转换单位、不求和、不去重，也不将累计计数解释为 QPS 或时间窗口内的请求量。原始文本仅通过各自指标出口提供，不在内置页面预览。

---

## 路由列表

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/metrics/apisix` | APISIX 指标原样透传，指标文本格式（`text/plain; version=0.0.4`，不是统一 JSON 响应） |
| GET | `/api/metrics/caddy` | Caddy 指标原样透传，指标文本格式 |
| GET | `/api/metrics/host` | 最后保存的主机监控指标，指标文本格式；禁用或存储未就绪返回 503 |
| GET | `/api/metrics/containers` | 各容器最后保存的资源指标，指标文本格式；禁用或存储未就绪返回 503 |
| GET | `/api/metrics/status` | 本机、容器及已配置网关的指标状态 |
| PUT | `/api/metrics/gateway/:name` | 开启或关闭指标，`name` 为 `host` / `containers` / `caddy` / `apisix`；保留原路径兼容已有调用 |

---

## 接口详情

### 指标抓取

使用「API Token」以 Bearer 方式认证（创建方式见 [account.md](system/account.md)）。令牌所属成员需被授予对应路由的权限，例如只授予 `GET /api/metrics/host` 即只能抓取本机监控。

```bash
curl -H "Authorization: Bearer $TOKEN" "$API_ROOT/metrics/host"
```

抓取配置示例（`scrape_configs` 语法）：

```yaml
scrape_configs:
  - job_name: isrvd-caddy
    metrics_path: /api/metrics/caddy
    authorization: { credentials: <API Token> }
    static_configs: [{ targets: ["isrvd.example.com:8080"] }]
```

`apisix`、`host`、`containers` 同理，分别配置为独立 job。网关未配置返回 404，抓取失败返回 502（抓取端记为目标不可用）。

本机和容器出口未配置开关时默认启用。禁用时在读取记录前返回 503 和统一 JSON 错误 `指标出口已禁用`，不输出指标文本；开关只控制出口读取，不影响后台采集、历史保存或告警。开关与成员抓取权限相互独立。

`/api/metrics/host` 每次请求直接读取最后保存的主机记录，不重新采集。记录的新鲜度由 `monitor.interval` 和存储刷盘决定；后台采集关闭时仅能读取既有记录。尚无保存记录时返回 200 和空文本，存储未就绪返回 503，读取或解析失败返回 502。其余指标均为该记录中的采样值，`isrvd_host_cpu_usage_percent` 为各逻辑核使用率的算术平均，反映采集时刻而非区间平均。出口不输出逐样本时间戳，抓取端可用 `time() - isrvd_host_last_collect_timestamp_seconds` 判断数据是否过期。输出指标（`_total` 为 counter，其余为 gauge）：

| 指标 | 标签 |
|------|------|
| `isrvd_host_uptime_seconds`、`isrvd_host_cpu_usage_percent` | |
| `isrvd_host_last_collect_timestamp_seconds` | 最后保存记录的采集时间（Unix 秒） |
| `isrvd_host_memory_total_bytes`、`isrvd_host_memory_used_bytes` | |
| `isrvd_host_swap_total_bytes`、`isrvd_host_swap_used_bytes` | |
| `isrvd_host_filesystem_size_bytes`、`isrvd_host_filesystem_used_bytes` | `device` `mountpoint` `fstype` |
| `isrvd_host_network_receive_bytes_total`、`isrvd_host_network_transmit_bytes_total` | `interface` |
| `isrvd_host_disk_read_bytes_total`、`isrvd_host_disk_written_bytes_total` | `device` |
| `isrvd_host_gpu_utilization_percent`、`_memory_used_bytes`、`_memory_total_bytes`、`_temperature_celsius` | `gpu` `name`（无 GPU 时不输出） |
| `isrvd_go_goroutines`、`isrvd_go_memory_alloc_bytes`（堆已分配字节） | isrvd 进程自身，不属于主机 |

`/api/metrics/containers` 从保留文件中取各完整容器 ID 最后保存的记录，不列举容器，也不请求 Docker Stats / Top。首次抓取恢复保留期内的记录，此后沿按日追加的文件只读取新增完整行，复用各容器最后落盘记录并清理过期记录；等待与读取均响应请求取消。尚无保存记录时返回 200 和空文本；Docker 暂时不可用不阻止读取已有记录。存储未就绪返回 503，读取或解析失败返回 502。超过 `max(3 × monitor.interval, 60)` 秒未更新的容器记录不输出，避免已停止或删除的容器以旧值被当作当前值；`monitor.interval` 不合法（后台采集未启动）时不过滤，请依据 `isrvd_container_last_collect_timestamp_seconds` 判断新鲜度。全部容器样本以 `name` 和完整 `id` 标识，避免重建、重名或短 ID 冲突产生重复序列；原监控记录不保存镜像信息，因此信息指标不含 `image` 标签：

| 指标 | 标签 | 说明 |
|------|------|------|
| `isrvd_container_info` | `name` `id` | 容器信息，值恒为 1 |
| `isrvd_container_last_collect_timestamp_seconds` | `name` `id` | 该容器最后保存记录的采集时间（Unix 秒） |
| `isrvd_container_cpu_usage_seconds_total` | `name` `id` | 累计 CPU 时间（秒），用 `rate()` 得到使用率 |
| `isrvd_container_memory_usage_bytes` | `name` `id` | 内存工作集（cgroup v1 扣除 `total_inactive_file`，v2 扣除 `inactive_file`；不扣除全部 `cache`） |
| `isrvd_container_memory_limit_bytes` | `name` `id` | 内存上限；未设置上限时 Docker 返回宿主机内存总量 |
| `isrvd_container_network_receive_bytes_total`、`isrvd_container_network_transmit_bytes_total` | `name` `id` | 全部网卡累计收发字节 |
| `isrvd_container_disk_read_bytes_total`、`isrvd_container_disk_written_bytes_total` | `name` `id` | 块设备累计读写字节 |
| `isrvd_container_pids` | `name` `id` | 容器内进程数 |

### 查询指标状态

**GET** `/api/metrics/status`

```bash
isrvd_get "/metrics/status"
```

响应 `payload`：

```json
[
  { "name": "host", "available": true, "enabled": true, "scrapable": true, "url": "/api/metrics/host" },
  { "name": "containers", "available": false, "enabled": false, "scrapable": false, "url": "/api/metrics/containers" },
  { "name": "caddy", "available": true, "enabled": true, "scrapable": true, "url": "/api/metrics/caddy" }
]
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `name` | string | 来源标识：`host` / `containers` / `caddy` / `apisix`；本机和容器始终返回，网关仅返回已配置项 |
| `available` | boolean | 本机始终为 true；容器表示 Docker 是否就绪（不影响保存开关）；网关表示管理服务是否可用 |
| `enabled` | boolean | 对应来源是否已开启指标，与采集能力独立；`enabledUnknown` 为 true 时不能据此判断关闭 |
| `enabledUnknown` | boolean? | 仅网关开关查询失败时返回 true；本机、容器、管理服务不可用或仅抓取失败时不返回 |
| `scrapable` | boolean | 本机和容器由 enabled 与 available 推导，不执行采集，不保证实际采集成功；网关实际检查抓取结果 |
| `url` | string | 所有来源均为 isrvd API 出口路径，不返回上游地址、URL 用户信息或查询凭据；节点和子路径部署需按当前 API 根路径生成出口地址 |
| `error` | string? | 按请求语言翻译的最近一次检查失败原因；管理接口失败仅返回通用提示，不泄露上游响应或凭据 |

指标区域等待 bootstrap 初始化完成后查询状态，需具有 `GET /api/metrics/status` 权限。无此权限时不请求状态接口；状态查询失败或开关状态未知时禁止网关开关操作，手动刷新可重试。本机/容器配置以系统配置草稿为准，不依赖指标状态读取权限；仅查看配置的用户不可修改开关。页面不发起指标出口请求，网关抓取检查仅属于服务端状态查询。开启本机/容器出口不会自动开启采集，也不保证记录持续更新；外部监控抓取仍须分别授予各指标出口的权限。

### 开启或关闭指标

**PUT** `/api/metrics/gateway/:name`

```bash
isrvd_put "/metrics/gateway/caddy" '{"enabled":true}'
isrvd_put "/metrics/gateway/host" '{"enabled":false}'
isrvd_put "/metrics/gateway/containers" '{"enabled":false}'
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `enabled` | boolean | `true` 开启，`false` 关闭（必填） |

`name` 仅接受 `host`、`containers`、`caddy`、`apisix`；未知或未配置网关返回 404，`enabled` 缺失或为 null 返回 400。本机和容器开关通过配置 Provider 串行保存，成功后立即生效，不触发本进程服务重载；Docker 不可用时仍允许保存容器开关。配置存储冲突返回 409，其他本地配置保存错误返回 500，失败不发布新开关状态。更新权限统一使用 `PUT /api/metrics/gateway/:name`，可管理全部四类来源，无需系统配置写权限。

对应网关服务不可用返回 503，网关 Admin API 调用失败返回 502。关闭 APISIX 指标时若其他全局规则仍启用该插件，返回 409，不修改任何规则。
