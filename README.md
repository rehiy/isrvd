# Isrvd - Web Server Manager

一个基于 Go 和 Vue.js 的轻量级服务器管理工具，支持文件操作、在线编辑、文件传输、压缩解压和实时终端。

## 功能特性

- 📁 **文件管理** - 浏览、创建、删除、重命名、权限修改
- 📤 **文件传输** - 上传下载文件
- 📝 **在线编辑** - 直接编辑文本文件
- 🗜️ **压缩解压** - ZIP 文件打包和解压
- 🖥️ **Web 终端** - 实时 Shell 交互
- 🔐 **用户认证** - 安全的登录系统

## 快速开始

### 编译环境

- Go 1.21+
- Node.js 16+

### 编译打包

```shell
chmod +x build.sh && ./build.sh
```

## 启动服务

```bash
export LISTEN_ADDR=":8080"           # 监听端口
export BASE_DIRECTORY="/home/data"   # 管理目录
export MEMBERS="admin:pass,user:123" # 成员配置

./isrvd
```

访问 `http://localhost:8080`

## 许可证

MIT License
