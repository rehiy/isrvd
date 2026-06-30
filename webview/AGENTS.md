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

### 1.6 页面容器、卡片与标题栏

- 列表/详情页最外层统一使用 `.page`，页面内部独立内容块才使用 `.card`
- 标题栏统一使用 `.page-toolbar` 类（定义于 `light_components.css`），固定在全局 header 下方；容器不写 `flex/justify-between`
- 内嵌面板使用 `.page-toolbar-static` 取消吸顶；配置页右侧目录使用 `.config-section-nav`，仅在桌面且视口高度 ≥ 960px 时启用 `sticky`，矮窗口下保持普通文档流
- 必须提供桌面 `hidden md:flex` 与移动布局；移动端可用 `flex md:hidden`，也可用 `block md:hidden` 外层 + 内部 `flex items-center justify-between`
- 详情页右侧仅保留刷新等功能按钮，**不添加返回按钮**

**toolbar 图标与标题（强制）**：

- 图标：使用 `.page-icon` 类（布局）+ 调用方内联背景色（如 `bg-blue-500`）（禁止手写 `w-9 h-9 rounded-lg flex ...`）
- 标题：`<h1 class="text-lg font-semibold text-slate-800 truncate">`（不得用 `h3`/`h2`/`text-base`）
- 副标题：`<p class="text-xs text-slate-500 truncate">`
- 移动端左侧容器：`flex items-center gap-3 min-w-0 flex-1`，图标加 `flex-shrink-0`，文字容器加 `min-w-0`
- 桌面端右侧按钮区：`flex items-center gap-2 flex-shrink-0`（刷新等功能按钮）
- 移动端右侧按钮区：`flex items-center gap-1 flex-shrink-0`
- 移动端 toolbar 内有多行内容（如标题行 + tab 行）时，行间距用 `mb-3`；移动端 tab 行用 `mt-3`
- 桌面端 toolbar 左侧标题区：`flex items-center gap-3`（图标 + 标题文字块）

**图标样式（强制）**：

- 形状：`rounded-lg`（禁止 `rounded-full`/`rounded-2xl`）
- 尺寸：使用对应 CSS 类——`.empty-state-icon`（空状态/登录 64×64）、`.page-icon`（toolbar 36×36）、`.list-icon`（移动卡片 40×40）、`.row-icon`（桌面表格 32×32）、`.card-icon`（卡片标题 24×24）

### 1.7 列表双视图与搜索（强制）

- 桌面：`.card-table hidden md:block` + `<table>`（表格直接贴边，无内边距，`overflow-x-auto` 由 `.card-table` 提供）
- 移动：`.card-body md:hidden space-y-3` + `.card-interactive` 卡片列表
- loading / 空状态：`.card-body`（带内边距）包裹 `.empty-state`
- 列表有数据时用 `<template v-else>` 包裹桌面和移动两个分支，**不得用 `<div v-else>`**（减少无意义 DOM 层级）

**标准列表页结构**：

```html
<!-- Loading -->
<div v-if="loading" class="card-body">
  <div class="empty-state">...</div>
</div>

<!-- 空状态 -->
<div v-else-if="list.length === 0" class="card-body">
  <div class="empty-state">...</div>
</div>

<!-- 列表 -->
<template v-else>
  <div class="card-table hidden md:block">
    <table>...</table>
  </div>
  <div class="card-body md:hidden space-y-3">
    ...移动端卡片...
  </div>
</template>
```
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

图标配色跟随资源页主色（图标背景通常用 `400` 色阶，toolbar 用 `500` 色阶）：Docker 容器 `emerald/slate`（按状态）、镜像 `blue`、网络 `purple`、数据卷 `amber`、仓库 `purple`；Swarm 节点 `blue`、服务 `emerald`、任务 `cyan`；APISIX 路由 `indigo`、上游 `emerald`、消费者 `violet`、SSL `cyan`、插件配置 `rose`、访问授权 `amber`；Caddy 路由 `indigo`、证书 `cyan`、全局配置 `violet`、原始配置 `slate`；Compose `amber`、Cron `amber/violet`、用户 `blue`、Filer `primary`。新模块选未用色（`rose`/`cyan`/`lime` 等）

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

