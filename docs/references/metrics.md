# 监控指标 API

模块名：`metrics`。始终可用；未配置 Caddy / APISIX Admin 地址时，对应出口返回 404，统计页无数据。所有路由需在成员 `permissions` 中逐条授予对应的 `METHOD /api/metrics...` 权限（创始人除外）。

- **指标出口**：`/api/metrics/apisix`、`/api/metrics/caddy` 原样透传网关自身的指标；`/api/metrics/host` 输出 isrvd 所在主机的监控指标，`/api/metrics/containers` 输出运行中容器的资源指标。各出口独立授权。
- **开关**：本机与容器指标出口独立控制，默认启用，持久化在 `metrics` 配置分区；禁用只阻止对应出口采集，不影响其他监控、监控日志或告警。Caddy / APISIX 仍通过 Admin API 开关网关指标。
- **监控指标页**（`/metrics`）：指标来源、Caddy 与 APISIX 统计使用同级分区，不再设置访问统计总标题，分区标题沿用系统配置页规范。来源卡片由统一开关组件作为外层容器，标题区包含来源名称、指标说明、状态及开关；内容区始终显示出口 URL、复制入口及权限提示，开关仅控制指标启停，关闭后不收起详情，避免并列卡片因开关状态产生高度差。无写权限或开关状态未知时只显示状态、不提供开关；错误提示始终可见。拥有任意 `metrics` 路由权限即可进入，各查询与操作仍按具体权限控制。Caddy 与 APISIX 统计同时展示，不使用 tab；各自标题区展示采样说明，页头刷新按钮前的时间范围选择器统一作用于两个网关，桌面与移动端一致；页面每 30 秒自动刷新，隐藏时暂停。支持可靠统计的 APISIX 分区包含汇总（请求总数、平均 QPS、5xx 占比、平均耗时、响应流量及 2xx／3xx／4xx／5xx 请求数）、趋势图及独立搜索的路由排行；排行保留桌面、移动双视图，无采样数据时仅显示该块的空状态。Caddy 分区仅显示原生指标无法可靠去重的限制说明，不展示错误汇总或等待采样提示。
- **访问统计**：后台每 30 秒抓取一次 APISIX 指标，采样时先对原始计数序列差分，再将区间增量保存在内存中（最近 24 小时，isrvd 重启或重载配置后清空），查询只聚合与窗口相交的区间，得到速率、状态码占比和平均耗时。Caddy 原生指标按 handler 计数，无法可靠去重，内置统计返回明确限制说明，不生成请求汇总；原始指标出口和开关不受影响。可靠的 Caddy 整请求统计需要独立的访问日志或整请求观察源。长期历史请使用外部监控系统抓取后存储。

## 抓取地址

| 网关 | 默认地址 | 说明 |
|------|----------|------|
| Caddy | `<caddy.adminUrl>/metrics` | Caddy Admin 端点 |
| APISIX | `<apisix.adminUrl 的 scheme://host>:9091/apisix/prometheus/metrics` | 可用 `apisix.metricsUrl` 覆盖 |

抓取地址只来自配置，不接受请求参数；抓取不跟随重定向。`apisix.metricsUrl` 含用户信息、查询参数、片段或无法解析时，系统配置查询将整个地址替换为 `******`，保存时回传该占位符保留原地址及凭据；提交完整新地址替换原值，提交空字符串恢复默认地址。配置保存的审计日志始终隐藏 `metricsUrl`。

## 开启指标做了什么

- **Caddy**：为全部 HTTP server 设置 `metrics`。优先写入 `{"per_host": true}`（原始指标带 `host` 标签，供外部监控使用）；仅当配置加载返回 HTTP 400 且明确不识别 `per_host` 字段时回退为 `{}`，兼容 Admin API 的 JSON 错误响应及纯文本响应（包括默认镜像使用的 Caddy 2.8.4），关闭后可再次开启；其他错误直接返回。已存在 `metrics` 的 server 保持不变。关闭时移除。镜像 `docker-caddy` 的默认配置已预置 `"metrics": {}`。
- **APISIX**：在 ID 为 `isrvd-metrics` 的 Global Rule 中启用 `prometheus` 插件（新建时 `prefer_name: true`），保留其他插件。已有规则实际启用了该插件时不重复创建；`_meta.disable: true` 不视为启用，自管规则重新开启时只移除禁用标志，保留原插件参数及其他元数据。关闭时只移除自管规则中的指标插件，规则为空才删除；若其他规则也实际开启了指标，返回 409，需在 APISIX 中关闭，不修改用户规则。

