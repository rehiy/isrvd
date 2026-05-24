# AGENTS.md — isrvd Webview 前端规范

> 适用于 `webview/` 前端代码；通用规则仍以 `../AGENTS.md` 为准。

---

## 1) 前端编码规范（Vue/Tailwind）

适用目录：`webview/src`

### 1.1 组件装饰器

使用 `vue-facing-decorator`（`@Component`、`@Prop`、`@Ref`、`@Watch` 等），类组件必须 `toNative()` 包装后导出。全局状态禁止再新增 `@Inject`/`@Provide` 方案，统一走 Pinia。

### 1.2 状态管理

- 全局状态使用 Pinia，入口为 `webview/src/stores/index.ts` 导出的 `usePortal()`
- `main.ts` 中创建 `createPinia()`，再初始化 `portal = usePortal()` 并注入路由守卫
- `portal` 聚合 `auth`、`system`、`ui`、`filer` 等子 store；组件内统一 `portal = usePortal()`
- 权限：`permissionsLoaded`（布尔）、`permissions`（`string[]`，格式为 `"METHOD /api/path"`），通过 `portal.hasPerm(moduleOrRoute)` 检查；支持模块名（如 `docker`）和精确路由（如 `GET /api/docker/containers`）

### 1.3 API 服务层

`service/api.ts` 单例 `class ApiService`，`export default new ApiService()`。请求统一通过 `http`/`httpBlob`（`service/axios.ts`），前者类型安全已解包为 `APIResponse`，后者 Blob 下载专用

### 1.4 类型定义与命名（强制）

`service/types/` 按域拆分（`docker`、`swarm`、`apisix`、`caddy`、`compose`、`cron`、`system`、`account`、`overview`、`filer`），`service/types.ts` 统一 `export *` 导出

| 场景 | 命名 | 示例 |
|---|---|---|
| 列表/概览 | `XxxInfo` | `DockerContainerInfo`、`SwarmNodeInfo` |
| 详情/单条查询 | `XxxDetail` | `DockerImageDetail`、`SwarmNodeDetail` |
| 创建请求 | `XxxCreate` | `DockerContainerCreate`、`ApisixRouteCreate` |
| 更新请求 | `XxxUpdate` | `ApisixConsumerUpdate` |
| 复用创建类型 | `type XxxCreate = XxxSpec` | `ApisixUpstreamCreate = ApisixUpstream` |
| 响应结果 | `XxxResult` | `AuthLoginResult`、`ApiTokenResult` |
| 枚举联合 | `XxxType` / `XxxMode` | `ApisixUpstreamType`、`ComposeDeployTarget` |

禁止：`VO`/`BO`/`DTO` 等后缀

### 1.5 方法命名规范（强制）

`ApiService` 方法命名：`domainResourceAction` 驼峰格式

| 操作 | 命名模式 | 示例 |
|---|---|---|
| 列表 | `domainResourceList(params?)` | `dockerContainerList()`、`apisixRouteList()`、`cronJobList()` |
| 单条 | `domainResource(id)` | `dockerImage(id)`、`swarmNode(id)` |
| 创建/更新/删除 | `domainResourceCreate/Update/Delete` | `dockerContainerCreate(data)`、`dockerImageDelete(id)`、`cronJobCreate(data)` |
| 操作/动作 | `domainResourceAction(id, action)` | `dockerContainerAction(id, 'start')`、`cronJobRun(id)` |
| 状态切换 | `domainResourceStatus(id, status)` | `apisixRouteStatus(id, 0)`、`cronJobEnable(id, enabled)` |
| 统计/日志 | `domainResourceStats/Logs(id)` | `dockerContainerStats(id)`、`cronJobLogs(id)` |

- **域名前缀**：`docker`、`swarm`、`apisix`、`caddy`、`account`、`system`、`filer`、`compose`、`cron`
- **资源名**：单数形式
- **分组注释**：`// ==================== XXXX 相关 ====================`

### 1.6 卡片与标题栏

- 列表/详情页统一 `.card mb-4`
- 标题栏：使用 `.card-toolbar` 类（定义于 `light_components.css`），容器不写 `flex/justify-between`
- 必须提供桌面 `hidden md:flex` 与移动布局；移动端可用 `flex md:hidden`，也可用 `block md:hidden` 外层 + 内部 `flex items-center justify-between`
- 详情页右侧仅保留刷新等功能按钮，**不添加返回按钮**

