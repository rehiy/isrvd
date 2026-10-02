# Docker 网络 API

## 列出网络

```bash
isrvd_get "/docker/networks"
```

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 网络短 ID（12位） |
| name | string | 网络名 |
| driver | string | 驱动（bridge/overlay/host/...） |
| subnet | string | 子网 |
| scope | string | 作用域（local/swarm） |

## 查看网络详情

```bash
isrvd_get "/docker/network/<NETWORK_ID>"
```

详情中 `id` 为完整 ID；额外字段：`gateway, internal, enableIPv6, containers[]{id, name, ipv4, ipv6, macAddress}`。`subnet` / `gateway` 取第一个 IPAM 配置；`ipv4` / `ipv6` 带 CIDR 前缀（如 `172.18.0.2/16`）。

## 创建网络

```bash
isrvd_post "/docker/network" '{"name":"<NAME>","driver":"<DRIVER>","subnet":"<CIDR>"}'
```

`name` 必填；`driver` 默认 `bridge`；`subnet` 可选，填写时会作为 Docker IPAM 子网创建网络。返回 `{id, name}`（`id` 为短 ID）。

## 删除网络

```bash
isrvd_post "/docker/network/<NETWORK_ID>/action" '{"action":"remove"}'
```
