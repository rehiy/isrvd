# Isrvd

轻量级 Web 服务器管理工具，基于 Go + Vue 3 构建，提供文件管理、在线编辑、Docker 管理和实时终端功能。

## 特性

- **文件管理** - 浏览、上传、下载、创建、删除、重命名、权限修改
- **在线编辑** - CodeMirror 6 编辑器，支持多种语言语法高亮
- **压缩解压** - ZIP 打包和解压
- **Docker 管理** - 容器、镜像、网络、卷的完整管理
- **Web 终端** - xterm.js 实时 Shell 交互
- **多用户** - 用户隔离、独立家目录、权限控制

## 快速开始

### 安装

从 [Releases](https://github.com/rehiy/isrvd/releases) 下载对应平台二进制文件。

### 配置

编辑 `config.yml`：

```yaml
server:
  listenAddr: ":8080"        # 监听地址
  jwtSecret: your-secret-key # JWT 密钥
  proxyHeaderName: ""        # 内网代理认证 Header 名（可选，见下方说明）
  rootDirectory: "."         # 根目录

members:
  - username: admin
    password: admin123
    homeDirectory: public
    allowTerminal: true      # 允许终端访问
```

支持环境变量覆盖：`DEBUG`、`LISTEN_ADDR`、`ROOT_DIRECTORY`、`JWT_SECRET`、`PROXY_HEADER_NAME`

#### 内网代理 Header 认证

当服务运行在内网反向代理（如 Nginx、Apisix、Traefik）之后时，可由代理在请求头中注入已认证的用户名，isrvd 直接信任该 Header，跳过 JWT 验证。

```yaml
server:
  proxyHeaderName: X-Remote-User  # 与代理配置的 Header 名保持一致
```

> **安全提示**：启用此功能时，请确保 isrvd 不对外网直接暴露，并在代理层严格过滤或覆盖该 Header，防止客户端伪造。

### 运行

```bash
./isrvd
```

访问 http://localhost:8080

## 功能详情

### 文件管理

| 操作 | 说明 |
|------|------|
| 浏览 | 目录列表，面包屑导航 |
| 上传 | 支持拖拽上传 |
| 下载 | 文件流下载 |
| 编辑 | 在线编辑文本文件 |
| 压缩 | 打包为 ZIP |
| 解压 | 解压 ZIP 文件 |
| 权限 | 修改文件权限 |

### Docker 管理

| 模块 | 功能 |
|------|------|
| 容器 | 列表、创建、启动、停止、重启、删除、日志 |
| 镜像 | 列表、拉取、删除 |
| 网络 | 列表、创建、删除 |
| 卷 | 列表、创建、删除 |

### 在线编辑

支持语言：CSS、Go、HTML、JavaScript、JSON、Markdown、Python、SQL、XML、YAML

### Web 终端

支持 Shell：bash、sh、zsh、fish

## 编译

### 环境要求

- Go 1.21+
- Node.js 16+

### 构建

```bash
chmod +x build.sh
./build.sh
```

构建产物位于 `build/` 目录。

## 技术栈

**后端**

| 技术 | 用途 |
|------|------|
| Go 1.25 | 服务端语言 |
| Gin | Web 框架 |
| JWT | 身份认证 |
| WebSocket | 终端通信 |
| Docker SDK | Docker 管理 |

**前端**

| 技术 | 用途 |
|------|------|
| Vue 3 | 前端框架 |
| Vue Router | 路由管理 |
| Vite 7 | 构建工具 |
| Tailwind CSS | 样式框架 |
| Axios | HTTP 客户端 |
| CodeMirror 6 | 代码编辑器 |
| xterm.js | 终端模拟器 |
| Cherry Markdown | Markdown 编辑器 |

## 安全特性

- **用户隔离** - 每个用户独立家目录
- **路径验证** - 防止目录遍历攻击
- **JWT 认证** - Token 有效期 24 小时
- **内网代理认证** - 可配置信任代理注入的用户名 Header，适用于 SSO/零信任网络
- **Zip Slip 防护** - 解压路径验证
- **终端权限控制** - 可配置禁止终端访问

## 许可证

GPL-3.0