**toolbar 图标与标题（强制）**：

- 图标：使用 `.page-icon` 类（布局）+ 调用方内联背景色（如 `bg-blue-500`）（禁止手写 `w-9 h-9 rounded-lg flex ...`）
- 标题：`<h1 class="text-lg font-semibold text-slate-800 truncate">`（不得用 `h3`/`h2`/`text-base`）
- 副标题：`<p class="text-xs text-slate-500 truncate">`
- 移动端左侧容器：`flex items-center gap-3 min-w-0 flex-1`，图标加 `flex-shrink-0`，文字容器加 `min-w-0`
- 桌面端右侧按钮区：`flex items-center gap-2 flex-shrink-0`（刷新等功能按钮）

**图标样式（强制）**：

- 形状：`rounded-lg`（禁止 `rounded-full`/`rounded-2xl`）
- 尺寸：使用对应 CSS 类——`.empty-state-icon`（空状态/登录 64×64）、`.page-icon`（toolbar 36×36）、`.list-icon`（移动卡片 40×40）、`.row-icon`（桌面表格 32×32）、`.card-icon`（卡片标题 24×24）

### 1.7 列表双视图与搜索（强制）

- 桌面：`hidden md:block overflow-x-auto` + `<table>`
- 移动：`md:hidden space-y-3 p-4` + `.card-interactive` 卡片列表（**`p-4` 不得省略**，禁止手写等价卡片 Tailwind 组合）
- 所有资源列表页必须提供 `searchText` + `filteredXxx` 过滤逻辑，并使用 `webview/src/component/page-search.vue`，禁止手写重复的搜索框 HTML
- 页面级键盘输入直达搜索只能通过 `PageSearch` 的 `type-to-search` 启用；禁止页面直接调用 `bindTypeToSearchFocus`
- 每个列表页只允许一个 `PageSearch` 设置 `type-to-search`（通常为桌面搜索框）；移动端搜索框复用同一个 `search-key`，但不设置 `type-to-search`
- `search-key` 使用稳定唯一值（建议与路由名一致，如 `docker-images`、`apisix-routes`、`system-audit-logs`）
- 过滤逻辑保留在页面内，不下沉到 `PageSearch`；组件只负责输入 UI、`v-model` 和键盘直达绑定

**移动端卡片顶部结构（强制）**：

```html
<div class="card-info-row">
  <div class="list-icon bg-xxx">
    <i class="fas fa-xxx text-white text-base"></i>
  </div>
  <div class="min-w-0">
    <span class="font-medium text-slate-800 text-sm truncate block">{{ 主名称 }}</span>
    <span class="text-xs text-slate-400 truncate block mt-0.5">{{ 副信息 }}</span>
  </div>
</div>
```

- 卡片图标：使用 `.list-icon` 类（40×40），禁止手写 `w-10 h-10 rounded-lg`
- 主名称：`font-medium text-slate-800 text-sm truncate block`
- 副信息：`text-xs text-slate-400 truncate block mt-0.5`

### 1.8 移动端卡片属性行对齐（强制）

卡片内每条属性行由 `<标签 span>` + `<值>` 组成，**行间距统一 `mb-3`**：

- 纯文本：`flex items-center gap-2 mb-3`，值用 `text-slate-500`
- badge/code：`flex items-start gap-2 mb-3`，标签加 `mt-0.5`
- badge 形状：`rounded-lg`（禁止 `rounded-full`/`rounded-md`）

```html
<!-- 纯文本 -->
<div class="flex items-center gap-2 mb-3">
  <span class="text-xs text-slate-400 flex-shrink-0">创建</span>
  <span class="text-xs text-slate-500">{{ value }}</span>
</div>

<!-- badge -->
<div class="flex items-start gap-2 mb-3">
  <span class="text-xs text-slate-400 flex-shrink-0 mt-0.5">状态</span>
  <span class="inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-medium ...">{{ value }}</span>
</div>

<!-- code -->
<div class="flex items-start gap-2 mb-3">
  <span class="text-xs text-slate-400 flex-shrink-0 mt-0.5">路径</span>
  <code class="text-xs bg-slate-100 px-2 py-0.5 rounded-lg break-all">{{ value }}</code>
</div>
```

