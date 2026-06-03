# AGENTS.md — isrvd Agent 操作指南

> 本文件是 `isrvd` 仓库根规范入口。子目录可有专项 `AGENTS.md`，目标：可执行、可验证、低歧义。

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

- 同领域功能聚合同包（`pkgs/docker/`、`pkgs/swarm/`、`pkgs/caddy/`），类型就近定义，不集中 `types.go`
- 层间通过接口或注入解耦；前端全局状态通过 Pinia `usePortal()` 聚合访问
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
docs/
├── SKILL.md                      ← 索引 + 决策树 + 常见工作流
├── scripts/
│   └── api.sh                    ← 认证持久化 + API 调用封装
└── references/
    ├── docker/{containers,images,networks,volumes,registries}.md
    ├── swarm/{info,services,tasks}.md
    ├── apisix/{routes,upstreams,consumers,ssl}.md
    ├── caddy/{routes,certs,config}.md
    ├── system/{overview,config,account,filer,cron}.md
    ├── overview/
    ├── compose.md
    ├── shell.md
    └── ssh/{hosts,sftp}.md
```

### 需要同步更新的文件

| 代码变更位置 | 需同步更新的文档 |
|---|---|
| `internal/server/ctrl_docker.go` | `docs/references/docker/` 下对应资源文件 |
| `internal/server/ctrl_swarm.go` | `docs/references/swarm/` 下对应资源文件 |
| `internal/server/ctrl_apisix.go` | `docs/references/apisix/` 下对应资源文件 |
| `internal/server/ctrl_caddy.go` | `docs/references/caddy/` 下对应资源文件 |
| `internal/server/ctrl_compose.go` | `docs/references/compose.md` |
| `internal/server/ctrl_cron.go` | `docs/references/system/cron.md` |
| `internal/server/ctrl_system.go` / `ctrl_account.go` | `docs/references/system/` 下对应文件 |
| `pkgs/*/`（数据结构变更） | 对应 docs 文件中的字段表 |
| 新增路由/模块 | `docs/SKILL.md` 索引表 + 决策树 |
| 脚本相关变更 | `docs/scripts/api.sh` |

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
- **模块前缀**：`docker`、`swarm`、`apisix`、`caddy`、`account`、`system`、`filer`、`compose`、`cron`
- **资源名**：单数形式，不重复模块语义
- **禁止**：`动词+资源` 旧式命名（`CreateRoute`、`ListContainers`、`apisixCreateRoute`）
- **注意**：类名为 `Docker` 时 `ContainerList` 不缩写为 `List`；类名为 `Apisix` 时 `RouteList` 不缩写为 `List`

### 类型定义

- 请求/响应结构体：定义在对应 handler 子包；业务模型：就近定义在对应 `pkgs` 文件
- 避免跨包重复定义语义相同结构体

### 配置结构体与 Provider

- 顶层 `Config`（`config/types.go`），子配置：`Server`、`AgentConfig`、`ApisixConfig`、`CaddyConfig`、`DockerConfig`、`MarketplaceConfig`、`MemberConfig`
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

## 6) 前端专项规范（Webview）

前端代码位于 `webview/`，详细 Vue/Tailwind/CSS 组件库、状态管理、import 排序、暗黑模式和样式自检规范见 `webview/AGENTS.md`。

修改 `webview/` 时必须同时遵守本文件的通用规则和 `webview/AGENTS.md` 的前端专项规范。

---

## 7) 路由与导航

- `/overview` 概览；`docker/overview`/`swarm/overview` 仅作组件不作独立菜单路由
- APISIX：`/apisix/routes`、`/apisix/upstreams`、`/apisix/plugin-configs`、`/apisix/ssls`、`/apisix/consumers`、`/apisix/whitelist`
- Caddy：`/caddy/routes`、`/caddy/certs`、`/caddy/global`、`/caddy/raw`
- Docker：`/docker/containers`、`/docker/images`、`/docker/networks`、`/docker/volumes`、`/docker/registries` 及对应详情页
- Swarm：`/swarm/nodes`、`/swarm/services`、`/swarm/tasks` 及对应详情/日志页
- 系统模块：`/system/config`、`/system/audit/logs`；用户管理：`/account/members`、个人设置：`/account/profile`
- 计划任务：`/cron/jobs`；Compose：`/compose/deploy`
- 折叠子菜单展开状态跟随当前路由（`@Watch` immediate）
- 侧边栏宽度 `w-16`（折叠）→ `w-64`（展开）
- 移动端：遮罩 + 抽屉式（`-translate-x-full lg:translate-x-0`），`toggleMobileSidebar`/`closeMobileSidebar`/`openMobileSidebar`；窗口 ≥ 1024px 自动关闭

---

## 8) 注册中心与服务初始化

启动顺序：`main → config.Init → registry.Init → server.StartApp`

可用性检查：由各 `service` 层的 `CheckAvailability(ctx)` 方法负责（`service/docker`、`service/swarm`、`service/apisix`、`service/caddy`、`service/compose`），不再通过 `registry` 层的独立函数检查。

已用命名：`registry.DockerService`、`registry.SwarmService`、`registry.ApisixClient`、`registry.CaddyService`

服务初始化（`server/app.go` `StartApp()`）：
- `overviewSvc`、`configSvc`、`auditSvc`、`accountSvc`、`filerSvc`、`shellSvc`、`agentSvc`：直接初始化
- `apisixSvc`、`caddySvc`：根据可用性检查可选初始化
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
cd webview && npm run lint                             # 前端类型检查 + ESLint
cd webview && npm run format:check                     # import 排序/格式检查（dry-run，需关注输出）
cd webview && python3 scripts/review-style.py          # 前端样式一致性辅助检查
```

`npm run format` 会直接修复 import/ESLint；仅在需要改写文件时使用。`review-style.py` 只有 ERROR 阻断，WARN 需人工确认。

- [ ] 编译通过；[ ] 无新增 lint 警告；[ ] 关键路径手动验证；[ ] 错误处理与日志符合规范；[ ] 未引入明文敏感信息；[ ] 相关文档与 skills 已同步更新

---

## 11) Git 约定

提交格式：`<type>: <subject>`（`feat`/`fix`/`refactor`/`style`/`docs`/`chore`）

分支：`main`（生产）、`dev`（开发）、`feature/<name>`、`fix/<name>`

---

本规范适用于本仓库所有 AI 代理协作与代码改动。如与用户当次明确需求冲突，按"指令优先级"处理并在输出中说明取舍。
