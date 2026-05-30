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
| size | number | 大小（字节） |

## 查看数据卷详情

```bash
isrvd_get "/docker/volume/<VOL_NAME>"
```

额外字段：`scope, refCount, usedBy[]{id, name, mountPath, readOnly}`

## 创建数据卷

```bash
isrvd_post "/docker/volume" '{"name":"<NAME>","driver":"local"}'
```

## 删除数据卷

```bash
isrvd_post "/docker/volume/<VOL_NAME>/action" '{"action":"remove"}'
```
