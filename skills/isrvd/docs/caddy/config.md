# Caddy 概览与原始配置 API

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

## 获取完整配置

```bash
isrvd_get "/caddy/config"
```

返回值是 Caddy 完整 JSON 配置（结构与 `GET /config/` 返回一致），可作为整体替换的输入。

> `docker-caddy` 镜像中，默认配置模板位于 `build/docker-caddy/caddy.json`，首次启动会复制并持久化到 `/data/conf/caddy.json`；Caddy 运行数据（自动签发证书、autosave 等）持久化在 `/data/caddy/`。
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

### 备份并修改部分字段

```bash
# 1. 备份当前配置
CFG=$(isrvd_get "/caddy/config")

# 2. 修改后提交（需要 HTTPS 时再显式加入 :443）
echo "$CFG" | jq '.apps.http.servers.srv0.listen = [":80", ":443", ":8443"]' \
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
