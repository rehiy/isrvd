# 系统配置与审计日志 API

## 配置存储

`CONFIG_PATH` 指定配置位置，未设置时读取 `./config.yml`。

```bash
CONFIG_PATH=/data/conf/isrvd.yml ./isrvd
CONFIG_PATH="etcd://user:pass@127.0.0.1:2379/isrvd/config?fallback=/data/conf/isrvd.yml" ./isrvd
```

说明：etcd value 使用同款 YAML；`fallback` 仅在 key 不存在时用于初始化。

## 配置重载

isrvd 支持运行时重载配置和服务连接，无需重启进程。

### 重载方式

| 方式 | 说明 |
|------|------|
| etcd 配置变更 | 自动触发，无需手动操作 |
| `kill -HUP <pid>` | 手动触发，适用于本地文件配置场景 |

```bash
kill -HUP $(pgrep isrvd)
```

### 触发行为

1. 重新从配置源加载配置
2. 重新初始化 registry 客户端连接（APISIX/Caddy/Docker）
3. 重新初始化各业务服务
4. 服务恢复可用后，对应 API 立即生效

### 服务不可用时的行为

服务初始化失败时，对应模块的路由仍然注册，但请求时返回 `503 Service Unavailable`：

```json
{
  "error": "apisix service unavailable",
  "module": "apisix",
  "label": "查询 APISIX 路由列表",
  "reload": "send SIGHUP to reload services"
}
```

## 获取配置

```bash
isrvd_get "/system/config"
```

| 字段 | 类型 | 说明 |
|------|------|------|
| server | object | `{debug, listenAddr, jwtExpiration, maxUploadSize, proxyHeaderName, proxyTrustedCIDRs, rootDirectory, allowedOrigins}`（jwtSecret 不返回） |
| agent | object | `{model, baseUrl}`（apiKey 不返回） |
| oidc | object | `{enabled, issuerUrl, clientId, redirectUrl, usernameClaim, scopes}`（clientSecret 不返回） |
| apisix | object | `{adminUrl}`（adminKey 不返回） |
| caddy | object | `{adminUrl}` |
| docker | object | `{host, containerRoot, registries}`（registry password 不返回） |
| marketplace | object | `{url}` |
| links | object[] | `{label, url, icon}` |

## 更新配置

> 先通过 `isrvd_get "/system/config"` 获取当前值，按需修改后提交，不要硬编码配置内容。

```bash
isrvd_put "/system/config" '<CURRENT_CONFIG_WITH_CHANGES>'
```

配置说明：

- `clientSecret`、`jwtSecret`、`apiKey`、`adminKey`、`docker.registries[].password` 等敏感字段不会通过 GET 返回；PUT 时为空表示保留原值。
- 启用 Header 代理认证时，必须配置 `server.proxyHeaderName`；`server.proxyTrustedCIDRs` 可限制可信代理来源（如 `["10.0.0.0/8"]`），**未配置时不做来源限制**（向后兼容）。
- `oidc.redirectUrl` 生产环境建议显式配置固定 HTTPS 地址；留空会按当前请求 Host 自动生成，适合本地开发。
- `oidc.usernameClaim` 默认 `sub`；如改用 `email`，需确保 IdP 已验证邮箱且本地 `members.username` 与邮箱完全一致。

---

## 审计日志

审计策略由后端路由的 `Audit` 字段控制：`0` 按 Method 审计（非 GET 与 WebSocket 记录），`-1` 忽略，`1` 强制记录。未显式配置时默认为 `0`。文件管理读取类接口 `/filer/list`、`/filer/read`、`/filer/download` 配置为 `-1`，不记录审计日志。

```bash
isrvd_get "/system/audit/logs?limit=20"
```

| 字段 | 类型 | 说明 |
|------|------|------|
| timestamp | string | 时间戳 |
| username | string | 操作用户 |
| method | string | HTTP 方法 |
| uri | string | 请求路径 |
| body | string | 请求体 |
| ip | string | 来源 IP |
| statusCode | number | 响应状态码 |
| success | boolean | 是否成功 |
| duration | number | 耗时（ms） |