### 1.9 表格第一列布局（强制）

- `<td>` 设 `max-w-[280px]`（纯短 ID 列除外）
- 副信息（描述/host/ID）显示在主名称下方，不占独立列
- 外层 flex 加 `min-w-0`，图标 `flex-shrink-0`，文字容器 `min-w-0` + `truncate block`
- 副信息样式：`text-xs text-slate-400 truncate block mt-0.5`；等宽内容加 `font-mono`；空值加 `v-if`
- **主名称直接用 `<span class="font-medium text-slate-800 truncate block">`，不得用额外 flex 容器包裹**

**桌面端表格文字颜色规范**：

- 数据列：`text-sm text-slate-600`
- 副信息行：`text-xs text-slate-400`
- 操作按钮列：`flex justify-end items-center gap-1`
- badge 形状：`rounded-lg`（禁止 `rounded-full`/`rounded-md`）

```html
<td class="px-4 py-3 max-w-[280px]">
  <div class="flex items-center gap-2 min-w-0">
    <div class="row-icon bg-xxx">
      <i class="fas fa-xxx text-white text-sm"></i>
    </div>
    <div class="min-w-0">
      <span class="font-medium text-slate-800 truncate block">{{ 主名称 }}</span>
      <span class="text-xs text-slate-400 truncate block mt-0.5">{{ 副信息 }}</span>
    </div>
  </div>
</td>
```

图标配色跟随资源页主色（图标背景通常用 `400` 色阶，toolbar 用 `500` 色阶）：Docker 容器 `emerald/slate`（按状态）、镜像 `blue`、网络 `purple`、数据卷 `amber`、仓库 `purple`；Swarm 节点 `blue`、服务 `emerald`、任务 `cyan`；APISIX 路由 `indigo`、上游 `emerald`、消费者 `violet`、SSL `cyan`、插件配置 `rose`、白名单 `amber`；Caddy 路由 `indigo`、证书 `cyan`、全局配置 `violet`、原始配置 `slate`；Compose `amber`、Cron `amber/violet`、用户 `blue`、Filer `primary`。新模块选未用色（`rose`/`cyan`/`lime` 等）

#### 1.9.1 状态文字颜色（强制）

**状态值优先用文字颜色区分，不用 badge**；仅枚举型分类字段（驱动、类型等）才用 badge。

| 状态值 | 颜色 | 示例 |
|---|---|---|
| 正常/运行/就绪（running / ready / active / enabled） | `text-emerald-600 font-medium` | 容器 running、节点 ready/active |
| 异常/停止/下线（stopped / down / error） | `text-red-500 font-medium` | 节点 down |
| 警告/排空/暂停（drain / paused / warning） | `text-amber-600 font-medium` | 节点 drain |
| 其他/未知 | `text-slate-500` | — |

- 通配符或空值（如 host `*`）用 `text-slate-400`（无 `font-medium`）
- 有意义的 host/域名用 `text-teal-600 font-medium`
- 数值型强调（如运行中任务数）用 `text-emerald-600 font-medium`

#### 1.9.2 枚举 badge 配色

枚举型分类字段（驱动、协议、类型等）使用 badge，颜色跟随模块主色：

| 模块/字段 | badge 配色 |
|---|---|
| 网络驱动（docker network driver） | `bg-purple-50 text-purple-700` |
| 路由上游（apisix upstream） | `bg-indigo-50 text-indigo-700` |
| 其他枚举 | 跟随模块主色，`bg-{color}-50 text-{color}-700` |

#### 1.9.3 权限/角色图标颜色

| 角色/权限 | 图标 | 颜色 |
|---|---|---|
| 创始人/超级管理员 | `fa-crown` | `text-violet-400` |
| 有自定义权限 | `fa-key` | `text-amber-400` |

### 1.10 操作按钮语义色

统一格式：使用 `.btn-icon-{color}` 语义类（如 `.btn-icon-slate`、`.btn-icon-blue`、`.btn-icon-emerald`）

