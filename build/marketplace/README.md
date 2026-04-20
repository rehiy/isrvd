# Docker 应用市场

一个现代化的 Docker 应用商店前端，提供可视化的应用浏览、搜索和安装体验。

## 功能特性

- **应用浏览** - 分类展示所有可用应用，支持搜索过滤
- **详情查看** - 查看应用介绍、版本列表、README 文档
- **一键安装** - 填写配置后生成安装脚本或通过 postMessage 与父窗口通信
- **响应式设计** - 完美适配桌面端和移动端
- **懒加载优化** - 图片按需加载，提升首屏性能

## 技术栈

- **Vue 3** - 渐进式 JavaScript 框架
- **Tailwind CSS** - 实用优先的 CSS 框架
- **Marked** - Markdown 解析器
- **Font Awesome** - 图标库

## 快速开始

### 本地运行

需要通过 HTTP 服务器访问（直接打开 HTML 文件会有跨域限制）：

```bash
# 使用 Python
python -m http.server 8080

# 或使用 Node.js
npx serve .
```

然后访问 <http://localhost:8080>

### Docker 部署

```bash
docker run -d -p 8080:80 rehiy/appstore
```

### 嵌入到其他系统

应用市场支持被嵌入到 iframe 中，安装时会通过 `postMessage` 向父窗口发送结构化数据：

```javascript
// 父窗口监听
window.addEventListener('message', (event) => {
  if (event.data?.source === 'marketplace' && event.data?.type === 'install') {
    // 处理安装请求
    console.log('安装应用:', event.data);
  }
});
```

## 构建说明

运行 `build.py` 会：

1. 从 GitHub 下载 1Panel appstore 源码
2. 解析 `data.yml` 文件生成 `index.json`（仅含应用元信息与版本号，不再携带 formFields 等详情）
3. 为每个版本在 `storage/应用名/版本号/` 下生成：
   - `meta.yml`：包含 `compose`（原始模板，含 `${VAR}` 占位，使用 YAML `|` 块字面量样式保持源文件可读性）、`formFields`（表单字段定义）、可选 `init`（附加运行文件 zip 相对路径）
   - `init.zip`：仅当存在 compose 以外的运行时文件（如配置脚本、SQL 等）时生成

```bash
pip install pyyaml
python build.py
```

## 数据结构

### index.json（轻量，仅元信息）

```json
{
  "version": "1.0",
  "tags": [{ "key": "Database", "name": "数据库", "sort": 1 }],
  "apps": {
    "mysql": {
      "name": "MySQL",
      "description": { "zh": "开源关系型数据库" },
      "versions": {
        "8.0": { "architectures": ["amd64", "arm64"] }
      }
    }
  }
}
```

### storage/{app}/{version}/meta.yml（表单与 compose）

```yaml
compose: |
  version: '3.8'
  services:
    mysql:
      image: mysql:8.0
      environment:
        MYSQL_ROOT_PASSWORD: ${MYSQL_ROOT_PASSWORD}
      networks:
        - app-network
  networks:
    app-network:
      external: true
formFields:
  - envKey: MYSQL_ROOT_PASSWORD
    type: password
    required: true
    label:
      zh: Root 密码
init: init.zip
```

### 安装 postMessage Payload

前端已在浏览器端对 `compose` 完成 `${VAR}` 插值，父窗口拿到即可直接落盘启动：

```json
{
  "source": "marketplace",
  "type": "install",
  "name": "my-mysql",
  "compose": "version: '3.8'\nservices:\n  mysql:\n    image: mysql:8.0\n    environment:\n      MYSQL_ROOT_PASSWORD: s3cret\n    networks:\n      - app-network\n...",
  "initURL": "https://marketplace.example.com/storage/mysql/8.0/init.zip"
}
```

字段说明：

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| `source` | string | 是 | 固定为 `marketplace` |
| `type` | string | 是 | 固定为 `install` |
| `name` | string | 是 | 实例名，作为目录名 / compose project 名，匹配 `[a-zA-Z0-9][a-zA-Z0-9_.-]*` |
| `compose` | string | 是 | 前端插值完毕的完整 `docker-compose.yml` 文本 |
| `initURL` | string | 否 | 附加运行文件 `init.zip` 的绝对下载地址；无附加文件时字段省略 |

系统变量 `APP_NAME` / `CONTAINER_NAME` / `NETWORK_NAME`（默认 `app-network`）由前端自动注入到插值过程中，父窗口无需关心。

---

## 致谢

本项目数据来源于以下开源仓库，感谢他们的辛勤付出：

- [1Panel AppStore](https://github.com/1Panel-dev/appstore) - 提供丰富的 Docker 应用模板和配置文件
- [Vue.js](https://vuejs.org/) - 渐进式 JavaScript 框架
- [Tailwind CSS](https://tailwindcss.com/) - 实用优先的 CSS 框架
- [Marked](https://marked.js.org/) - 高性能 Markdown 解析器
- [Font Awesome](https://fontawesome.com/) - 经典图标库
- [GitHub Markdown CSS](https://github.com/sindresorhus/github-markdown-css) - GitHub 风格 Markdown 样式
- [Python PyYAML](https://pyyaml.org/) - YAML 解析库
- [Nginx](https://nginx.org/) - 高性能 Web 服务器
- [Docker](https://www.docker.com/) - 容器化平台
