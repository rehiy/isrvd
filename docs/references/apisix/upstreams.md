# APISIX 上游 API

`upstream` 和 `upstream_id` 二选一：内联适合简单场景，引用适合多路由共享；同时传入时以 `upstream_id` 为准。

## Upstream 字段

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | | 上游 ID（只读，由 APISIX 生成） |
| name | string | ✅ | 名称 |
| type | string | ✅ | `roundrobin` / `chash` / `ewma` / `least_conn` |
| nodes | object \| array | ✅ | `{"addr:port": weight}`，或 APISIX 数组格式；不能为空 |
| desc | string | | 描述 |
| hash_on | string | | chash 依据 |
| key | string | | hash_on 对应的 key |
| scheme | string | | `http`/`https`/`grpc`/`grpcs` |
| pass_host | string | | `pass`/`node`/`rewrite` |
| upstream_host | string | | rewrite 时的主机名 |
| retries | number | | 重试次数；`0` 等同不设置（使用 APISIX 默认值） |
| retry_timeout | number | | 重试超时（秒）；`0` 等同不设置 |
| timeout | object | | `{"connect":5,"send":10,"read":10}` |
| create_time | number | | 创建时间（只读，Unix 秒） |
| update_time | number | | 更新时间（只读，Unix 秒） |

## 列出上游

```bash
isrvd_get "/apisix/upstreams"
```

## 查看上游详情

```bash
isrvd_get "/apisix/upstream/<UPSTREAM_ID>"
```

## 创建上游

```bash
isrvd_post "/apisix/upstream" '{"name":"<NAME>","type":"roundrobin","nodes":{"<HOST>:<PORT>":<WEIGHT>},"retries":2,"timeout":{"connect":5,"send":10,"read":10}}'
```

## 更新上游

```bash
isrvd_put "/apisix/upstream/<UPSTREAM_ID>" '{"name":"<NAME>","type":"roundrobin","nodes":{"<HOST>:<PORT>":<WEIGHT>}}'
```

更新调用 APISIX `PUT /upstreams/<ID>`，为**全量替换**：必填项与创建相同，未传的可选字段会被清除。

## 删除上游

```bash
isrvd_delete "/apisix/upstream/<UPSTREAM_ID>"
```
