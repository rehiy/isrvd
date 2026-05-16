# Caddy TLS 证书 API

> Caddy 证书有 3 种来源，对应 `apps.tls.certificates.load_files`、`apps.tls.certificates.load_pem`、`apps.tls.automation.policies[].subjects`。
> 后端用复合主键 `<source>-<index>` 统一定位（例如 `file-0`、`pem-1`、`automate-2`）。

## 列出证书

```bash
isrvd_get "/caddy/certs"
```

返回数组，每项字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| key | string | 复合主键 `<source>-<index>`（只读） |
| source | string | `file` / `pem` / `automate` |
| certificate | string | file: 路径；pem: PEM 文本；automate: 不返回 |
| tags | string[] | Caddy 内部标签（可选） |
| format | string | 仅 file 使用，证书格式（默认 PEM） |
| subject | string | 仅 automate 使用，目标主机名 |

> 列表接口不返回 `keyContent`；更新时留空表示保留原私钥。

## 添加证书

```bash
# 来源 1: 磁盘文件
isrvd_post "/caddy/cert" '{
  "source": "file",
  "certificate": "/etc/caddy/example.crt",
  "keyContent": "/etc/caddy/example.key",
  "tags": ["example", "prod"]
}'

# 来源 2: 内联 PEM（适合不愿写盘的小规模部署）
isrvd_post "/caddy/cert" '{
  "source": "pem",
  "certificate": "-----BEGIN CERTIFICATE-----\nMIIDxxx...\n-----END CERTIFICATE-----",
  "keyContent": "-----BEGIN PRIVATE KEY-----\nMIIEvxxx...\n-----END PRIVATE KEY-----"
}'

# 来源 3: 自动签发（ACME）
isrvd_post "/caddy/cert" '{
  "source": "automate",
  "subject": "example.com"
}'
```

> ⚠️ 内联 PEM 会作为字符串写入 Caddy 完整配置，整体替换时一并下发；
> 注意配置文件大小，敏感性较高时优先用 `file` 来源 + 文件系统权限控制。

## 更新证书

```bash
# key 是列表返回里的 key 字段
isrvd_put "/caddy/cert/file-0" '{
  "source": "file",
  "certificate": "/etc/caddy/new.crt",
  "keyContent": "/etc/caddy/new.key"
}'
```

> 不允许跨来源更新（例如 `file-0 → pem`）；如果要换来源，请先删除再创建。

## 删除证书

```bash
isrvd_delete "/caddy/cert/automate-0"
```

> 删除会触发整体配置 reload，相同 source 下后续元素的 index 会前移。

## 典型工作流

### 把通配符证书铺到所有 Caddy 路由

```bash
# 1. 把证书写到 filer 内（容器可读路径）
isrvd_upload "/filer/upload" "file" "wildcard.example.com.crt" "path=caddy/certs"
isrvd_upload "/filer/upload" "file" "wildcard.example.com.key" "path=caddy/certs"

# 2. 在 Caddy 中引用
isrvd_post "/caddy/cert" '{
  "source": "file",
  "certificate": "/data/caddy/certs/wildcard.example.com.crt",
  "keyContent": "/data/caddy/certs/wildcard.example.com.key"
}'

# 3. 创建路由（同域名会自动匹配证书）
isrvd_post "/caddy/route" '{
  "match": {"hosts": ["api.example.com"]},
  "handler": {"kind": "reverse_proxy", "upstreams": ["backend:8080"]}
}'
```

### 启用自动签发

```bash
# 前置条件：caddy 容器需要能从外网访问 80/443，或配置 DNS-01 challenge
isrvd_post "/caddy/cert" '{"source": "automate", "subject": "example.com"}'
isrvd_post "/caddy/cert" '{"source": "automate", "subject": "api.example.com"}'
```
