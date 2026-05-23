# AGENTS.md — isrvd Agent 操作指南

> 本文件是 `isrvd` 仓库唯一代码规范入口。目标：可执行、可验证、低歧义。

---

## 1) 指令优先级

冲突时：用户明确需求 → 安全稳定性 → 本规范 → 现有代码风格
无法消解时：不泄露敏感信息、不引入破坏性变更、不修改需求范围外逻辑

---

## 2) 工作流

- **先理解再改动**：定位模块、调用链、类型与边界，不基于猜测修改
- **小步提交**：最小可行改动，每步可解释"为什么改、改了什么、如何验证"
- **变更后验证**：执行相关静态检查；无法完整验证时说明风险
- **同步更新文档与 Skills**：修改代码时，必须同步更新所有相关的说明文档和 skills 文件，确保文档与代码始终一致

---

## 3) 项目架构

### 分层依赖方向

`config → internal/registry → pkgs → internal/service → internal/server`

- `pkgs/`：原生客户端层，返回 SDK/底层类型
- `internal/service/`：业务组合、类型转换、参数校验
- `internal/server/`：HTTP 入口，解析请求 → 调用 service → 返回响应
- `internal/registry/`：外部服务初始化、生命周期管理与可用性检查

### 禁止

- `pkgs/` 依赖 `internal/`
- `handler` 中堆叠业务逻辑
- `service/handler` 直接从配置创建外部客户端

### 内聚与耦合

- 同领域功能聚合同包（`pkgs/docker/`、`pkgs/swarm/`），类型就近定义，不集中 `types.go`
- 层间通过接口或注入解耦；前端组件通过 `Provide/Inject` 获取全局状态
- 判定：改一个功能只需改一处；若需同时改多个包同名函数，说明内聚不足

---

## 4) 代码变更同步文档规范

修改代码时，必须同步更新所有相关的说明文档和 skills 文件，确保文档与代码始终一致。

### 适用范围

所有涉及以下变更的场景：

- 新增/修改/删除 API 路由或参数
- 新增/修改/删除 数据结构（Request/Response 字段）
- 新增/修改/删除 业务逻辑或工作流
- 新增/修改/删除 配置项或环境变量
- 新增/修改/删除 Shell 脚本中的命令或参数

### Skill 文件结构

```
skills/isrvd/
├── SKILL.md                      ← 索引 + 决策树 + 常见工作流
├── scripts/api.sh                ← 认证持久化 + API 调用封装
└── docs/
    ├── docker/{containers,images,networks,volumes,registries}.md
    ├── swarm/{info,services,tasks}.md
    ├── compose.md
    ├── apisix/{routes,upstreams,consumers,ssl}.md
    └── system/{overview,config,account,filer,cron}.md
```

### 需要同步更新的文件

| 代码变更位置 | 需同步更新的文档 |
|---|---|
| `internal/server/ctrl_docker.go` | `skills/isrvd/docs/docker/` 下对应资源文件 |
| `internal/server/ctrl_swarm.go` | `skills/isrvd/docs/swarm/` 下对应资源文件 |
| `internal/server/ctrl_apisix.go` | `skills/isrvd/docs/apisix/` 下对应资源文件 |
| `internal/server/ctrl_compose.go` | `skills/isrvd/docs/compose.md` |
| `internal/server/ctrl_cron.go` | `skills/isrvd/docs/system/cron.md` |
| `internal/server/ctrl_system.go` / `ctrl_account.go` | `skills/isrvd/docs/system/` 下对应文件 |
| `pkgs/*/`（数据结构变更） | 对应 docs 文件中的字段表 |
| 新增路由/模块 | `skills/isrvd/SKILL.md` 索引表 + 决策树 |
| 脚本相关变更 | `skills/isrvd/scripts/api.sh` |

### 执行步骤

1. **识别影响范围**：修改代码后，定位对应的 docs 文件（按模块+资源查找）
2. **同步更新文档**：
   - API 变更 → 更新对应 docs 文件的 bash 用法、请求体、响应字段
   - 数据结构变更 → 更新字段表，确保与 Go struct json tag 一致
   - 新增路由 → 在 `SKILL.md` 索引表和决策树中添加条目
3. **验证格式**：所有 API 必须以 `isrvd_get/post/put/patch/delete/upload` bash 用法为主要表达方式，不得只写 HTTP 格式