| 操作 | 配色 | 图标 |
|---|---|---|
| 创建/启动/激活 | `emerald` | `fa-plus` / `fa-play` |
| 编辑/重启 | `blue` | `fa-pen` / `fa-rotate` |
| 停止/排空/告警 | `amber` | `fa-stop` / `fa-arrow-down` |
| 删除/移除 | `red` | `fa-trash` / `fa-xmark` |
| 详情/日志/暂停 | `slate` | `fa-circle-info` / `fa-file-lines` / `fa-pause` |
| 统计/扩缩容 | `indigo` | `fa-chart-line` / `fa-up-right-and-down-left-from-center` |
| 终端 | `teal` | `fa-terminal` |
| 禁用/只读 | `slate-300 cursor-not-allowed` | — |

> 编辑按钮颜色跟随资源主色：APISIX 路由 `indigo`、消费者 `violet`、SSL `cyan`、上游 `emerald`、插件配置 `rose`；Caddy 路由 `indigo`、证书 `cyan`；其他模块通常使用 `blue`，但可按资源主色保持一致。

**移动端操作按钮区（强制）**：

- 容器：`flex flex-wrap gap-1.5 pt-2 border-t border-slate-100`（**`gap-1.5` 不得用 `gap-1`**）

### 1.11 表单与敏感字段

- 表单容器 `max-w-3xl space-y-4`
- label：`block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1`，input 通用 `.input`，help `text-xs text-slate-400 mt-1`
- 密钥/密码：后端敏感字段 `json:"-"`，前端 `type="password" autocomplete="new-password"`，留空保存=不修改，placeholder："留空保持不变"

- **Toggle Switch 规范**（单一功能启用/禁用开关）：使用 `.toggle-row` + `.toggle` + `.toggle-thumb` CSS 类；通过 `:class="{ 'toggle-on': value }"` 控制激活态；默认激活色为 indigo，确需 violet 场景（如 Caddy Global）加 `toggle-violet` 修饰符；禁止手写内联 Tailwind toggle 样式：

  ```html
  <div class="toggle-row">
    <div>
      <span class="text-sm text-slate-700">开关文字</span>
      <p class="text-xs text-slate-400 mt-0.5">描述说明（可选）</p>
    </div>
    <button type="button" class="toggle" :class="{ 'toggle-on': field }"
            role="switch" :aria-checked="field" @click="field = !field">
      <span class="toggle-thumb" />
    </button>
  </div>
  ```

  使用场景：单个布尔型功能开关，且有描述文字（如 WebSocket 代理、FastCGI、目录浏览、TLS 配置项等）。

- **Checkbox 规范**（多选列表 / 无描述的简单开关）：统一使用以下结构，`<label>` 包裹 checkbox + 文字，描述段落在 `<label>` 外同级；禁止使用独立 `id`/`for` 绑定方式，禁止使用 `accent-*` 或裸 `w-4 h-4` 类（权限多选列表除外）：

  ```html
  <!-- 无描述 -->
  <label class="check-label">
    <input v-model="field" type="checkbox" class="rounded border-slate-300 text-{color}-500 focus:ring-{color}-500" />
    <span class="text-sm text-slate-600">选项文字</span>
  </label>

  <!-- 有描述（描述与 label 同级，mt-1 无缩进） -->
  <div>
    <label class="check-label">
      <input v-model="field" type="checkbox" class="rounded border-slate-300 text-{color}-500 focus:ring-{color}-500" />
      <span class="text-sm text-slate-700">选项文字</span>
    </label>
    <p class="text-xs text-slate-400 mt-1">描述说明</p>
  </div>
  ```

  使用场景：多选列表（插件、权限矩阵）或无描述的简单开关。`{color}` 与页面/资源主色一致。

### 1.12 概览 Widget 统计卡片（强制）

概览页各服务 widget（`overview/widget/*.vue`）中的统计卡片网格列数按元素数量定：

| 元素数 | 响应式 grid 类 | 示例 |
|---|---|---|
| 6 | `grid-cols-2 md:grid-cols-3 lg:grid-cols-6` | Docker、APISIX |
| 4 | `grid-cols-2 md:grid-cols-3 lg:grid-cols-4` | Caddy |
| 其他 | `grid-cols-2 md:grid-cols-3 lg:grid-cols-N`（N=元素数） | 按实际补充 |