## 统计口径

- Caddy：原生指标按 handler 调用记录，同一请求可经过多个 handler（例如 `file_server` 的 `pass_thru` 接 `reverse_proxy`），聚合后没有请求标识可供去重。内置统计通过 `error` 明确告知限制，`series`、`sites` 返回空数组，`summary` 零值不代表真实访问量；原始指标仍可抓取。
- APISIX：站点即路由；耗时取 `apisix_http_latency{type="request"}`，毫秒换算为秒。
- 使用指标名及完整标签标识原始计数序列（不受标签排列顺序影响）；先逐序列差分和处理重置，再聚合为网关、站点的区间增量，最后合并趋势点。新序列从零计入，计数器回退时把当前值当作增量；首次有效采样仅建立基线，不计入网关启动前的历史流量。相邻有效采样间隔超过 5 分钟时，只更新基线，不计入该区间。
- 空白、HTML、语法错误、重复目标序列、负计数、非有限目标计数（`NaN` / `Inf`）及可能使窗口聚合溢出的数值均跳过，不覆盖有效基线；其他通用指标的合法 `NaN` / `Inf` 不影响采样。首次无流量的合法指标响应可建立空基线；已有请求基线后缺失全部目标计数的响应不清空基线。
- 每个网关仅保留一份完整原始序列基线，部分序列短暂缺失时保留其最后有效值最多 5 分钟，避免恢复时重复计入累计量。历史只保存区间增量及起止时间，每个区间保留请求增量最多的 20 个站点。站点重新入榜仍使用完整原始序列求差，不计入历史累计量。最终排行在保留的区间站点中汇总后取前 20 项，未入区间前 20 的站点增量不计入排行（不影响网关总量）；同请求数时按站点名称排序。
- 与查询窗口相交的区间均参与统计；跨边界区间按重叠时长比例估算请求数、响应流量及耗时样本，因此计数可能为小数。速率分母使用窗口内实际覆盖的采样时长，不向最新采样之后或采集中断处外推流量。

---

