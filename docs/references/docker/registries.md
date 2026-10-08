# Docker 镜像仓库 API

## 列出仓库

```bash
isrvd_get "/docker/registries"
```

| 字段 | 类型 | 说明 |
|------|------|------|
| name | string | 仓库名 |
| url | string | 仓库地址 |
| username | string | 用户名 |
| description | string | 描述 |

## 添加仓库

```bash
isrvd_post "/docker/registry" '{"name":"<NAME>","url":"<REGISTRY_URL>","username":"<USER>","password":"<PASS>","description":"<DESC>"}'
```

`name`、`url` 必填；仓库以 `url` 作为唯一键，重复时返回 400"仓库地址已存在"。密码会去除首尾空白后保存。

## 更新仓库

```bash
isrvd_put "/docker/registry?url=<REGISTRY_URL>" '{"name":"<NAME>","url":"<REGISTRY_URL>","username":"<USER>","password":"<PASS>","description":"<DESC>"}'
```

> Query 中的 `url` 为原仓库地址（必填）；密码为空（含纯空白）时保留原密码，非空密码会去除首尾空白后保存。原仓库不存在或新 `url` 与其他仓库冲突时返回 400。

## 删除仓库

```bash
isrvd_delete "/docker/registry?url=<REGISTRY_URL>"
```

缺少 `url` 或仓库不存在时返回 400。
