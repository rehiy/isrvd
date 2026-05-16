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
| hasTls | boolean | 是否配置了 TLS app |
| available | boolean | Admin API 是否可达 |

## 全局选项

### 查询全局选项

```bash
isrvd_get "/caddy/global"
```

| 字段 | 类型 | 说明 |
|------|------|------|
| adminListen | string | Admin API 监听地址，例如 `127.0.0.1:2019` |
| adminDisabled | boolean | 是否禁用 Admin API |
| logLevel | string | 全局日志级别：`DEBUG` / `INFO` / `WARN` / `ERROR` |
| httpPort | number | HTTP 监听端口，默认 80 |
| httpsPort | number | HTTPS 监听端口，默认 443 |
| gracePeriod | string | 优雅关闭等待时间，例如 `10s` |

### 更新全局选项

```bash
isrvd_put "/caddy/global" '{"logLevel":"WARN","gracePeriod":"10s"}'
```

请求体字段同上，**只传需要修改的字段**，数值为 0 / 空字符串 / false 的字段不会覆盖现有值。  
保存后通过 `POST /load` 整体原子替换 Caddy 运行配置，立即生效（Admin 监听地址变更需重启 Caddy）。

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

### 调整日志级别与关闭等待时间

```bash
isrvd_put "/caddy/global" '{"logLevel":"WARN","gracePeriod":"10s"}'
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
