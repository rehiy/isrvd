# Web File Manager

一个基于 Go 和 Vue.js 的现代化文件管理器，提供完整的文件系统操作和终端交互功能。

## 功能特性

- 🗂️ 文件浏览和管理（创建、删除、重命名、权限修改）
- 📤 文件上传和下载
- 📝 在线文件编辑
- 🗜️ 文件压缩和解压
- 🖥️ Web终端（支持Shell交互）
- 🔐 用户认证和会话管理
- 🌐 现代化的响应式界面

## 项目结构

```text
.
├── main.go                 # 应用入口
├── go.mod                  # Go模块定义
├── go.sum                  # Go依赖锁定
├── internal/               # 内部包（不对外暴露）
│   ├── config/            # 配置管理
│   ├── handlers/          # HTTP处理器
│   ├── middleware/        # 中间件
│   ├── models/           # 数据模型
│   ├── router/           # 路由配置
│   └── services/         # 业务逻辑服务
├── pkg/                   # 公共包（可对外暴露）
│   ├── auth/             # 认证模块
│   └── utils/            # 工具函数
├── front/                 # Vue.js前端源码
└── public/               # 编译后的静态文件
```

## 环境配置

### 环境变量

- `BASE_DIR`: 文件管理器的根目录（默认: 当前目录）
- `PORT`: 服务端口（默认: 8080）
- `USERS`: 用户配置，格式：`username1:password1,username2:password2`（默认: admin:admin）

### 示例

```bash
export BASE_DIR="/home/user/files"
export PORT="9000"
export USERS="admin:secret123,user:pass456"
```

## 构建和运行

### 后端构建

```bash
# 构建
go build -o web-indexr-go

# 运行
./web-indexr-go
```

### 前端构建

```bash
cd front
npm install
npm run build
```

### Docker 运行

```bash
# 构建镜像
docker build -t web-indexr-go .

# 运行容器
docker run -d \
  -p 8080:8080 \
  -v /path/to/files:/data \
  -e BASE_DIR=/data \
  -e USERS="admin:yourpassword" \
  web-indexr-go
```

## API 接口

### 认证

- `POST /api/login` - 用户登录
- `POST /api/logout` - 用户登出

### 文件操作

- `GET /api/files?path=/` - 获取文件列表
- `POST /api/upload` - 上传文件
- `GET /api/download?file=/path/to/file` - 下载文件
- `DELETE /api/delete?file=/path/to/file` - 删除文件
- `POST /api/mkdir` - 创建目录
- `POST /api/newfile` - 新建文件
- `GET /api/edit?file=/path/to/file` - 读取文件内容
- `PUT /api/edit?file=/path/to/file` - 保存文件内容
- `PUT /api/rename` - 重命名文件
- `GET /api/chmod?file=/path/to/file` - 获取文件权限
- `PUT /api/chmod?file=/path/to/file` - 修改文件权限
- `POST /api/zip` - 压缩文件
- `POST /api/unzip` - 解压文件

### WebSocket

- `GET /ws/shell` - Shell终端WebSocket连接

## 安全特性

- JWT会话管理（24小时过期）
- 路径验证（防止目录遍历攻击）
- Zip Slip攻击防护
- CORS支持
- 自动会话清理

## 技术栈

### 后端

- Go 1.21+
- Gin Web框架
- Gorilla WebSocket
- PTY终端仿真

### 前端

- Vue.js 3
- Vite构建工具
- 现代化CSS

## 开发指南

### 代码结构说明

1. **internal/**: 内部包，遵循Go最佳实践，不对外暴露
   - `config/`: 应用配置管理
   - `handlers/`: HTTP请求处理器
   - `middleware/`: 中间件（认证、CORS等）
   - `models/`: 数据模型定义
   - `router/`: 路由配置
   - `services/`: 业务逻辑服务层

2. **pkg/**: 公共包，可以被其他项目引用
   - `auth/`: 会话管理和认证
   - `utils/`: 通用工具函数

### 扩展功能

要添加新功能，请遵循以下步骤：

1. 在 `internal/models/` 中定义数据模型
2. 在 `internal/services/` 中实现业务逻辑
3. 在 `internal/handlers/` 中创建HTTP处理器
4. 在 `internal/router/` 中注册路由
5. 更新前端界面（如需要）

## 许可证

MIT License
