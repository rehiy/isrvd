# AGENTS.md — isrvd Agent Operating Guide

> 本文件定义 AI 代理在 `isrvd` 仓库中的统一工作方式。
> 目标：可执行、可验证、低歧义。

---

## 1) 指令优先级（必须遵守）

当规则冲突时，按以下顺序执行：

1. **用户当前明确需求**
2. **安全与稳定性约束**
3. **本文件规范（AGENTS.md）**
4. **现有代码风格与目录约定**

如果规则冲突且无法自动化消解，优先保证：

- 不泄露敏感信息
- 不引入破坏性变更
- 不修改需求范围外逻辑

---

## 2) 工作流（标准执行流程）

### 2.1 先理解再改动

- 先定位相关模块、调用链、类型定义与边界条件。
- 变更前确认：入口、依赖方向、数据结构、错误处理方式。
- 不基于猜测修改。

### 2.2 小步提交

- 优先最小可行改动（small diff）。
- 每一步都应能解释"为什么改、改了什么、如何验证"。

### 2.3 变更后验证

- 至少执行与改动相关的静态检查或最小验证。
- 若无法执行完整验证，明确说明已做的检查与未覆盖风险。

---

## 3) 项目架构约束（必须遵守）

### 3.1 分层职责

- `pkgs/`：原生客户端层，返回 SDK/底层原生类型。
- `internal/service/`：业务组合、类型转换、参数校验。
- `internal/server/`：HTTP 入口与轻量 handler（解析请求、调用 service、返回响应）。
- `internal/registry/`：外部服务初始化、生命周期管理与可用性检查。

依赖方向：

`config -> internal/registry -> pkgs -> internal/service -> internal/server`

### 3.2 明确禁止

- `pkgs/` 依赖 `internal/`。
- 在 handler 中堆叠业务逻辑。
- 在 service/handler 中直接从配置创建外部客户端（应经 `registry`）。

### 3.3 高内聚、低耦合

- **内聚**：同一领域的功能与数据放在同一包/文件中，避免跨包拆散。
  - `pkgs/docker/` 聚合所有 Docker 操作，`pkgs/swarm/` 聚合所有 Swarm 操作。
  - 类型就近定义在对应文件，不集中到 `types.go`。
  - 前端 `service/types/` 按域拆分（`docker.ts`、`swarm.ts`、`apisix.ts`…）。
- **耦合**：层与层之间通过接口或注入解耦，禁止跨层直达。
  - `pkgs/` 不依赖 `internal/`，配置通过构造函数注入。
  - `service` 不感知 HTTP 框架，只做业务组合与类型转换。
  - `handler` 不直接创建外部客户端，统一从 `registry` 获取。
  - 前端组件通过 `Provide/Inject` 获取全局状态，不直接 import store 实例。
- **判定标准**：改动一个功能时，只需改一处代码、一个包、一个文件。如果需要同时改多个包的同名函数或平行结构，说明内聚不足或耦合过紧。

---

## 4) 后端编码规范（Go）

### 4.1 HTTP 与响应

- 状态码使用 `net/http` 常量，禁止硬编码数字。
- 成功返回统一 `helper.RespondSuccess`。
- 失败返回统一 `helper.RespondError`。
- 参数绑定优先 `ShouldBindJSON/ShouldBindQuery/ShouldBindURI`。
- 绑定失败返回 `err.Error()`，不要用固定错误文案覆盖细节。

### 4.2 WebSocket

- 统一使用 `helper.WsUpgrader`。
- 不在 handler 中定义私有 upgrader。

### 4.3 错误处理

- 禁止 `fmt.Errorf(err.Error())`。
- 使用 `fmt.Errorf("...: %w", err)` 包装内部错误。
- `pkgs` 调外部 SDK/API 失败时应透传原始错误；仅在本层自有逻辑失败时加上下文。

### 4.4 命名与日志

- 缩写全大写：`CPU`、`ID`、`URL`、`HTTP`。
- 日志统一 `logman`，禁止 `log.Println` / `fmt.Println`。
- 日志使用键值对，不拼接字符串。

### 4.5 类型定义

- 请求/响应结构体：定义在对应 handler 子包。
- 业务模型：就近定义在对应 `pkgs` 文件（不要集中 `types.go`）。
- 避免跨包重复定义语义相同结构体。

### 4.6 配置结构体命名

- 顶层：`Config`（`config/types.go`）
- 子配置：`Server`、`AgentConfig`、`ApisixConfig`、`DockerConfig`、`MarketplaceConfig`、`MemberConfig`
- 镜像仓库：`DockerRegistry`（含 `Name`、`URL`、`Username`、`Password`、`Description`）
- 结构体字段使用指针类型 `*` 和 YAML 标签（如 `yaml:"jwtSecret"`）

---

## 5) 前端编码规范（Vue/Tailwind）

适用目录：`webview/src`

### 5.1 组件装饰器模式

- 使用 `vue-facing-decorator`：`@Component`、`@Inject`、`@Prop`、`@Ref`、`@Watch`、`@Provide`
- 组件类必须用 `toNative()` 包装后导出（如 `export default toNative(App)`）
- `@Inject` 从 `APP_STATE_KEY` / `APP_ACTIONS_KEY` 注入状态和操作

### 5.2 状态管理

- 全局状态：`store/state.ts` 使用 `reactive` + Provide/Inject 模式
- Provide/Inject 键：`APP_STATE_KEY`（`app.state`）、`APP_ACTIONS_KEY`（`app.actions`）
- 全局权限状态 `permState`（`reactive`）：供路由守卫使用，避免循环依赖
  - `loaded`、`isPrimary`、`permissions`
