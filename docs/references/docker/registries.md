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

## 更新仓库

```bash
isrvd_put "/docker/registry?url=<REGISTRY_URL>" '{"name":"<NAME>","url":"<REGISTRY_URL>","username":"<USER>","password":"<PASS>","description":"<DESC>"}'
```

> 密码为空时保留原密码。

## 删除仓库

```bash
isrvd_delete "/docker/registry?url=<REGISTRY_URL>"
```