- 大屏（`lg:`）下必须一行铺满，禁止多行堆叠
- 卡片统一：`rounded-xl border border-slate-200 bg-white p-4 hover:shadow-md transition-shadow`
- 卡片内容：图标（`.page-icon` + 模块主色）+ 数值（`text-2xl font-bold text-slate-800`）+ 标签（`text-xs text-slate-500 leading-tight`），垂直居中 `flex flex-col items-center gap-2 text-center`
- 加载状态：`flex items-center justify-center py-10` + `spinner`
- 错误/不可用状态：`flex items-center gap-3 py-6 px-4 rounded-xl bg-slate-50`
- `available` 字段由后端返回，`load()` 必须完整赋值所有 `CaddyInfo`/`DockerInfo` 字段，不得遗漏

### 1.13 统一工具与轮询

通用函数复用 `webview/src/helper/utils.ts`；轮询间隔用 `POLL_INTERVAL`，禁止硬编码

### 1.14 import 分组排序

`<script>` 内 import 按以下顺序，组间空一行，组内字母升序：

1. 第三方库  2. `@/stores/...` / `@/stores`  3. `@/router`  4. `@/service/...`  5. `@/helper/...`  6. `@/component/...`  7. `@/views/...`  8. 其余 `@/`  9. 相对路径

同模块普通导入在前、`type` 导入在后紧邻。批量整理：`cd webview && python3 scripts/sort-imports.py --dry-run`，或使用 `npm run format:check` 检查、`npm run format` 修复。

### 1.15 终端能力

系统终端走 `helper/shell.ts`，容器终端走 `helper/container-exec.ts`，禁止页面直接创建 Terminal/WebSocket 实例

### 1.16 暗黑模式样式（强制）

暗黑模式样式统一在 `webview/src/assets/dark.css` 中定义，**禁止在组件中使用 `dark:` 前缀**。

当前暗黑模式主要通过 `.dark` 下覆盖 Tailwind CSS 变量实现：`--color-slate-*`、`--color-primary-*`、`--color-blue-*` 等；浅色主题变量在 `webview/src/assets/light.css` 的 `@theme` 中定义。

**新增颜色时必须同步更新**：

1. 如新增主题色，先在 `light.css @theme` 中补齐浅色变量
2. 在 `dark.css .dark` 中补齐对应暗色变量
3. 仅对变量无法覆盖的特殊类写显式选择器，例如透明度类（`bg-white/80`、`bg-slate-900/60`）、终端/日志区域、CodeMirror 等
4. 显式选择器中包含 `/`、`:`、`group-hover:` 时按 CSS 选择器规则转义

**终端/日志区域**：使用 `bg-slate-900` 保持深色背景，不跟随主题切换

### 1.17 CSS 组件库规范（强制）

所有前端组件样式**优先使用** `webview/src/assets/light_components.css` 中定义的 CSS 类，禁止在同功能场景下使用等价的 inline Tailwind 类。新增样式必须先评估是否应提取到 CSS 文件。

#### 文件结构与类清单

`light_components.css` 包含 8 个模块，按功能分组：

**1. 卡片（Card）**

| 类 | 用途 | 禁止手写 |
|---|---|---|
| `.card` | 页面级主卡片 | `bg-white rounded-2xl shadow-soft border border-slate-200/60` |
| `.card-interactive` | 列表交互式卡片（悬停阴影） | `rounded-xl border border-slate-200 bg-white p-4 hover:shadow-sm` |
| `.card-toolbar` | 页面级卡片顶栏（灰底） | `bg-slate-50 border-b border-slate-200 rounded-t-2xl px-4 md:px-6 py-3` |
| `.card-header` | 小卡片标题栏（widget/模态框内部） | `px-4 py-3 border-b border-slate-100 flex items-center gap-2` |
| `.card-actions` | 移动端卡片底部操作按钮栏 | `flex flex-wrap gap-1.5 pt-2 border-t border-slate-100` |
| `.card-info-row` | 移动端卡片主信息行（图标+文字） | `flex items-center gap-3 min-w-0 flex-1 mb-3` |
| `.option-card` / `.option-card-inactive` / `.option-card-disabled` / `.option-card-icon` | 单选模式/来源选择卡片 | `text-left rounded-xl border p-3 transition-colors`、`w-8 h-8 rounded-lg flex items-center justify-center` |
| `.editor-container` | 代码编辑器/内容外层边框 | `rounded-xl overflow-hidden border border-slate-200` |
| `.modal-card` | 模态框卡片容器 | `bg-white rounded-2xl shadow-2xl border border-slate-200` |

**2. 图标容器（Icon Container）**