### 注意事项

- 文档中标注"只读"的字段（如 `create_time`、`update_time`、`id`）也必须列出
- 请求体和响应体中的字段必须与 Go struct 的 json tag 完全一致
- 路由路径必须与 `ctrl_*.go` 中的注册路径完全一致
- 每个 API 端点必须有对应的 bash 示例，不能只写 `GET /api/...` HTTP 格式

---

## 5) 后端编码规范（Go）

### HTTP 与响应

- 状态码用 `net/http` 常量；成功 `httpd.RespondSuccess`，失败 `httpd.RespondError`
- 绑定优先 `ShouldBindJSON/ShouldBindQuery/ShouldBindURI`，绑定失败返回 `err.Error()`
- WebSocket 统一用 `wsConfig.Handler()`，不在 handler 中定义私有配置

### 错误处理

- 禁止 `fmt.Errorf(err.Error())`，使用 `fmt.Errorf("...: %w", err)` 包装
- `pkgs` 调外部失败时透传原始错误；仅本层自有逻辑失败时加上下文

### 命名与日志

- 缩写全大写：`CPU`、`ID`、`URL`、`HTTP`、`JWT`、`CORS`；`JWTToken` 冗余，改用 `JWT`、`JWTCheck`、`JWTUsername`
- 日志统一 `logman`，禁止 `log.Println`/`fmt.Println`；使用键值对不拼接字符串

### 方法命名规范（强制）

**Handler（`internal/server/`）** — 格式：`{module}{Resource}{Action}`

| 操作 | 命名模式 | 示例 |
|---|---|---|
| 列表 | `{module}{Resource}List` | `dockerContainerList`、`apisixRouteList` |
| 单条 | `{module}{Resource}Inspect` | `dockerImageInspect`、`swarmNodeInspect` |
| 创建/更新/删除 | `{module}{Resource}Create/Update/Delete` | `apisixRouteCreate`、`dockerImageDelete` |
| 操作/动作 | `{module}{Resource}Action` | `dockerContainerAction`、`swarmNodeAction` |
| 状态切换 | `{module}{Resource}StatusPatch` | `apisixRouteStatusPatch` |
| 日志/统计 | `{module}{Resource}Logs/Stats` | `dockerContainerLogs`、`dockerContainerStats` |

**Service / Pkgs（`internal/service/`、`pkgs/`）** — 格式：`{Resource}{Action}`（去掉类名前缀）

| 操作 | 命名模式 | 示例 |
|---|---|---|
| 列表/单条/查询 | `{Resource}List` / `{Resource}Inspect` / `{Resource}` | `RouteList()`、`ImageInspect(id)`、`Stat(ctx)`、`Probe(ctx)` |
| 获取详情 | `{Resource}Inspect` | `NodeInspect(ctx, id)` |
| 创建/更新/删除 | `{Resource}Create/Update/Delete` | `RouteCreate(req)`、`RouteDelete(id)` |
| 状态切换 | `{Resource}StatusPatch` | `RouteStatusPatch(id, status)` |
| 特殊操作 | `{Resource}{Verb}` | `ContainerAction(id, action)`、`WhitelistRevoke(...)` |

- **查询接口不加 `Get` 后缀**：方法名本身已表达"获取"语义（`Stat()`、`Probe()`、`Info()`、`JoinToken()`、`ConfigAll()`），禁止改为 `StatGet()`、`probeGet()` 等形式
- **`Get` 仅用于必要场景**：当方法名去掉 `Get` 后会与已有方法冲突或语义不明时，才可保留 `Get` 后缀
- **模块前缀**：`docker`、`swarm`、`apisix`、`account`、`system`、`filer`、`compose`
- **资源名**：单数形式，不重复模块语义
- **禁止**：`动词+资源` 旧式命名（`CreateRoute`、`ListContainers`、`apisixCreateRoute`）
- **注意**：类名为 `Docker` 时 `ContainerList` 不缩写为 `List`；类名为 `Apisix` 时 `RouteList` 不缩写为 `List`

### 类型定义

- 请求/响应结构体：定义在对应 handler 子包；业务模型：就近定义在对应 `pkgs` 文件
- 避免跨包重复定义语义相同结构体

### 配置结构体与 Provider

