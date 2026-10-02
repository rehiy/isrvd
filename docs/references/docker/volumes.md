# Docker 数据卷 API

## 列出数据卷

```bash
isrvd_get "/docker/volumes"
```

| 字段 | 类型 | 说明 |
|------|------|------|
| name | string | 卷名 |
| driver | string | 驱动 |
| mountpoint | string | 挂载路径 |
| createdAt | string | 创建时间 |
| size | number | 大小（字节）；列表接口不计算，当前恒为 `0` |

## 查看数据卷详情

```bash
isrvd_get "/docker/volume/<VOL_NAME>"
```

额外字段：`scope, size, refCount, usedBy[]{id, name, mountPath, readOnly}`。`size` / `refCount` 为 `-1` 表示未知（Docker 未返回 UsageData 时，通常即为 `-1`）；`usedBy` 仅统计 `type=volume` 的挂载。

## 创建数据卷

```bash
isrvd_post "/docker/volume" '{"name":"<NAME>","driver":"local"}'
```

`name` 必填；`driver` 默认 `local`。返回 `{name, mountpoint}`。

## 删除数据卷

```bash
isrvd_post "/docker/volume/<VOL_NAME>/action" '{"action":"remove"}'
```

仅支持 `remove`，为强制删除。