**间距规范（强制）**：

| 场景 | 类 |
|---|---|
| 表单字段间距 | `space-y-4` |
| 表单分组（section）间距 | `space-y-6`，或 `divide-y divide-slate-100` + 每组 `py-6`（首组 `pb-6`，末组 `pt-6`） |
| 表单内嵌子分组分隔线 | `border-t border-slate-200 pt-6`，分组标题 `mb-4` |
| 表单保存/操作按钮区 | `mt-6 pt-4 border-t border-slate-200` |
| 表单提交按钮与说明文字 | `flex items-center gap-3` 或 `flex flex-col sm:flex-row sm:items-center gap-3` |
| **card-body 内容区（详情/图表/通用内容）** | `card-body space-y-4` |
| **card-body 移动端卡片列表** | `card-body md:hidden space-y-3` |
| **card-body 日志/紧凑内容区** | `card-body space-y-3` |

> `space-y-6` 仅用于表单分组（section）间距，**禁止**用于 `card-body` 内容区（详情页等）。

- 表单容器 `max-w-3xl space-y-4`
- label：`block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1`，input 通用 `.input`，help `text-xs text-slate-400 mt-1`
- 密钥/密码：后端敏感字段 `json:"-"`，前端 `type="password" autocomplete="new-password"`，留空保存=不修改，placeholder："留空保持不变"

- **Toggle Switch 规范**（单一功能启用/禁用开关）：统一使用 `<ToggleCard>` 组件（`webview/src/component/toggle-card.vue`），禁止手写内联 Tailwind toggle 样式或直接使用底层 CSS 类：

  | Prop | 类型 | 说明 |
  |---|---|---|
  | `v-model` | `boolean` | 开关状态 |
  | `label` | `string`（必填） | 开关标题 |
  | `desc` | `string` | 描述文字（可选，也可用 `#desc` slot 传入富文本） |
  | `violet` | `boolean` | 激活色改为 violet（如 Caddy Global 场景） |
  | `disabled` | `boolean` | 禁用状态 |

  ```html
  <!-- 无 body：普通 toggle 行 -->
  <ToggleCard v-model="field" label="开关文字" desc="描述说明（可选）" />

  <!-- 有 body：展开时变 card，收缩时仍是普通 toggle 行 -->
  <ToggleCard v-model="field" label="开关文字">
    <!-- 展开后显示的内容 -->
  </ToggleCard>

  <!-- desc 含 HTML 时用 #desc slot -->
  <ToggleCard v-model="field" label="开关文字">
    <template #desc>描述 <code>示例</code></template>
  </ToggleCard>
  ```

  使用场景：单个布尔型功能开关（如 WebSocket 代理、FastCGI、目录浏览、TLS 配置项等）；有子配置项时传入 default slot，展开后自动呈现 card 样式。

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

通用函数按职责复用 `webview/src/helper/format.ts`、`file.ts`、`dom.ts` 等细分模块；轮询间隔使用 `helper/format.ts` 的 `POLL_INTERVAL`，禁止硬编码

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
4. 显式选择器中包含 `/`、`:`、`group-hover:` 时按 CSS 选择器规则转义；透明白底类如 `bg-white/80`、`bg-white/95` 必须在 `dark.css` 中显式覆盖

**终端/日志区域**：使用 `bg-slate-900` 保持深色背景，不跟随主题切换

### 1.17 CSS 组件库规范（强制）

所有前端组件样式**优先使用** `webview/src/assets/light_components.css` 中定义的 CSS 类，禁止在同功能场景下使用等价的 inline Tailwind 类。新增样式必须先评估是否应提取到 CSS 文件。

#### 文件结构与类清单

`light_components.css` 包含 10 个模块，按功能分组：

**1. 卡片（Card）**