- 顶层 `Config`（`config/types.go`），子配置：`Server`、`AgentConfig`、`ApisixConfig`、`DockerConfig`、`MarketplaceConfig`、`MemberConfig`
- `Server` 必须作为 `config.Server` 结构体统一访问，禁止重新展开为 `config.Debug`、`config.ListenAddr` 等包级散变量
- 镜像仓库 `DockerRegistry`（含 `Name`、`URL`、`Username`、`Password`、`Description`）
- 字段使用指针类型 `*` 和 YAML 标签；配置持久化以 YAML 结构为准，禁止用 API 响应的 `json:"-"` 语义保存配置

**配置 Provider 规范（强制）**

- `provider.go` 只定义 `ConfigProvider` 抽象、全局 provider、`Init/SetProvider` 和基于 `CONFIG_PATH` 的适配器分发；禁止在其中解析具体存储细节
- 适配器文件自解析自身配置：`provider_yaml.go` 处理文件路径；`provider_etcd.go` 处理 `etcd://user:pass@host/key?query` URI
- `CONFIG_PATH` 是唯一入口：普通路径/`file://` 使用 YAML；`etcd://` 使用 etcd；禁止新增 `CONFIG_PROVIDER`、`ETCD_ENDPOINTS`、`ETCD_CONFIG_KEY` 等平行入口
- etcd value 存储完整 `config.yml` 同款 YAML 文本，便于 `config.yml` 与 etcd 互迁；禁止改为 JSON，避免敏感字段因 `json:"-"` 丢失
- etcd URI 使用标准形式：`etcd://user:pass@host1:2379,host2:2379/isrvd/config?scheme=http&timeout=5s&fallback=/path/config.yml`
- etcd 认证优先从 URI userinfo 读取；生产场景可用 `ETCD_USERNAME`、`ETCD_PASSWORD` 补充或覆盖；特殊字符必须 URL encode
- `fallback` 只在 etcd key 不存在时触发：读取 YAML 后写入 etcd；etcd 连接失败、权限错误、超时、已有值解析失败均不得 fallback
- etcd watch 只允许做变更检测和重启提示，禁止自动 `Apply` 或静默重建 registry/service
- YAML 明文密码迁移是 `provider_yaml.go` 的历史兼容逻辑，禁止放入 provider 抽象或 etcd provider

---

## 6) 前端编码规范（Vue/Tailwind）

适用目录：`webview/src`

### 6.1 组件装饰器

使用 `vue-facing-decorator`（`@Component`、`@Inject`、`@Prop`、`@Ref`、`@Watch`、`@Provide`），类必须 `toNative()` 包装后导出。`@Inject` 从 `APP_STATE_KEY`/`APP_ACTIONS_KEY` 注入

### 6.2 状态管理

- 全局状态 `store/state.ts` 使用 `reactive` + Provide/Inject
- 键：`APP_STATE_KEY`（`app.state`）、`APP_ACTIONS_KEY`（`app.actions`）
- 权限：`permissionsLoaded`（布尔）、`permissions`（`string[]`，格式为 `"METHOD /api/path"`），通过 `hasPerm(module)` 检查
- 初始化 `initProvider()` 返回 `{ state, actions }`

### 6.3 API 服务层

`service/api.ts` 单例 `class ApiService`，`export default new ApiService()`。请求统一通过 `http`/`httpBlob`（`service/axios.ts`），前者类型安全已解包为 `APIResponse`，后者 Blob 下载专用

### 6.4 类型定义与命名（强制）

`service/types/` 按域拆分（`docker`、`swarm`、`apisix`、`compose`、`cron`、`system`、`account`、`overview`、`filer`），`service/types.ts` 统一 `export *` 导出

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

### 6.5 方法命名规范（强制）

`ApiService` 方法命名：`domainResourceAction` 驼峰格式

| 操作 | 命名模式 | 示例 |
|---|---|---|
| 列表 | `domainResourceList(params?)` | `dockerContainerList()`、`apisixRouteList()`、`cronJobList()` |
| 单条 | `domainResource(id)` | `dockerImage(id)`、`swarmNode(id)` |
| 创建/更新/删除 | `domainResourceCreate/Update/Delete` | `dockerContainerCreate(data)`、`dockerImageDelete(id)`、`cronJobCreate(data)` |
| 操作/动作 | `domainResourceAction(id, action)` | `dockerContainerAction(id, 'start')`、`cronJobRun(id)` |
| 状态切换 | `domainResourceStatus(id, status)` | `apisixRouteStatus(id, 0)`、`cronJobEnable(id, enabled)` |
| 统计/日志 | `domainResourceStats/Logs(id)` | `dockerContainerStats(id)`、`cronJobLogs(id)` |

