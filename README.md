# web-indexr-go

一个基于 Go 的 Web 索引与文件管理服务。

## 项目结构

- `main.go`：程序入口，启动 Web 服务。
- `server/`：后端服务逻辑，包括认证、文件系统、压缩等处理器和中间件。
- `public/`：前端静态资源目录。
  - `index.html`：主页面。
  - `assets/`：静态资源（JS、CSS、字体等）。

## 主要功能

- 用户认证
- 文件浏览与管理
- 文件压缩/解压
- RESTful API 支持
- 前后端分离，前端基于 Vue.js

## 快速开始

1. **安装依赖**

   ```bash
   go mod tidy
   ```

2. **运行服务**

   ```bash
   go run main.go
   ```

3. **访问前端**

   打开浏览器访问 [http://localhost:8080](http://localhost:8080)

## 依赖

- Go 1.18+
- Vue.js (前端)
- Bootstrap, Font Awesome, Axios (前端依赖)

## 目录说明

- `server/`：后端核心逻辑
- `public/`：前端静态资源

## 许可证

MIT License