| 类 | 用途 | 禁止手写 |
|---|---|---|
| `.page` | 页面级全宽内容容器 | `min-w-0` |
| `.page-toolbar` | 页面标题栏，吸附于全局 header 下方 | `sticky top-16 z-30 bg-slate-100/80 backdrop-blur-xl border-b border-slate-200/70 px-4 py-3` |
| `.page-toolbar-static` | 内嵌面板工具栏修饰类，取消吸顶并防止收缩 | `position: static; z-index: auto; flex-shrink: 0` |
| `.card` | 页面内部内容卡片 | `rounded-xl shadow-soft border border-slate-200/60 overflow-hidden` |
| `.card-interactive` | 列表交互式卡片（悬停阴影） | `rounded-xl border border-slate-200 p-4 hover:shadow-sm` |
| `.card-header` | 小卡片标题栏（widget/模态框内部） | `px-4 py-3 border-b border-slate-100 flex items-center gap-2` |
| `.card-body` | 卡片内容区（表单/详情/loading/empty/移动端卡片列表），带内边距 | `p-4` |
| `.card-table` | 卡片表格区（桌面端表格），无内边距，表格直接贴边 | `overflow-x-auto` |
| `.card-actions` | 移动端卡片底部操作按钮栏 | `flex flex-wrap gap-1.5 pt-2 border-t border-slate-100` |
| `.card-info-row` | 移动端卡片主信息行（图标+文字） | `flex items-center gap-3 min-w-0 flex-1 mb-3` |
| `.title-text` | 通用标题文字 | `text-lg font-semibold text-slate-800 truncate` |
| `.toolbar-desktop` / `.toolbar-mobile` | toolbar 桌面/移动外层 | `hidden md:flex items-center justify-between`、`flex md:hidden items-center justify-between` |
| `.title-group` / `.title-group-static` / `.inline-info` | 图标 + 文本信息组 | `flex items-center gap-3 min-w-0 flex-1`、`flex items-center gap-3 min-w-0`、`flex items-center gap-2 min-w-0` |
| `.action-group-desktop` / `.action-group` / `.action-group-sm` | 操作按钮组 | `hidden md:flex items-center gap-2 flex-shrink-0`、`flex items-center gap-2 flex-shrink-0`、`flex items-center gap-1 flex-shrink-0` |
| `.item-title` / `.item-title-sm` / `.item-subtitle` / `.item-subtitle-mono` / `.prop-label-start` | 列表/表格主副标题与属性标签 | 对应常用标题、副标题、mono 副标题、起始属性标签样式 |
| `.card-prop-row` | 移动端卡片属性行（标签+纯文本值） | `flex items-center gap-2 mb-3` |
| `.card-prop-row-start` | 移动端卡片属性行（标签+badge/code，值换行对齐） | `flex items-start gap-2 mb-3` |
| `.detail-value` | 详情页属性值容器（灰色圆角背景块） | `px-3 py-2 bg-slate-50 rounded-lg` |
| `.detail-value-mono` | 详情页属性值容器（等宽字体，用于 ID/代码/路径） | `block px-3 py-2 bg-slate-50 rounded-lg text-xs font-mono text-slate-700 break-all` |
| `.option-card` / `.option-card-inactive` / `.option-card-disabled` / `.option-card-icon` | 单选模式/来源选择卡片 | `text-left rounded-xl border p-3 transition-colors`、`w-8 h-8 rounded-lg flex items-center justify-center` |
| `.editor-container` | 代码编辑器/内容外层边框 | `rounded-xl overflow-hidden border border-slate-200` |
| `.modal-card` | 模态框卡片容器 | `bg-white rounded-xl shadow-2xl border border-slate-200` |

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
| `.spinner` | 加载动画基础样式 `animate-spin rounded-full border-4 border-slate-200 border-t-primary-500` |
| `.spinner-lg` / `.spinner-md` | 加载动画尺寸：`.spinner-lg` 为 `w-12 h-12 mb-3` + spinner 基础样式；`.spinner-md` 为 `w-8 h-8 mr-2` + spinner 基础样式 |