- **域名前缀**：`docker`、`swarm`、`apisix`、`account`、`system`、`filer`、`compose`、`cron`
- **资源名**：单数形式
- **分组注释**：`// ==================== XXXX 相关 ====================`

### 6.6 卡片与标题栏

- 列表/详情页统一 `.card mb-4`
- 标题栏：使用 `.card-toolbar` 类（定义于 `light_components.css`），容器不写 `flex/justify-between`
- 必须提供桌面 `hidden md:flex` 与移动 `flex md:hidden` 双布局
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

### 6.7 列表双视图与搜索（强制）

- 桌面：`hidden md:block overflow-x-auto` + `<table>`
- 移动：`md:hidden space-y-3 p-4` + 卡片列表（**`p-4` 不得省略**，卡片 `rounded-xl border border-slate-200 bg-white p-4 transition-all hover:shadow-sm`）
- 所有资源列表页必须提供 `searchText` + `filteredXxx` 过滤逻辑，并使用 `webview/src/component/page-search.vue`，禁止手写重复的搜索框 HTML
- 页面级键盘输入直达搜索只能通过 `PageSearch` 的 `type-to-search` 启用；禁止页面直接调用 `bindTypeToSearchFocus`
- 每个列表页只允许一个 `PageSearch` 设置 `type-to-search`（通常为桌面搜索框）；移动端搜索框复用同一个 `search-key`，但不设置 `type-to-search`
- `search-key` 使用稳定唯一值（建议与路由名一致，如 `docker-images`、`apisix-routes`、`system-audit-logs`）
- 过滤逻辑保留在页面内，不下沉到 `PageSearch`；组件只负责输入 UI、`v-model` 和键盘直达绑定

**移动端卡片顶部结构（强制）**：

```html
<div class="flex items-center gap-3 min-w-0 flex-1 mb-3">
  <div class="list-icon bg-xxx flex items-center justify-center flex-shrink-0">
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

### 6.8 移动端卡片属性行对齐（强制）

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
  <code class="text-xs bg-slate-100 px-2 py-0.5 rounded break-all">{{ value }}</code>
</div>
```

### 6.9 表格第一列布局（强制）

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
    <div class="row-icon bg-xxx flex items-center justify-center flex-shrink-0">
      <i class="fas fa-xxx text-white text-sm"></i>
    </div>
    <div class="min-w-0">
      <span class="font-medium text-slate-800 truncate block">{{ 主名称 }}</span>
      <span class="text-xs text-slate-400 truncate block mt-0.5">{{ 副信息 }}</span>
    </div>
  </div>
