# APISIX 路由管理 API

> 所有接口前缀: `/api/apisix`  
> 只读操作需要 `apisix:r` 权限，写操作需要 `apisix:rw` 权限

---

## §1 路由查询

### §1.1 列出路由

```
GET /api/apisix/routes
```

返回 `Route[]`：

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 路由 ID |
| name | string | 路由名称 |
| uri | string | 匹配路径（单个） |
| uris | string[] | 匹配路径（多个） |
| host | string | 匹配域名（单个） |
| hosts | string[] | 匹配域名（多个） |
| desc | string | 描述 |
| status | number | `1`=启用, `0`=禁用 |
| priority | number | 优先级（数字越大越优先） |
| enable_websocket | boolean | 是否启用 WebSocket 代理 |
| plugin_config_id | string | 引用的插件配置 ID |
| upstream_id | string | 引用的上游 ID |
| upstream | object | 内联上游配置 |
| plugins | object | 插件配置 |

### §1.2 查看路由详情

```
GET /api/apisix/route/:id
```

---

## §2 上游配置

`upstream` 和 `upstream_id` 二选一：

- **内联 upstream**：适合简单场景，直接在路由中定义
- **upstream_id**：引用已有上游，适合多路由共享

内联 upstream 结构：

```json
{
  "type": "roundrobin",
  "nodes": {
    "127.0.0.1:8080": 1,
    "127.0.0.1:8081": 2
  }
}
```

> `type` 可选: `roundrobin`（加权轮询）、`chash`（一致性哈希）、`ewma`（最小延迟）  
> `nodes` 的值为权重

---

## §3 创建路由

```
POST /api/apisix/routes
```

**最小示例**：

```json
{
  "name": "my-api",
  "uri": "/api/*",
  "status": 1,
  "upstream": {
    "type": "roundrobin",
    "nodes": { "127.0.0.1:3000": 1 }
  }
}
```

**完整示例**（含域名、插件、WebSocket）：

```json
{
  "name": "web-app",
  "uri": "/*",
  "host": "app.example.com",
  "status": 1,
  "priority": 10,
  "enable_websocket": true,
  "upstream": {
    "type": "roundrobin",
    "nodes": { "10.0.0.1:8080": 1, "10.0.0.2:8080": 1 }
  },
  "plugins": {
    "proxy-rewrite": {
      "regex_uri": ["^/api/v1/(.*)", "/$1"]
    },
    "limit-req": {
      "rate": 100,
      "burst": 50,
      "key_type": "var",
      "key": "remote_addr",
      "rejected_code": 429
    }
  }
}
```

---

## §4 更新路由

```
PUT /api/apisix/route/:id
Body: <同创建路由的 Route 结构>
```

---

## §5 启用/禁用路由

```
PATCH /api/apisix/route/:id/status
Body: { "status": 1 }   // 1=启用, 0=禁用
```

---

## §6 删除路由

```
DELETE /api/apisix/route/:id
```

---

## §7 消费者管理

### §7.1 列出消费者

```
GET /api/apisix/consumers
```

### §7.2 创建消费者

```
POST /api/apisix/consumers
Body: { "username": "consumer1", "desc": "描述" }
```

### §7.3 更新消费者

```
PUT /api/apisix/consumer/:username
Body: { "desc": "新描述" }
```

### §7.4 删除消费者

```
DELETE /api/apisix/consumer/:username
```

---

## §8 白名单管理

### §8.1 获取白名单

```
GET /api/apisix/whitelist
```

### §8.2 撤销白名单

```
PUT /api/apisix/whitelist/revoke
Body: { "route_id": "路由ID", "consumer_name": "消费者名" }
```

---

## §9 其他查询

```
GET /api/apisix/plugin_configs    # 列出插件配置
GET /api/apisix/plugins           # 列出所有可用插件
GET /api/apisix/upstreams         # 列出已定义的上游
```

---

## 常见工作流

### 为新服务配置路由

```bash
# 1. 查看现有路由，避免冲突
isrvd_get "/apisix/routes"

# 2. 查看可用上游
isrvd_get "/apisix/upstreams"

# 3. 创建路由
isrvd_post "/apisix/routes" '{
  "name": "new-service",
  "uri": "/new-service/*",
  "host": "api.example.com",
  "status": 1,
  "upstream": {
    "type": "roundrobin",
    "nodes": {"127.0.0.1:3000": 1}
  }
}'

# 4. 验证
isrvd_get "/apisix/routes"
```

### 临时禁用路由

```bash
isrvd_patch "/apisix/route/ROUTE_ID/status" '{"status": 0}'
```

### 修改路由上游节点

```bash
# 获取当前路由配置
isrvd_get "/apisix/route/ROUTE_ID"

# 更新（需要传完整 Route 结构）
isrvd_put "/apisix/route/ROUTE_ID" '{
  "name": "my-api",
  "uri": "/api/*",
  "status": 1,
  "upstream": {
    "type": "roundrobin",
    "nodes": {"10.0.0.1:3000": 1, "10.0.0.2:3000": 1}
  }
}'
```