## 路由列表

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/metrics/apisix` | APISIX 指标原样透传，指标文本格式（`text/plain; version=0.0.4`，不是统一 JSON 响应） |
| GET | `/api/metrics/caddy` | Caddy 指标原样透传，指标文本格式 |
| GET | `/api/metrics/host` | 主机实时监控指标，指标文本格式；禁用返回 503 |
| GET | `/api/metrics/containers` | 运行中容器的实时资源指标，指标文本格式；禁用或 Docker 不可用返回 503 |
| GET | `/api/metrics/status` | 本机、容器及已配置网关的指标状态 |
| GET | `/api/metrics/stats` | 访问统计，查询参数 `since`（秒，默认 3600，限制在 60 ~ 86400） |
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

本机和容器出口未配置开关时默认启用。禁用后，在采集前返回 503 和统一 JSON 错误 `指标出口已禁用`，不输出指标文本；已通过检查的在途采集仍可能完成。开关与成员抓取权限相互独立。

`/api/metrics/host` 启用时每次请求实时采集，不依赖 `monitor.interval`。输出指标（`_total` 为 counter，其余为 gauge）：

| 指标 | 标签 |
|------|------|
| `isrvd_host_uptime_seconds`、`isrvd_host_cpu_usage_percent` | |
| `isrvd_host_memory_total_bytes`、`isrvd_host_memory_used_bytes` | |
| `isrvd_host_swap_total_bytes`、`isrvd_host_swap_used_bytes` | |
| `isrvd_host_filesystem_size_bytes`、`isrvd_host_filesystem_used_bytes` | `device` `mountpoint` `fstype` |
| `isrvd_host_network_receive_bytes_total`、`isrvd_host_network_transmit_bytes_total` | `interface` |
| `isrvd_host_disk_read_bytes_total`、`isrvd_host_disk_written_bytes_total` | `device` |
| `isrvd_host_gpu_utilization_percent`、`_memory_used_bytes`、`_memory_total_bytes`、`_temperature_celsius` | `gpu` `name`（无 GPU 时不输出） |
| `isrvd_go_goroutines`、`isrvd_go_memory_alloc_bytes` | isrvd 进程自身 |

`/api/metrics/containers` 每次请求实时采集全部运行中容器（并发采集，每个容器采样约 1 秒，建议抓取间隔不小于 30 秒）。部分采集成功时跳过失败的容器；非空列表全部采集失败或列表查询失败时返回 502。没有运行中的容器时返回 200 和空指标，Docker 不可用时返回 503。容器以 `name` 标识，随容器重建而变化的 `id`、`image` 只放在 `isrvd_container_info` 上：

| 指标 | 标签 | 说明 |
|------|------|------|
| `isrvd_container_info` | `name` `id` `image` | 容器信息，值恒为 1 |
| `isrvd_container_cpu_usage_seconds_total` | `name` | 累计 CPU 时间（秒），用 `rate()` 得到使用率 |
| `isrvd_container_memory_usage_bytes` | `name` | 内存工作集（cgroup v1 扣除 `total_inactive_file`，v2 扣除 `inactive_file`；不扣除全部 `cache`） |
| `isrvd_container_memory_limit_bytes` | `name` | 内存上限；未设置上限时 Docker 返回宿主机内存总量 |
| `isrvd_container_network_receive_bytes_total`、`isrvd_container_network_transmit_bytes_total` | `name` | 全部网卡累计收发字节 |
| `isrvd_container_disk_read_bytes_total`、`isrvd_container_disk_written_bytes_total` | `name` | 块设备累计读写字节 |
| `isrvd_container_pids` | `name` | 容器内进程数 |

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

统计页首次查询等待 bootstrap 初始化完成，完成后立即加载，无需等待 30 秒轮询。统计查询与状态查询独立授权，仅授予 `GET /api/metrics/stats` 也可查看统计。统计页在状态查询成功时隐藏未配置、管理服务不可用或确认未开启指标的网关段落。无状态查询权限、状态请求失败或开关查询失败时，不阻断已授权的统计展示；开关状态未知时暂禁开关操作，等待刷新恢复。仅指标抓取失败不会把确认关闭的网关重新显示。

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

`name` 仅接受 `host`、`containers`、`caddy`、`apisix`；未知或未配置网关返回 404，`enabled` 缺失或为 null 返回 400。本机和容器开关通过配置 Provider 串行保存，成功后立即生效，不触发本进程服务重载或清空统计；Docker 不可用时仍允许保存容器开关。配置存储冲突返回 409，其他本地配置保存错误返回 500，失败不发布新开关状态。更新权限统一使用 `PUT /api/metrics/gateway/:name`，可管理全部四类来源，无需系统配置写权限。

对应网关服务不可用返回 503，网关 Admin API 调用失败返回 502。关闭 APISIX 指标时若其他全局规则仍启用该插件，返回 409，不修改任何规则。

### 查询访问统计

**GET** `/api/metrics/stats?since=3600`

```bash
isrvd_get "/metrics/stats?since=3600"
```

响应 `payload` 为数组，每个已配置的网关一项；没有采样数据时 `series` 与 `sites` 为空数组。

| 字段 | 类型 | 说明 |
|------|------|------|
| `name` | string | 网关标识 |
| `summary` | object | 窗口汇总 |
| `series` | object[] | 趋势点 |
| `sites` | object[] | APISIX 路由排行，最多 20 项，按请求数降序 |
| `error` | string? | 无法提供可靠访问统计时的已翻译说明；有值时不可把 `summary` 零值当作真实流量 |

`summary`：`requests`（请求总数）、`s2xx` / `s3xx` / `s4xx` / `s5xx`（各状态码类请求数）、`errorRate`（5xx 占比，0 ~ 1）、`rps`（平均请求速率，次/秒）、`avg`（平均耗时，毫秒）、`bytesOut`（响应字节数）。

`series[]`：`ts`（Unix 秒）、`rps`、`s2xx` / `s3xx` / `s4xx` / `s5xx`（速率，次/秒）、`avg`（平均耗时，毫秒）。

`sites[]`：`site`（APISIX 为路由名称）、`requests`、`s4xx`、`s5xx`、`avg`（毫秒）、`bytesOut`。Caddy 的 `error` 有值时仅展示限制说明，不绘制汇总或趋势。