| 类 | 尺寸 | 用途 | 禁止手写 |
|---|---|---|---|
| `.page-icon` | 36×36 | Toolbar 模块标识图标（布局） | `w-9 h-9 rounded-lg flex items-center justify-center flex-shrink-0` |
| `.row-icon` | 32×32 | 桌面表格行第一列图标 | `w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0` |
| `.list-icon` | 40×40 | 移动端卡片列表图标 | `w-10 h-10 rounded-lg flex items-center justify-center flex-shrink-0` |
| `.card-icon` | 24×24 | 卡片标题图标容器 | `w-6 h-6 rounded-md flex items-center justify-center` |
| `.empty-state-icon` | 64×64 | 空状态图标容器 | `w-16 h-16 rounded-lg bg-slate-100 flex items-center justify-center mb-4` |

**3. 状态占位（Loading / Empty State）**

| 类 | 用途 |
|---|---|
| `.empty-state` | 加载中 / 空状态块 `flex flex-col items-center justify-center py-20` |
| `.spinner` | 加载动画 `animate-spin rounded-full border-4 border-slate-200 border-t-primary-500` |

**4. 按钮（Button）**

| 类 | 用途 | 禁止手写 |
|---|---|---|
| `.btn` | 基础按钮（inline-flex，焦点&禁用处理） | — |
| `.btn-{color}` | 颜色变体（primary/blue/cyan/indigo/amber/emerald/danger/rose/purple/violet/secondary） | — |
| `.btn-ghost` | 幽灵按钮（透明背景） | — |
| `.btn-icon` | 图标按钮（正方形 padding，工具栏通用） | — |
| `.btn-icon-{color}` | 图标按钮语义色（slate/blue/indigo/violet/cyan/teal/emerald/amber/rose/red） | `text-{color}-600 hover:bg-{color}-50` |
| `.btn-icon-sm` | 小型图标按钮 32×32（关闭/刷新等非行操作） | `w-8 h-8 flex items-center justify-center rounded-lg text-slate-400 hover:text-slate-600 hover:bg-slate-100` |
| `.tab-group` | Tab 切换按钮组容器（整体高度 36px） | `bg-slate-100 p-1 rounded-lg flex items-center gap-0.5` |
| `.tab-btn` | Tab 按钮（含图标，h-7） | `h-7 px-3 text-xs font-medium rounded-md transition-colors flex items-center gap-1.5` |
| `.tab-btn-text` | Tab 按钮（纯文字，h-7） | `h-7 px-3 text-xs font-medium rounded-md transition-colors` |
| `.tab-btn-active` | Tab 激活态（白底+阴影） | `bg-white shadow-sm` |
| `.tab-btn-inactive` | Tab 非激活态 | `text-slate-500 hover:text-slate-700` |
| `.btn-category` / `-active` / `-inactive` | 分类过滤标签按钮 | — |
| `.btn-panel-toggle` / `-active` / `-inactive` | 面板模式切换按钮 | — |
| `.btn-tag-remove` | 多选标签删除按钮（×） | — |

**5. 导航与菜单（Navigation）**

| 类 | 用途 |
|---|---|
| `.nav-link` | 侧边栏导航链接 |
| `.dropdown-item` | 下拉菜单项（普通） |
| `.dropdown-item-danger` | 下拉菜单项（危险/注销） |
| `.breadcrumb-btn` | 面包屑路径按钮 |

**6. 表格（Table）**

| 类 | 用途 | 禁止手写 |
|---|---|---|
| `.th` | th 单元格（左对齐） | `text-left px-4 py-3 text-xs font-semibold text-slate-600 uppercase tracking-wider` |
| `.th-sm` | th 单元格（紧凑型） | `text-left px-3 py-2 text-xs font-semibold text-slate-600 uppercase tracking-wider` |
| `.th-right` | th 单元格（右对齐，操作列） | `text-right px-4 py-3 text-xs font-semibold text-slate-600 uppercase tracking-wider` |

**7. 表单（Form）**

