# Caddy 路由 API

> Caddy 路由模型与 APISIX 不同：路由是 `apps.http.servers.<name>.routes` 数组中的元素，
> 用数组下标（index）作为主键。默认 server 名称为 `srv0`，可通过 query 参数 `?server=<name>` 切换。

## 列出路由

```bash
isrvd_get "/caddy/routes"                      # 默认 server=srv0
isrvd_get "/caddy/routes?server=srv0"          # 显式指定 server
```

| 字段 | 类型 | 说明 |
|------|------|------|
| index | number | 数组下标（路由主键，只读，由后端返回） |
| group | string | 路由分组（可选） |
| match | object | 匹配条件，结构见下表 |
| handler | object | 处理器配置，结构见下表 |
| terminal | boolean | 终止后续路由匹配 |
| id | string | Caddy `@id` 字段（可选，用于 admin api 寻址） |

### match 字段

| 字段 | 类型 | 说明 |
|------|------|------|
| hosts | string[] | 匹配 Host，留空表示不限制 |
| paths | string[] | 匹配 Path，支持 `*` 通配符 |
| methods | string[] | 匹配 HTTP 方法 |

> 字段为空数组等价于"不限制"。

### handler 字段

handler 通过 `kind` 字段区分类型，每种类型只填对应字段。

| kind | 字段 | 说明 |
|------|------|------|
| `reverse_proxy` | `upstreams: string[]` | 反向代理上游 `host:port` 列表；Web UI 可选择运行中 Docker 容器与端口自动填充 |
| `file_server` | `root: string`, `browse: boolean` | 静态文件服务 |
| `static_response` | `statusCode: number`, `body: string` | 直接返回固定响应 |
| `raw` | `raw: any` | 原始 handle 数组（任意 Caddy 模块），高级用法 |

## 查看路由详情

```bash
isrvd_get "/caddy/route/<INDEX>"
isrvd_get "/caddy/route/0?server=srv0"
```

## 创建路由

```bash
# 反向代理
isrvd_post "/caddy/route" '{
  "match": {"hosts": ["api.example.com"], "paths": ["/v1/*"]},
  "handler": {"kind": "reverse_proxy", "upstreams": ["backend:8080"]}
}'

# 静态文件
isrvd_post "/caddy/route" '{
  "match": {"paths": ["/static/*"]},
  "handler": {"kind": "file_server", "root": "/var/www", "browse": false}
}'

# 静态响应
isrvd_post "/caddy/route" '{
  "match": {"paths": ["/health"]},
  "handler": {"kind": "static_response", "statusCode": 200, "body": "OK"}
}'

# 原始 handle 数组（如 headers + reverse_proxy 链式组合）
isrvd_post "/caddy/route" '{
  "match": {"hosts": ["a.example.com"]},
  "handler": {"kind": "raw", "raw": [
    {"handler": "headers", "response": {"set": {"X-Forwarded-Host": ["{http.request.host}"]}}},
    {"handler": "reverse_proxy", "upstreams": [{"dial": "backend:8080"}]}
  ]}
}'
```

返回值 `payload.index`：新路由在数组中的下标。

## 更新路由

```bash
isrvd_put "/caddy/route/<INDEX>" '{
  "match": {"hosts": ["api.example.com"]},
  "handler": {"kind": "reverse_proxy", "upstreams": ["new-backend:8080"]}
}'
```

> 路由更新会触发 Caddy 整体配置 reload（`POST /load`），原子替换。

## 删除路由

```bash
isrvd_delete "/caddy/route/<INDEX>"
```

> 注意：删除后下标会重排，后续操作请重新查询路由列表获取最新 index。