- 初始化函数：`initProvider()` 返回 `{ state, actions }`

### 5.3 API 服务层

- `service/api.ts`：单例 `class ApiService`，`export default new ApiService()`
- 所有 API 请求统一通过 `http` / `httpBlob` 封装（`service/axios.ts`）
- `http`：类型安全的 HTTP 客户端（响应拦截器已解包为 `APIResponse`）
- `httpBlob`：Blob 下载专用（`responseType: 'blob'` 时拦截器解包后直接是 Blob）

### 5.4 类型定义组织

- `service/types/` 按域拆分：`docker.ts`、`swarm.ts`、`apisix.ts`、`compose.ts`、`system.ts`、`auth.ts`、`common.ts`、`filer.ts`
- `service/types.ts` 统一 `export * from './types/...'` 导出
- Swarm 服务列表 DTO 使用 `SwarmServiceInfo`（避免与 `SwarmService` 管理器同名）

### 5.5 卡片与标题栏

- 列表页/详情页统一 `.card mb-4`。
- 标题栏容器固定：

`bg-slate-50 border-b border-slate-200 rounded-t-2xl px-4 md:px-6 py-3`

- 容器本身不写 `flex/justify-between`。
- 必须同时提供桌面与移动双布局：`hidden md:flex` / `flex md:hidden`。
- 详情页标题栏与列表页结构相同，右侧仅保留刷新等功能按钮。**不添加返回按钮**，用户通过侧边栏菜单导航。

### 5.6 列表双视图（强制）

- 桌面：`hidden md:block overflow-x-auto` + `<table>`
- 移动：`md:hidden space-y-3 p-4` + 卡片列表
- **移动外层 `p-4` 不得省略**。
- 移动端子卡片：`rounded-xl border border-slate-200 bg-white p-4`

### 5.7 操作按钮语义色

- 创建/启动：`emerald`
- 编辑/重启：`blue`
- 网络/扩缩容：`indigo`
- 停止/告警：`amber`
- 删除：`red`
- 中性：`slate`
- 日志：`slate`
- 统计：`indigo`
- 终端：`teal`

按钮统一使用 `btn-icon`（如该页面已有约定）。

### 5.8 表单与敏感字段

- 表单容器：`max-w-3xl space-y-4`
- 字段结构：`label + input(.input) + help text`
  - label：`block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1`
  - input：通用 `.input` 样式
  - help text：`text-xs text-slate-400 mt-1`
- 密钥/密码类字段：
  - 后端仅返回 `xxxSet`，不回传明文
  - 前端 `type="password" autocomplete="new-password"`
  - 留空保存表示"不修改已有值"
  - placeholder 动态提示："已设置（留空保持不变）" / "尚未设置"

### 5.9 统一工具与轮询

- 通用工具函数复用 `webview/src/helper/utils.ts`。
- 轮询间隔使用 `POLL_INTERVAL`，禁止硬编码毫秒值。

### 5.10 终端能力

- 系统终端统一走 `helper/shell.ts`
- 容器终端统一走 `helper/container-exec.ts`
- 禁止在页面直接创建 Terminal / WebSocket 实例

---

## 6) 路由与导航约束

- `/overview`：整体概览
- `docker/overview` 与 `swarm/overview` 仅作为组件，不作为独立菜单路由
- 设置模块拆分：
  - `/system/settings`
  - `/system/members`
- 侧边栏"系统设置"位于最后
- 首个系统账号禁止删除（前后端双重保护）

### 6.1 导航折叠子菜单

- 折叠状态：子菜单展开状态自动跟随当前路由（`@Watch` immediate）
- 侧边栏折叠：宽度 `w-16`（折叠）→ `w-64`（展开）
- 移动端：遮罩层 + 抽屉式侧边栏（`-translate-x-full lg:translate-x-0`）
- 移动端菜单切换：`toggleMobileSidebar` / `closeMobileSidebar` / `openMobileSidebar`
- 窗口 `resize` 事件：宽度 ≥ 1024px 时自动关闭移动侧边栏

---

## 7) 注册中心与服务初始化

- 启动顺序：`main -> config.Load -> registry.Init -> server.NewApp`
- 统一通过 `internal/registry/` 管理外部依赖
- 可用性检查：
  - `IsDockerAvailable`
  - `IsSwarmAvailable`
  - `IsApisixAvailable`

已使用命名：`registry.DockerService`、`registry.SwarmService`、`registry.ApisixClient`

---

## 8) 安全基线（必须遵守）

1. 禁止硬编码密钥、密码、令牌。
2. 敏感配置仅返回 `xxxSet` 布尔值。
3. 文件系统操作必须防目录遍历。
4. 解压必须防 Zip Slip。
5. WebSocket 必须经过认证链路。
6. 关键资源（首个系统账号、内置角色等）前后端双重校验。

---

## 9) 质量门禁（提交前自检）

至少完成：

- [ ] 编译通过（后端/前端受影响部分）
- [ ] 无新增 lint 警告或错误
- [ ] 关键路径手动验证（最小闭环）
- [ ] 错误处理与日志符合规范
- [ ] 未引入明文敏感信息

如无法完整执行，需在结果中说明原因与风险。

---

## 10) Git 约定

### 10.1 提交信息

格式：

`<type>: <subject>`

类型：`feat`、`fix`、`refactor`、`style`、`docs`、`chore`

### 10.2 分支

- `main`：生产
- `dev`：开发
- `feature/<name>`：功能
- `fix/<name>`：修复

---

本规范适用于本仓库所有 AI 代理协作与代码改动。
如与用户当次明确需求冲突，按"指令优先级"处理，并在输出中说明取舍。
