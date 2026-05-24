# Caddy 概览、全局选项与原始配置 API

## 概览信息

```bash
isrvd_get "/caddy/info"
```

| 字段 | 类型 | 说明 |
|------|------|------|
| adminUrl | string | Caddy Admin API 地址（来自系统配置） |
| servers | number | server 总数 |
| routes | number | 所有 server 路由总数 |
| certs | number | TLS 证书总数，包含配置证书与 Caddy 运行时已签发缓存证书 |
| hasTls | boolean | 是否配置了 TLS app |
| available | boolean | Admin API 是否可达 |

## 全局选项

### 查询全局选项

```bash
isrvd_get "/caddy/global"
```

| 字段 | 类型 | 说明 |
|------|------|------|
| logLevel | string | 全局日志级别：`DEBUG` / `INFO` / `WARN` / `ERROR` |
| persistConfig | boolean | 是否持久化配置到磁盘（`admin.config.persist`） |
| storageModule | string | 证书存储模块名，例如 `file_system`，留空使用默认 |
| storageRoot | string | 存储根路径（`file_system` 模块的 `root` 字段） |
| email | string | ACME 注册邮箱 |
| acmeCA | string | 自定义 ACME 目录 URL，留空使用 Let's Encrypt |
| localCerts | boolean | 使用本地自签证书（`internal` issuer），不走 ACME |
| onDemandTLS | boolean | 启用 on_demand TLS，连接时动态申请证书（同时设置全局 `permission` 和默认策略的 `on_demand: true`） |
| onDemandAsk | string | ask 鉴权端点 URL，Caddy 申请证书前向此发 GET 请求，返回 2xx 才允许；Caddy v2.8+ 必须配置，留空则不设（仅测试环境） |
| autoHttpsDisable | boolean | 全局禁用自动 HTTPS（`apps.http.automatic_https.disable`） |
| autoHttpsDisableRedirects | boolean | 禁用 HTTP→HTTPS 自动跳转（`apps.http.automatic_https.disable_redirects`） |

### 更新全局选项

```bash
isrvd_put "/caddy/global" '{"email":"you@example.com","logLevel":"WARN"}'
```

## 获取完整配置

```bash
isrvd_get "/caddy/config"
```

返回值是 Caddy 完整 JSON 配置（结构与 `GET /config/` 返回一致），可作为整体替换的输入。

> **`caddy` 镜像说明**：默认配置模板位于 `build/docker-caddy/caddy.json`，首次启动会复制并持久化到 `/data/conf/caddy.json`；Caddy 运行数据（自动签发证书、autosave 等）持久化在 `/data/caddy/`。
> 默认模板只监听 `:80`，并设置 `automatic_https.disable_redirects=true`，避免没有证书时自动跳转 HTTPS 导致首次访问失败。
> 顶层字段未建模到强类型时（如 `apps.layer4`、自定义 app）会通过 `Extras` 透传，不会丢失。

## 整体替换配置

```bash
isrvd_post "/caddy/config" "$(jq -n '{
  config: {
    admin: {listen: "127.0.0.1:2019"},
    apps: {
      http: {
        servers: {
          srv0: {
            listen: [":80"],
            automatic_https: {disable_redirects: true},
            routes: []
          }
        }
      }
    }
  }
}')"
```

请求体 `{ config: <caddy-json> }`，等价于 `POST /load`，**整体原子替换**当前运行配置。

> ⚠️ 这是危险操作，会影响所有正在运行的路由；推荐先用 `isrvd_get "/caddy/config"` 备份再修改。

## 典型工作流

### 启用 HTTPS 自动签发

```bash
# 设置 ACME 邮箱（必填）
isrvd_put "/caddy/global" '{"email":"you@example.com"}'
# 使用本地自签证书（不走 ACME）
isrvd_put "/caddy/global" '{"localCerts":true}'
```

### 调整日志级别

```bash
isrvd_put "/caddy/global" '{"logLevel":"WARN"}'
```

### 备份并修改部分字段

```bash
# 1. 备份当前配置
CFG=$(isrvd_get "/caddy/config")

# 2. 修改后提交（需要 HTTPS 时再显式加入 :443）
echo "$CFG" | jq '.apps.http.servers.srv0.listen = [":80", ":443"]' \
  | jq '.apps.http.servers.srv0.automatic_https.disable_redirects = false' \
  | jq '{config: .}' \
  | xargs -0 -I{} isrvd_post "/caddy/config" '{}'
```

### 健康检查

```bash
isrvd_get "/caddy/info" '.available'   # 应返回 true
isrvd_get "/caddy/routes"              # 路由数大于 0 表示正常工作
```

### 故障排查

```bash
# 查看 admin api 真实返回
isrvd_get "/caddy/config" | jq '.admin, .apps.http.servers'
```
