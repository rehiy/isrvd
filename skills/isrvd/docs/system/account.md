# 账户与成员管理 API

## 获取认证信息

```bash
isrvd_get "/account/info"
```

返回：`{mode, username, member, oidcEnabled, oidcBtnLabel}`

> `oidcBtnLabel` 为 OIDC 登录按钮自定义名称，未配置时为空字符串。

## 登录

```bash
isrvd_post "/account/login" '{"username":"<USER>","password":"<PASS>"}'
```

返回：`{"token": "eyJ...", "username": "<USER>"}`

> 通常使用 `isrvd_login` 命令而非直接调用此接口。

## OIDC 登录

```bash
# 浏览器跳转，完成 OIDC Authorization Code Flow
GET /api/account/oidc/login
```

回调路径：`GET /api/account/oidc/callback`。登录成功后重定向到 `/?oidc_code=<CODE>`；失败重定向到 `/?oidc_error=<ERROR>`。

用一次性登录码换取系统 JWT：

```bash
isrvd_post "/account/oidc/exchange" '{"code":"<OIDC_CODE>"}'
```

返回：`{"token": "eyJ...", "username": "<USER>"}`

> OIDC 提取的用户名由 `oidc.usernameClaim` 指定，默认 `sub`，必须存在于 `members.username`；不存在时与 Header 认证一致，登录失败且不会自动创建成员。一次性 `oidc_code` 是短期凭证，勿复制或写入外部日志；Header 代理认证模式下不显示 OIDC 登录入口。

## 列出路由权限

```bash
isrvd_get "/account/routes"
```

返回按 `module`、`key` 排序的路由列表。

| 字段 | 类型 | 说明 |
|------|------|------|
| key | string | 路由权限键，格式为 `METHOD /api/path` |
| module | string | 模块名 |
| label | string | 路由显示名 |
| access | number | 访问级别：`0`=需要具体权限，`1`=登录即可访问，`-1`=匿名访问 |

> 路由未显式配置 `access` 时默认为 `0`，即需要具体权限。

## 创建 API Token

```bash
isrvd_post "/account/token" '{"name":"<TOKEN_NAME>","expiresIn":2592000}'
```

返回：`{"token": "长效token..."}`

## 修改密码

```bash
isrvd_put "/account/password" '{"oldPassword":"<OLD>","newPassword":"<NEW>"}'
```

## 列出成员

```bash
isrvd_get "/account/members"
```

| 字段 | 类型 | 说明 |
|------|------|------|
| username | string | 用户名 |
| homeDirectory | string | 主目录 |
| founder | boolean | 是否为创建者 |
| description | string | 描述 |
| permissions | string[] | 权限列表 |

## 创建成员

```bash
isrvd_post "/account/member" '{"username":"<USER>","password":"<PASS>","homeDirectory":"<HOME_DIR>","description":"<DESC>","permissions":["GET /api/docker/containers","GET /api/docker/images"]}'
```

## 更新成员

```bash
isrvd_put "/account/member/<USER>" '{"description":"<DESC>","permissions":["GET /api/docker/containers","GET /api/docker/images","GET /api/swarm/services"]}'
```

> password 为空则不修改。

## 删除成员

```bash
isrvd_delete "/account/member/<USER>"
```