| 类 | 用途 | 禁止手写 |
|---|---|---|
| `.form-label` | 字段 label（灰色大写标题） | `block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1` |
| `.section-title` | 详情页信息分组标题 | `text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3` |
| `.input` | 文本输入框 | `w-full px-4 py-2.5 bg-white border border-slate-200 rounded-xl placeholder:text-slate-400 text-slate-700 hover:border-slate-300` |
| `.select-sm` | 小尺寸 select（toolbar 内使用，固定 `h-9`；基础颜色/边框来自全局 `select`） | `h-9 px-3 pr-8 rounded-md text-xs` |
| `.select-search-header` | 下拉选择器粘性搜索头部 | `px-3 py-2 bg-slate-50 border-b border-slate-100 flex items-center gap-2 sticky top-0 z-10` |
| `.select-list` | 下拉选择器选项列表容器 | `px-2 py-1.5 grid grid-cols-1 gap-0.5 bg-white` |
| `.select-footer` | 下拉选择器底部工具栏 | `px-3 py-2 bg-slate-50 border-t border-slate-100 flex items-center justify-between` |
| `.mobile-search` | 移动端搜索栏容器（桌面端隐藏） | `md:hidden px-4 py-2 border-b border-slate-100` |
| `.check-label` | checkbox/radio label 包裹层 | `flex items-center gap-2 cursor-pointer select-none w-fit` |
| `.toggle-row` / `.toggle` / `.toggle-on` / `.toggle-violet` / `.toggle-thumb` | 功能启用/禁用开关 | — |
| `.text-mono-muted` | 代码/ID 副文字（灰色 mono 截断） | `block font-mono text-xs text-slate-400 truncate mt-0.5` |

**8. 徽章与标签（Badge）**

| 类 | 用途 |
|---|---|
| `.badge` / `.badge-primary` / `.badge-warning` | 通用徽章（`inline-flex items-center px-3 py-1 rounded-lg text-xs font-medium`） |
| `.update-link` | 版本更新跳转链接标签 |

#### 使用规则

- **toolbar 图标与标题**：使用 `.page-icon`（布局）+ 调用方内联背景色，禁止手写等价 inline Tailwind
- **toolbar 下拉框**：使用 `.select-sm`（固定 `h-9`），禁止手写等价的 inline Tailwind
- **toolbar 操作按钮**：使用 `.btn-icon-{color}` 语义类，禁止手写 `text-{color}-600 hover:bg-{color}-50`
- **移动端搜索**：使用 `.mobile-search` 类，禁止手写 `md:hidden px-4 py-2 border-b ...`
- **表格表头**：使用 `.th` / `.th-sm` / `.th-right`，禁止手写等价的 inline Tailwind
- **卡片操作栏（移动端）**：使用 `.card-actions`，`gap-1.5` 不得用 `gap-1`
- **单选模式/来源卡片**：使用 `.option-card` + `.option-card-inactive` / `.option-card-disabled` + `.option-card-icon`
- **Tab 切换**：使用 `.tab-group` + `.tab-btn` / `.tab-btn-text`，整体高度 36px
- **独立详情页 toolbar**：去掉 `<ContainerNav>` Tab 导航，改用独立 `.card-toolbar`（图标+标题在左，操作控件在右），不添加返回按钮
- **header 区域文字按钮统一**：`app.vue` header 中的文字按钮（如工具箱链接、AI 助手）统一使用 `btn btn-ghost px-4 py-2 text-sm gap-2`，禁止个别按钮手写冗余 inline class

#### 禁止

- 在有对应 CSS 类的情况下使用等价的 inline Tailwind 类
- 在组件中使用 `dark:` 前缀（暗黑模式样式统一在 `dark.css`）
- `.btn-icon-sm` 用于表格/移动卡片行级操作（应使用 `.btn-icon-{color}`）

#### 样式自检脚本

`webview/scripts/review-style.py` 是本文件样式规范的辅助检查脚本：当前扫描 `webview/src/views/**/*.vue`，跳过路径中包含 `widget` 的文件；`ERROR` 会返回非零，`WARN` 需要人工处理。脚本不能替代完整人工 review，新增 CSS 组件类或强制规范时需同步更新该脚本。

---

## 2) 前端质量门禁

```bash
npm run lint                             # 类型检查 + ESLint
npm run format:check                     # import 排序/格式检查（dry-run，需关注输出）
python3 scripts/review-style.py          # 前端样式一致性辅助检查
```

`npm run format` 会直接修复 import/ESLint；仅在需要改写文件时使用。`review-style.py` 只有 ERROR 阻断，WARN 需人工确认。