**4. 按钮（Button）**

| 类 | 用途 | 禁止手写 |
|---|---|---|
| `.btn` | 基础按钮（inline-flex，焦点&禁用处理） | — |
| `.btn-square` | toolbar 中仅显示图标的 36×36 方形按钮，需与 `.btn` + 颜色类组合使用 | `w-9 h-9 !px-0` |
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
| `.btn-proto` / `-active` / `-inactive` | 协议/枚举选择按钮（如 不限/HTTP/HTTPS） | `px-3 py-2 rounded-lg text-sm font-medium border transition-colors` |
| `.btn-add-row` | 虚线添加行按钮（表格式编辑区底部） | `w-full flex items-center justify-center gap-2 py-3 rounded-lg border border-dashed ...` |

**5. 导航与菜单（Navigation）**

| 类 | 用途 |
|---|---|
| `.nav-link` | 侧边栏导航链接 |
| `.nav-link-active` | 侧边栏导航激活态 |
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
| `.section-title` | 详情页信息分组标题 | `text-sm font-semibold text-slate-700 mb-3 pb-2 border-b border-slate-200` |
| `.section-title-table` | 后面紧跟表格时（去掉 margin，横线充当表格顶边框） | `mb-0` |
| `.input` | 文本输入框 | `w-full px-4 py-2.5 bg-white border border-slate-200 rounded-xl placeholder:text-slate-400 text-slate-700 hover:border-slate-300` |
| `.input + .btn` | input 后紧跟的按钮等高处理 | `height: 46px` |
| `select` | select 基础样式（自定义箭头、边框交互） | `bg-white border border-slate-200 text-slate-700 appearance-none truncate hover:border-slate-300` |
| `select.input` | select 表单尺寸（配合 `.input` 使用） | `pr-10 rounded-xl`，固定高度 46px |
| `.select-sm` | 小尺寸 select（toolbar 内使用，固定 `h-9`） | `h-9 px-3 pr-8 rounded-md text-xs` |
| `.select-search-header` | 下拉选择器粘性搜索头部 | `px-3 py-2 bg-slate-50 border-b border-slate-100 flex items-center gap-2 sticky top-0 z-10` |
| `.select-list` | 下拉选择器选项列表容器 | `px-2 py-1.5 grid grid-cols-1 gap-0.5 bg-white` |
| `.select-footer` | 下拉选择器底部工具栏 | `px-3 py-2 bg-slate-50 border-t border-slate-100 flex items-center justify-between` |
| `.mobile-search` | 移动端搜索栏容器（桌面端隐藏） | `md:hidden px-4 py-2 border-b border-slate-100` |
| `.check-label` | checkbox/radio label 包裹层 | `flex items-center gap-2 cursor-pointer select-none w-fit` |
| `.toggle-row` / `.toggle` / `.toggle-on` / `.toggle-violet` / `.toggle-thumb` | 功能启用/禁用开关底层样式（由 `<ToggleCard>` 组件封装，禁止直接使用） | — |
| `.toggle-expand-card` / `.toggle-expand-card-open` / `.toggle-expand-body` | 展开式 toggle card 底层样式（由 `<ToggleCard>` 传入 default slot 时自动使用，禁止直接使用） | — |
| `.text-mono-muted` | 代码/ID 副文字（灰色 mono 截断） | `block font-mono text-xs text-slate-400 truncate mt-0.5` |
| `.code-chip` | 小号等宽代码片段 | `inline-block text-xs px-2 py-0.5 rounded-lg font-mono break-all` |
| `.panel-frame` | 轻量边框面板 | `rounded-xl border border-slate-200 overflow-hidden` |
| `.empty-note` | 紧凑空内容提示 | `text-sm text-slate-400 py-6 text-center bg-slate-50 rounded-xl` |

**8. 徽章与标签（Badge）**

