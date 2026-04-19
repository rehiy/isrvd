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
2. 解析 `data.yml` 文件生成 `index.json`
3. 打包各版本文件到 `storage/应用名/版本号.zip`

```bash
pip install pyyaml
python build.py
```

## 数据结构

### index.json

```json
{
  "version": "1.0",
  "tags": [{ "key": "Database", "name": "数据库", "sort": 1 }],
  "apps": {
    "mysql": {
      "name": "MySQL",
      "description": { "zh": "开源关系型数据库" },
      "versions": {
        "8.0": {
          "formFields": [
            { "envKey": "MYSQL_ROOT_PASSWORD", "type": "password", "required": true }
          ]
        }
      }
    }
  }
}
```

### 安装 Payload

```json
{
  "source": "marketplace",
  "protocol": 1,
  "type": "install",
  "app": { "key": "mysql", "name": "MySQL" },
  "version": { "value": "8.0", "url": "..." },
  "instance": { "name": "my-mysql", "safeName": "my-mysql" },
  "env": { "system": {...}, "user": {...} },
  "deploy": { "network": "app-network" }
}
```

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
