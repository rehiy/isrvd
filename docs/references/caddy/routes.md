# Caddy 路由 API

> Caddy 路由模型与 APISIX 不同：路由是 `apps.http.servers.<name>.routes` 数组中的元素，
> 用数组下标（index）作为主键。默认 server 名称为 `srv0`，可通过 query 参数 `?server=<name>` 切换。
>
> **接口直接使用 Caddy 原生 JSON 结构**，前端负责组装，后端透传给 Caddy，无中间转换层。

## 列出路由

```bash
isrvd_get "/caddy/routes"                      # 默认 server=srv0
isrvd_get "/caddy/routes?server=srv0"          # 显式指定 server
```

| 字段 | 类型 | 说明 |
|------|------|------|
| index | number | 数组下标（路由主键，只读，由后端附加） |
| group | string | 路由分组（可选） |
| match | MatchSet[] | 匹配条件数组，元素间为 OR 关系，同一元素内字段为 AND |
| handle | Handler[] | 处理器数组，按顺序执行 |
| terminal | boolean | 终止后续路由匹配 |
| @id | string | Caddy `@id` 字段（可选，用于 admin api 寻址） |

### MatchSet 字段（Caddy 原生）

| 字段 | 类型 | 说明 |
|------|------|------|
| host | string[] | 匹配 Host，留空表示不限制 |
| path | string[] | 匹配 Path，支持 `*` 通配符 |
| method | string[] | 匹配 HTTP 方法 |
| header | object | 按请求头匹配，key 为头字段名，value 为匹配值列表；value 为空数组表示只要该头存在即匹配 |
| protocol | string | 匹配协议：`http` / `https`，留空不限制 |

### Handler 字段（Caddy 原生）

每个 handler 必须包含 `handler` 字段（模块名），其余字段视模块而定。

| handler 值 | 说明 |
|---|---|
| `reverse_proxy` | 反向代理 |
| `file_server` | 静态文件服务 |
| `static_response` | 静态响应 |
| `rewrite` | URI 重写 |
| `headers` | 请求头/响应头操作 |

## 查看路由详情

```bash
isrvd_get "/caddy/route/<INDEX>"
isrvd_get "/caddy/route/0?server=srv0"
```

## 创建路由

```bash
# 反向代理
isrvd_post "/caddy/route" '{
  "match": [{"host": ["api.example.com"], "path": ["/v1/*"]}],
  "handle": [{"handler": "reverse_proxy", "upstreams": [{"dial": "backend:8080"}]}]
}'

# 反向代理（含超时）
isrvd_post "/caddy/route" '{
  "match": [{"host": ["api.example.com"]}],
  "handle": [{
    "handler": "reverse_proxy",
    "upstreams": [{"dial": "backend:8080"}],
    "transport": {"protocol": "http", "dial_timeout": "10s", "response_header_timeout": "30s"}
  }]
}'

# 按协议匹配（HTTP 跳转）
isrvd_post "/caddy/route" '{
  "match": [{"protocol": "http"}],
  "handle": [{"handler": "static_response", "status_code": 301}],
  "terminal": true
}'

# 按请求头匹配
isrvd_post "/caddy/route" '{
  "match": [{"header": {"X-Internal": []}}],
  "handle": [{"handler": "reverse_proxy", "upstreams": [{"dial": "internal-svc:8080"}]}]
}'

# 反向代理转发前重写 URI
isrvd_post "/caddy/route" '{
  "match": [{"path": ["/api/*"]}],
  "handle": [{
    "handler": "reverse_proxy",
    "upstreams": [{"dial": "backend:8080"}],
    "rewrite": {"strip_path_prefix": "/api"}
  }]
}'

# 设置响应头（CORS / 安全头）+ 反向代理
isrvd_post "/caddy/route" '{
  "match": [{"path": ["/api/*"]}],
  "handle": [
    {
      "handler": "headers",
      "response": {"set": {"Access-Control-Allow-Origin": ["*"], "X-Frame-Options": ["SAMEORIGIN"]}}
    },
    {"handler": "reverse_proxy", "upstreams": [{"dial": "backend:8080"}]}
  ]
}'

# 透传真实 IP（请求头）
isrvd_post "/caddy/route" '{
  "handle": [{
    "handler": "headers",
    "request": {
      "set": {
        "X-Real-IP": ["{http.request.remote.host}"],
        "X-Forwarded-For": ["{http.request.remote.host}"]
      }
    }
  }]
}'

# 静态文件
isrvd_post "/caddy/route" '{
  "match": [{"path": ["/static/*"]}],
  "handle": [{"handler": "file_server", "root": "/var/www"}]
}'

# 静态响应
isrvd_post "/caddy/route" '{
  "match": [{"path": ["/health"]}],
  "handle": [{"handler": "static_response", "status_code": 200, "body": "OK"}]
}'

# URI 重写
isrvd_post "/caddy/route" '{
  "match": [{"path": ["/api/*"]}],
  "handle": [{"handler": "rewrite", "strip_path_prefix": "/api"}]
}'

# 完整 URI 替换（支持占位符）
isrvd_post "/caddy/route" '{
  "match": [{"path": ["/old/*"]}],
  "handle": [{"handler": "rewrite", "uri": "/new/{http.request.uri.path.1}"}]
}'

# FastCGI（PHP-FPM）
isrvd_post "/caddy/route" '{
  "match": [{"host": ["php.example.com"]}],
  "handle": [{
    "handler": "reverse_proxy",
    "upstreams": [{"dial": "php-fpm:9000"}],
    "transport": {"protocol": "fastcgi", "root": "/var/www/html"}
  }]
}'
```

返回值 `payload.index`：新路由在数组中的下标。

## 更新路由

```bash
isrvd_put "/caddy/route/<INDEX>" '{
  "match": [{"host": ["api.example.com"]}],
  "handle": [{"handler": "reverse_proxy", "upstreams": [{"dial": "new-backend:8080"}]}]
}'
```

> 路由更新会触发 Caddy 整体配置 reload（`POST /load`），原子替换。

## 删除路由

```bash
isrvd_delete "/caddy/route/<INDEX>"
```

> 注意：删除后下标会重排，后续操作请重新查询路由列表获取最新 index。