</td>
```

图标配色（色阶 `400`）：容器 `emerald`/`slate`（按状态）、镜像 `blue`、网络 `purple`、数据卷 `amber`、仓库 `blue-500`、Swarm 服务 `emerald`、节点 `blue`、路由 `indigo`、白名单 `amber`、消费者 `violet`、计划任务 `violet`、用户 `blue-500`。新模块选未用色（`rose`/`cyan`/`lime` 等）

#### 6.9.1 状态文字颜色（强制）

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

#### 6.9.2 枚举 badge 配色

枚举型分类字段（驱动、协议、类型等）使用 badge，颜色跟随模块主色：

| 模块/字段 | badge 配色 |
|---|---|
| 网络驱动（docker network driver） | `bg-purple-50 text-purple-700` |
| 路由上游（apisix upstream） | `bg-indigo-50 text-indigo-700` |
| 其他枚举 | 跟随模块主色，`bg-{color}-50 text-{color}-700` |

#### 6.9.3 权限/角色图标颜色

| 角色/权限 | 图标 | 颜色 |
|---|---|---|
| 创始人/超级管理员 | `fa-crown` | `text-violet-400` |
| 有自定义权限 | `fa-key` | `text-amber-400` |

### 6.10 操作按钮语义色

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

> APISIX 域各资源编辑按钮颜色：路由 `indigo`、消费者 `violet`、SSL `cyan`、上游 `emerald`、插件配置 `rose`；其余模块（docker/swarm/account/filer/compose）统一 `blue`

**移动端操作按钮区（强制）**：

- 容器：`flex flex-wrap gap-1.5 pt-2 border-t border-slate-100`（**`gap-1.5` 不得用 `gap-1`**）

### 6.11 表单与敏感字段

- 表单容器 `max-w-3xl space-y-4`
- label：`block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1`，input 通用 `.input`，help `text-xs text-slate-400 mt-1`
- 密钥/密码：后端敏感字段 `json:"-"`，前端 `type="password" autocomplete="new-password"`，留空保存=不修改，placeholder："留空保持不变"

- **Toggle Switch 规范**（单一功能启用/禁用开关）：使用 `.toggle-row` + `.toggle` + `.toggle-thumb` CSS 类；通过 `:class="{ 'toggle-on': value }"` 控制激活态；Caddy 模块用 `toggle-violet` 修饰符；禁止手写内联 Tailwind toggle 样式：

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

  使用场景：多选列表（插件、权限矩阵）或无描述的简单开关。`{color}` 与页面主色一致，如 `indigo`（APISIX/Docker）、`violet`（Caddy）、`blue`（系统/账户）。

### 6.12 概览 Widget 统计卡片（强制）

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

### 6.13 统一工具与轮询

通用函数复用 `webview/src/helper/utils.ts`；轮询间隔用 `POLL_INTERVAL`，禁止硬编码

### 6.14 import 分组排序

`<script>` 内 import 按以下顺序，组间空一行，组内字母升序：

1. 第三方库  2. `@/store/...`  3. `@/router`  4. `@/service/...`  5. `@/helper/...`  6. `@/component/...`  7. `@/views/...`  8. 其余 `@/`

同模块普通导入在前、`type` 导入在后紧邻。批量整理：`cd webview && python3 sort-imports.py src`（支持 `--dry-run`）

### 6.15 终端能力

系统终端走 `helper/shell.ts`，容器终端走 `helper/container-exec.ts`，禁止页面直接创建 Terminal/WebSocket 实例

### 6.16 暗黑模式样式（强制）

暗黑模式样式统一在 `webview/src/assets/dark.css` 中定义，**禁止在组件中使用 `dark:` 前缀**。

**CSS 选择器转义规则**：

| 原始类名 | CSS 选择器 | 说明 |
|---|---|---|
| `bg-white/80` | `.dark .bg-white\\/80` | 斜杠 `/` 转义为 `\\/` |
| `lg:bg-white` | `.dark .lg\\:bg-white` | 冒号 `:` 转义为 `\\:` |
| `hover:bg-slate-100` | `.dark .hover\\:bg-slate-100:hover` | `hover:` 转义 + `:hover` 伪类 |
| `group-hover:text-primary-600` | `.dark .group:hover .group-hover\\:text-primary-600` | 组悬停需 `.group:hover` 父选择器 |

**已覆盖的颜色类**：

- 背景：`bg-slate-50/100/200/700/800/900`、`bg-white`、半透明 `bg-white/80/95`
- 文字：`text-slate-100/200/300/400/500/600/700/800/900`
- 强调色：`blue`、`primary`、`emerald`、`amber`、`red`、`indigo`、`violet`、`teal`、`cyan`、`rose`、`orange`、`sky`

**新增颜色时必须同步更新**：

1. 在 `dark.css` 对应色系区块添加样式
2. 同时添加 `bg-`、`text-`、`hover:bg-`、`hover:text-` 等变体
3. 检查是否有响应式前缀（`lg:`、`md:`）或组悬停（`group-hover:`）需求

**终端/日志区域**：使用 `bg-slate-900` 保持深色背景，不跟随主题切换

### 6.17 CSS 组件库规范（强制）

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
| `.loading-state` | 加载中 `flex flex-col items-center justify-center py-20` |
| `.empty-state` | 空状态（布局同 loading-state，语义不同） |
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
| `.select-sm` | 小尺寸 select（toolbar 内使用，固定 `h-9`） | `h-9 px-3 bg-white border border-slate-200 rounded-lg text-xs text-slate-700 hover:border-slate-300` |
| `.select-search-header` | 下拉选择器粘性搜索头部 | `px-3 py-2 bg-slate-50 border-b border-slate-100 flex items-center gap-2 sticky top-0 z-10` |
| `.select-list` | 下拉选择器选项列表容器 | `px-2 py-1.5 grid grid-cols-1 gap-0.5 bg-white` |
| `.select-footer` | 下拉选择器底部工具栏 | `px-3 py-2 bg-slate-50 border-t border-slate-100 flex items-center justify-between` |
| `.mobile-search` | 移动端搜索栏容器（桌面端隐藏） | `md:hidden px-4 py-2 border-b border-slate-100` |
| `.check-label` | checkbox/radio label 包裹层 | `flex items-center gap-2 cursor-pointer select-none w-fit` |
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
- **Tab 切换**：使用 `.tab-group` + `.tab-btn` / `.tab-btn-text`，整体高度 36px
- **独立详情页 toolbar**：去掉 `<ContainerNav>` Tab 导航，改用独立 `.card-toolbar`（图标+标题在左，操作控件在右），不添加返回按钮
- **header 区域文字按钮统一**：`app.vue` header 中的文字按钮（如工具箱链接、AI 助手）统一使用 `btn btn-ghost px-4 py-2 text-sm gap-2`，禁止个别按钮手写冗余 inline class

#### 禁止

- 在有对应 CSS 类的情况下使用等价的 inline Tailwind 类
- 在组件中使用 `dark:` 前缀（暗黑模式样式统一在 `dark.css`）
- `.btn-icon-sm` 用于表格行操作（应使用 `.btn-icon-{color}`）

---

## 7) 路由与导航

- `/overview` 概览；`docker/overview`/`swarm/overview` 仅作组件不作独立菜单路由
- 系统模块：`/system/config`、`/system/audit`；用户管理：`/account/members`
- 计划任务：`/cron/jobs`
- 折叠子菜单展开状态跟随当前路由（`@Watch` immediate）
- 侧边栏宽度 `w-16`（折叠）→ `w-64`（展开）
- 移动端：遮罩 + 抽屉式（`-translate-x-full lg:translate-x-0`），`toggleMobileSidebar`/`closeMobileSidebar`/`openMobileSidebar`；窗口 ≥ 1024px 自动关闭

---

## 8) 注册中心与服务初始化

启动顺序：`main → config.Init → registry.Init → server.StartApp`

可用性检查：`IsDockerAvailable`、`IsSwarmAvailable`、`IsApisixAvailable`

已用命名：`registry.DockerService`、`registry.SwarmService`、`registry.ApisixClient`

服务初始化（`server/app.go` `StartApp()`）：
- `overviewSvc`、`configSvc`、`auditSvc`、`accountSvc`、`filerSvc`：直接初始化
- `apisixSvc`：根据可用性检查可选初始化
- `dockerSvc`、`swarmSvc`：根据 Docker 可用性可选初始化
- `composeSvc`：根据 Docker 可用性可选初始化
- `cronSvc`：始终初始化，可选依赖 `registry.DockerService`（用于 DOCKER 类型任务）

---

## 9) 安全基线（必须遵守）

1. 禁止硬编码密钥/密码/令牌
2. 敏感配置仅返回 `xxxSet` 布尔值
3. 文件系统操作防目录遍历；解压防 Zip Slip
4. WebSocket 必须经过认证链路
5. 关键资源（内置角色等）前后端双重校验

---

## 10) 质量门禁（提交前自检）

```bash
go test ./...                                          # 后端编译/测试
cd webview && npm run lint                             # 前端类型检查
cd webview && npm run format                           # import 排序检查
cd webview && python3 script/review-style.py           # 前端样式一致性检查
```

- [ ] 编译通过；[ ] 无新增 lint 警告；[ ] 关键路径手动验证；[ ] 错误处理与日志符合规范；[ ] 未引入明文敏感信息；[ ] 相关文档与 skills 已同步更新

---

## 11) Git 约定

提交格式：`<type>: <subject>`（`feat`/`fix`/`refactor`/`style`/`docs`/`chore`）

分支：`main`（生产）、`dev`（开发）、`feature/<name>`、`fix/<name>`

---

本规范适用于本仓库所有 AI 代理协作与代码改动。如与用户当次明确需求冲突，按"指令优先级"处理并在输出中说明取舍。