| 类 | 用途 | 禁止手写 |
|---|---|---|
| `.badge` / `.badge-sm` / `.badge-xs` / `.badge-primary` / `.badge-warning` / `.badge-success` / `.badge-muted` | 通用徽章 | `.badge`: `inline-flex items-center px-3 py-1 rounded-lg text-xs font-medium`; `.badge-sm`: `inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-medium`; `.badge-xs`: `inline-flex items-center px-1.5 py-0.5 rounded-lg text-xs font-medium` |

**9. 概览统计卡片（Stat Card）**

| 类 | 用途 | 禁止手写 |
|---|---|---|
| `.stat-card` | 概览 Widget 统计卡片（规范 1.12） | `rounded-xl border border-slate-200 p-4 hover:shadow-md transition-shadow flex items-center gap-3` |
| `.overview-loading` / `.overview-unavailable` | 概览 Widget 加载/不可用状态 | `flex items-center justify-center py-10`、`flex items-center gap-3 py-6 px-4 rounded-xl bg-slate-50` |
| `.monitor-chart-box` / `.monitor-legend-item` / `.metric-card` / `.metric-legend` | 监控图表与指标块 | 对应监控图表容器、图例项、指标卡片和指标图例样式 |

**10. 终端页面（Terminal Page）**

| 类 | 用途 | 禁止手写 |
|---|---|---|
| `.terminal-pane` | 终端黑色面板（固定深色背景，不跟随主题切换） | `flex-1 p-2 md:p-3 overflow-hidden`，背景 `#0f172a` |

#### 使用规则

- **toolbar 图标与标题**：使用 `.page-icon`（布局）+ 调用方内联背景色，禁止手写等价 inline Tailwind
- **toolbar 下拉框**：使用 `.select-sm`（固定 `h-9`），禁止手写等价的 inline Tailwind
- **toolbar 操作按钮**：使用 `.btn-icon-{color}` 语义类，禁止手写 `text-{color}-600 hover:bg-{color}-50`
- **移动端搜索**：使用 `.mobile-search` 类，禁止手写 `md:hidden px-4 py-2 border-b ...`
- **表格表头**：使用 `.th` / `.th-sm` / `.th-right`，禁止手写等价的 inline Tailwind
- **卡片操作栏（移动端）**：使用 `.card-actions`，`gap-1.5` 不得用 `gap-1`
- **单选模式/来源卡片**：使用 `.option-card` + `.option-card-inactive` / `.option-card-disabled` + `.option-card-icon`
- **Tab 切换**：使用 `.tab-group` + `.tab-btn` / `.tab-btn-text`，整体高度 36px
- **协议/枚举选择按钮**：使用 `.btn-proto` + `.btn-proto-active` / `.btn-proto-inactive`
- **虚线添加行按钮**：使用 `.btn-add-row`（表格式编辑区底部）
- **概览统计卡片**：使用 `.stat-card`（规范 1.12）
- **独立详情页 toolbar**：去掉 `<ContainerNav>` Tab 导航，改用独立 `.page-toolbar`（图标+标题在左，操作控件在右），不添加返回按钮
- **header 区域文字按钮统一**：`app.vue` header 中的文字按钮（如工具箱链接、AI 助手）统一使用 `btn btn-ghost px-4 py-2 text-sm gap-2`，禁止个别按钮手写冗余 inline class
- **`.section-title` 强制规范**：
  - 文字颜色 `text-slate-700`、字号 `text-sm`、加底部边框 `border-b border-slate-200 pb-2`
  - 后面紧跟表格时，追加 `section-title-table` 类（去掉 margin，横线直接紧贴表格）
  - 禁止手写 `text-xs ... text-slate-500` 等过时写法

#### 禁止

- 在有对应 CSS 类的情况下使用等价的 inline Tailwind 类
- 在组件中使用 `dark:` 前缀（暗黑模式样式统一在 `dark.css`）
- `.btn-icon-sm` 用于表格/移动卡片行级操作（应使用 `.btn-icon-{color}`）
- `.section-title` 后紧跟表格时，手写 `border-b` 或 `mb-3`（应使用 `.section-title-table`）

#### 样式自检脚本

`webview/scripts/review-style.py` 是本文件样式规范的辅助检查脚本：当前扫描 `webview/src/views/**/*.vue` 与 `webview/src/component/**/*.vue`；`ERROR` 会返回非零，`WARN` 需要人工处理。脚本不能替代完整人工 review，新增 CSS 组件类或强制规范时需同步更新该脚本。

---

## 1.18 Explorer 与 SFTP 强一致性规范

Explorer（文件管理器）与 SFTP（SSH 文件传输）是两个功能相似的文件管理模块，**必须保持强一致性**，包括样式、交互、逻辑三个层面。

### 1.18.1 组件结构一致性

| 要求 | 说明 |
|---|---|
| 模态框组件 | 必须使用 `BaseModal` 组件，通过 `expose: ['show']` 暴露方法 |
| 模态框引用 | 必须使用 `ref="modalRef"`，并通过 `@Ref` 获取引用 |
| 模态框状态 | 使用局部 `loading` 状态（非全局 `portal` 状态），通过 `:loading="loading"` 绑定 |
| 成功回调 | 通过 `this.$emit('success')` 通知父组件刷新，而非直接调用全局方法 |

### 1.18.2 样式一致性

| 要求 | 说明 |
|---|---|
| 按钮配色 | 必须遵循 **1.10 操作按钮语义色**规范，且两个模块配色完全一致 |
| 图标使用 | 相同功能的按钮必须使用相同的图标（如权限都用 `fa-key`） |
| CSS 类 | 使用相同的 CSS 类（如 `btn-icon-slate`、`input`、`form-label` 等） |
| 布局结构 | 模态框内部布局、表单结构、提示信息样式必须一致 |

### 1.18.3 交互一致性

| 要求 | 说明 |
|---|---|
| 按钮文本 | 动态按钮文本格式一致（如 `"保存中..."` / `"保存文件"`） |
| 禁用逻辑 | 相同场景的禁用逻辑一致（如 `:disabled="!formData.newName.trim()"`） |
| 通知提示 | 成功/失败通知格式一致（如 `"文件保存成功"`、`"权限修改失败: " + error`） |
| 路径处理 | 文件路径构建逻辑一致（都使用 `basePath + '/' + file.name` 格式） |

### 1.18.4 配色对照表

| 操作 | 配色 | Explorer | SFTP |
|---|---|---|---|
| 编辑/重命名 | `blue` | `btn-icon-blue` | `btn-icon-blue` |
| 权限查看 | `slate` | `btn-icon-slate` | `btn-icon-slate` |
| 删除 | `red` | `btn-icon-red` | `btn-icon-red` |
| 下载/预览/进入 | `slate` | `btn-icon-slate` | `btn-icon-slate` |
| 压缩/解压 | `amber` | `btn-icon-amber` | - |
| 上传 | `teal` | - | `btn-icon-teal` |

### 1.18.5 检查清单

在修改 Explorer 或 SFTP 相关代码时，必须检查：

- [ ] 模态框组件结构是否与对方一致（`BaseModal`、`expose`、`ref`）
- [ ] 按钮配色是否与对方一致（参考 1.18.4 配色对照表）
- [ ] 图标是否与对方一致（相同功能用相同图标）
- [ ] 表单样式是否与对方一致（相同的 CSS 类）
- [ ] 按钮文本动态显示是否一致（"操作中..." / "操作名"）
- [ ] 成功/失败通知格式是否一致
- [ ] 路径构建逻辑是否一致

**违反强一致性的代码将被视为不规范，需要在合并前修复。**

---

## 2) 前端质量门禁

```bash
npm run lint                             # 类型检查 + ESLint
npm run format:check                     # import 排序/格式检查（dry-run，需关注输出）
python3 scripts/review-style.py          # 前端样式一致性辅助检查
```

`npm run format` 会直接修复 import/ESLint；仅在需要改写文件时使用。`review-style.py` 只有 ERROR 阻断，WARN 需人工确认。
