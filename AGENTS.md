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
- **同步更新文档**：外部可观察的 API、配置、数据结构、工作流或脚本用法发生变化时同步文档；纯重构、格式整理和内部并发封装无需制造无意义文档差异

---

## 3) 项目架构

### 实际依赖方向

仓库不是单一线性依赖链，当前 import 边界为：

```text
server/cmd/server ────────────→ server/config + server/app + server/gateway + server/service/{account,node}
server/gateway ───────────────→ server/service/node
server/config ────────────────→ pkgs/cstore
server/service/node ──────────→ server/config + pkgs/cstore + libgo/wstunnel
server/service/{account,apisix,...} → server/config / pkgs/*
server/service/{docker,webssh} → server/service/shell（终端桥接复用）
server/service/{cron,monitor} → server/service/notify（任务失败与资源告警）
server/service/notify ────────→ pkgs/apisix + libgo/certify（证书到期检测，客户端由 app 注入；Caddy 证书经 `notify.CaddyCert` 由 app 适配，notify 不依赖 service/caddy）
server/app ────────────────→ server/config + server/service/{account,apisix,...} + pkgs/* + public
```

- `pkgs/`：底层客户端、存储适配和 SDK 类型转换；不依赖 `server/service/`、`server/app/`
- `server/service/{account,apisix,...}`：业务组合、参数校验、稳定 API 类型转换；各服务在 `NewService()` 中直接构造底层客户端；不得依赖 `server/app`，**不得依赖 gin**（需要请求信息时接收 `*http.Request`、`context.Context` 或普通结构体）
- `server/app/`：Gin HTTP/WebSocket 入口、路由索引、中间件、服务生命周期与响应封装
- `server/gateway/`：中控模式的网关（节点管理 API、`/n/<节点ID>/` 转发、受管机隧道入口），以 `http.Handler` 形式由 `cmd/server` 装配；不依赖 `server/app`，页面一律由 webview 提供；`?token=` 的适用范围与 isrvd 保持一致，由 `cmd/server` 通过 `Options.QueryTokenRoutes`（来自 `app.QueryTokenRoutes()`）注入，不得在网关里另写一份路由清单；上游不可达返回 502，不得当作未登录（401）；操作经 `Options.Audit`（`app.Audit()`，写入 app 当前的审计服务，重载后自动指向新实例）按 isrvd 审计基线记录，不得另建 `AuditService` 写同一目录；会随配置重载变化的值（如 `allowedOrigins`）以函数传入、每次读取，不得在启动时固化
- `server/cmd/server/`：`main.go` 先 `parse` 校验参数（参数错误或 `--help` 不得读取或回写配置），再 `config.Init → run`；`launch.go` 按 `--mode`（`server`/`center`/`agent`）装配并调用 `app.StartApp`，是唯一的组装点，业务逻辑不得写在这里。构建与运行一律用包路径 `./server/cmd/server`（含多个文件，不能再用单文件路径）

### 禁止

- `pkgs/` 依赖 `server/service/` 或 `server/app/`
- `handler` 中堆叠业务逻辑
- `service/handler` 直接从配置创建外部客户端；外部客户端统一由 service 层在 `NewService()` 时自行初始化

### 内聚与耦合

- 同领域功能聚合同包（`pkgs/docker/`、`pkgs/swarm/`、`pkgs/caddy/`），类型就近定义，不集中 `types.go`
- 层间通过接口或注入解耦；前端全局状态通过 Pinia `usePortal()` 聚合访问
- 判定：改一个功能只需改一处；若需同时改多个包同名函数，说明内聚不足

---

## 4) 代码变更同步文档规范

外部行为变化时必须同步相关说明文档，确保文档与代码一致。仅移动代码、抽取辅助函数、修复缩进、返回对象快照化等不改变外部契约的修改，不要求更新 API 文档。

### 适用范围

所有涉及以下外部可观察变更的场景：

- 新增/修改/删除 API 路由或参数
- 新增/修改/删除 数据结构（Request/Response 字段）
- 新增/修改/删除 业务逻辑或工作流
- 新增/修改/删除 配置项或环境变量
- 新增/修改/删除 Shell 脚本中的命令或参数

### Docs 文件结构

```
docs/
├── SKILL.md                      ← 索引 + 决策树 + 常见工作流
├── multi-node.md                 ← 多服务器管理（部署、参数、行为与边界）
├── scripts/
│   ├── api.sh                    ← Bash API 调用封装
│   ├── api.js                    ← JavaScript API 调用封装
│   └── api.py                    ← Python API 调用封装
└── references/
    ├── docker/{containers,images,networks,volumes,registries}.md
    ├── swarm/{info,services,tasks}.md
    ├── apisix/{routes,upstreams,consumers,ssl}.md
    ├── caddy/{routes,servers,certs,config,basic-auth}.md
    ├── system/{config,account,filer,cron}.md
    ├── ssh/{hosts,sftp}.md
    ├── node/{nodes,codes}.md
    ├── copilot.md
    ├── overview.md
    ├── compose.md
    ├── local.md
    └── shell.md
```

### 需要同步更新的文件

| 代码变更位置 | 需同步更新的文档 |
| --- | --- |
| `server/app/ctrl_docker*.go` | `docs/references/docker/` 下对应资源文件 |
| `server/app/ctrl_swarm.go` | `docs/references/swarm/` 下对应资源文件 |
| `server/app/ctrl_apisix.go` | `docs/references/apisix/` 下对应资源文件 |
| `server/app/ctrl_caddy.go` | `docs/references/caddy/` 下对应资源文件 |
| `server/app/ctrl_compose.go` | `docs/references/compose.md` |
| `server/app/ctrl_cron.go` | `docs/references/system/cron.md` |
| `server/app/ctrl_system.go` / `ctrl_account.go` | `docs/references/system/` 下对应文件 |
| `server/app/ctrl_filer.go` | `docs/references/system/filer.md` |
| `server/app/ctrl_webssh.go` | `docs/references/ssh/` 下对应文件 |
| `server/app/ctrl_copilot.go` | `docs/references/copilot.md` |
| `server/app/ctrl_overview.go` | `docs/references/overview.md` |
| `server/app/ctrl_local.go` | `docs/references/local.md` |
| `server/app/ctrl_shell.go` | `docs/references/shell.md` |
| `server/gateway/`、`server/service/node/` | `docs/references/node/` 下对应文件，部署与行为变化同步 `docs/multi-node.md` |
| `pkgs/*/`（数据结构变更） | 对应 docs 文件中的字段表 |
| 新增路由/模块 | `docs/SKILL.md` 索引表 + 决策树 |
| API 调用脚本变更 | `docs/scripts/api.sh`、`api.js`、`api.py` 中受影响的实现 |
| 构建/开发脚本用法变更 | `README.md` 中对应的构建或开发说明 |

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

- 状态码用 `net/http` 常量；HTTP JSON 响应统一走 `server/app/response.go` 的 `respondSuccess`、`respondError`、`respondResult`
- `respondResult` 仅用于“成功 200、service 错误 500”的标准查询；需要自定义成功文案时用 `respondResultMsg(c, "中文提示", data, err)`（无数据负载传 `nil`）；需要 400/404/503 等其他状态码时显式调用对应响应函数。`server/cmd/openapi-gen` 识别 `respondSuccess`、`respondResult`、`respondResultMsg` 三者推断响应类型，新增响应封装函数时须同步更新生成器
- 绑定优先 `ShouldBindJSON/ShouldBindQuery/ShouldBindURI`，绑定失败返回 `err.Error()`
- WebSocket 统一用 `wsConfig.Handler()`，不在 handler 中定义私有配置

### 错误处理

- 禁止 `fmt.Errorf(err.Error())`，使用 `fmt.Errorf("...: %w", err)` 包装
- `pkgs` 调外部失败时透传原始错误；仅本层自有逻辑失败时加上下文

### 命名与日志

- 缩写全大写：`CPU`、`ID`、`URL`、`HTTP`、`JWT`、`CORS`；`JWTToken` 冗余，改用 `JWT`、`JWTCheck`、`JWTUsername`
- 日志统一 `logman`，禁止 `log.Println`/`fmt.Println`；使用键值对不拼接字符串

### 方法命名规范（强制）

**Handler（`server/app/`）** — 格式：`{module}{Resource}{Action}`

| 操作 | 命名模式 | 示例 |
| --- | --- | --- |
| 列表 | `{module}{Resource}List` | `dockerContainerList`、`apisixRouteList` |
| 单条 | `{module}{Resource}Inspect` | `dockerImageInspect`、`swarmNodeInspect` |
| 创建/更新/删除 | `{module}{Resource}Create/Update/Delete` | `apisixRouteCreate`、`dockerImageDelete` |
| 操作/动作 | `{module}{Resource}Action` | `dockerContainerAction`、`swarmNodeAction` |
| 状态切换 | `{module}{Resource}StatusPatch` | `apisixRouteStatusPatch` |
| 日志/统计 | `{module}{Resource}Logs/Stats` | `dockerContainerLogs`、`dockerContainerStats` |

**Service / Pkgs（`server/service/{account,apisix,...}`、`pkgs/`）** — 格式：`{Resource}{Action}`（去掉类名前缀）

| 操作 | 命名模式 | 示例 |
| --- | --- | --- |
| 列表/单条/查询 | `{Resource}List` / `{Resource}Inspect` / `{Resource}` | `RouteList()`、`ImageInspect(id)`、`Stat(ctx)`、`Probe(ctx)` |
| 获取详情 | `{Resource}Inspect` | `NodeInspect(ctx, id)` |
| 创建/更新/删除 | `{Resource}Create/Update/Delete` | `RouteCreate(req)`、`RouteDelete(id)` |
| 状态切换 | `{Resource}StatusPatch` | `RouteStatusPatch(id, status)` |
| 特殊操作 | `{Resource}{Verb}` | `ContainerAction(id, action)`、`WhitelistUserCreate(...)` |

- **查询接口不加 `Get` 后缀**：方法名本身已表达"获取"语义（`Stat()`、`Probe()`、`Info()`、`JoinToken()`、`ConfigAll()`），禁止改为 `StatGet()`、`probeGet()` 等形式
- **`Get` 仅用于必要场景**：当方法名去掉 `Get` 后会与已有方法冲突或语义不明时，才可保留 `Get` 后缀
- **模块前缀**：`docker`、`swarm`、`apisix`、`caddy`、`account`、`system`、`filer`、`compose`、`cron`、`node`（`server/gateway` 的处理函数同样遵循 `{module}{Resource}{Action}`，如 `nodeCodeCreate`）
- **资源名**：单数形式，不重复模块语义
- **禁止**：`动词+资源` 旧式命名（`CreateRoute`、`ListContainers`、`apisixCreateRoute`）
- **注意**：类名为 `Docker` 时 `ContainerList` 不缩写为 `List`；类名为 `Apisix` 时 `RouteList` 不缩写为 `List`

### 文件内组织

文件内顺序（自上而下）：

1. **共享的 const/var 定义** — 被多个函数或多个文件使用、无单一归属的包级常量/变量，放在 import 之后、类定义之前
2. **类定义、初始化函数** — struct 定义及其 `New*` 构造函数紧随其后
3. **类共有函数、私有函数** — 业务方法按逻辑分组排列
4. **独立的辅助函数** — 非业务入口的内部工具函数统一放在文件末尾，用 `// ─── 辅助函数 ───` 注释标记

**就近定义优先于上面的固定顺序**，判定标准是「该声明是否被共享」，而不是「它出现在文件第几个」：

- 只服务单个函数的 const/var/类型**紧贴该函数**，不因「包级声明应在顶部」而上提（如 `server/service/filer/service.go` 的 `previewContentTypes` 紧贴 `PreviewContentType`，`server/service/overview/version.go` 的缓存变量紧贴 `CheckVersion`）
- 类型专属枚举紧贴其类型定义，不移到文件顶部（如 `pkgs/cstore/store.go`、`server/service/copilot/agui/event.go` 的 `EventType` 枚举）
- 禁止把函数专属的类型集中堆到文件顶部
- 某声明已紧贴它所服务的函数时，即使文件中更靠前的函数也引用了它，仍保持就近；**不要**为消除「先引用后定义」而把它上提到顶部（Go 允许同包内声明顺序任意，不影响编译）。例如 `server/service/notify/fault.go` 的 `containerFaultState` 紧贴 `checkContainer`，但更靠前的 `NewFaultWatcher` 同样引用它
- 不得以「声明出现在首个 type/func 之后」作为唯一判据——只有共享声明才适用第 1 条，函数专属声明适用就近原则

其他规则：

- handler 私有的 URI/query/body 小结构体放在 `server/app` 对应控制器；跨 handler 复用或参与业务校验的 Request/Response 放在对应 service 包（`server/service/{account,apisix,...}`）；SDK 转换模型放在 `pkgs`
- 避免跨包重复定义语义相同结构体

### 请求处理、权限与审计基线

**错误处理与响应**：

- 状态码一律使用 `net/http` 常量；成功用 `respondSuccess(c, "中文提示", data)`，service 调用结果统一用 `respondResult(c, data, err)`
- **绑定错误分流**：handler 若设置了 `http.MaxBytesReader`（如在线编辑的 `maxEditableJSONBytes`），其 `ShouldBindJSON` 必须用 `respondBindError`（把 `MaxBytesError` 转成 413 并给出友好提示）；未设大小限制的普通绑定用 `respondError(c, http.StatusBadRequest, err.Error())`
- 文件上传类接口走 `MaxUploadSize`：先用 `ContentLength` 预检，不走 `respondBindError`

**context 来源**：

- 请求处理路径一律使用 `c.Request.Context()`
- `context.Background()` 仅允许用于生命周期根、后台 goroutine 与优雅退出（`server/app/app.go` 生命周期、`lifecycle.go` 信号监听与优雅退出、`config/provider.go` 的 watch、cron 父 context、`server/cmd/server/launch.go` 的 agent 生命周期与网关优雅退出），禁止出现在请求处理路径

**服务 nil 契约**（新增服务时最易遗漏）：

- 外部依赖不可用时 `app.xxxSvc` 为 `nil`，而不是返回带错误的空对象
- 会被 nil 接收者调用的访问器（如 `Raw()`）必须做**接收者** nil 安全判断（`if s == nil`），典型调用方 `app.swarmSvc.Raw()` 在服务不可用时会传入 nil
- `CheckAvailability` 只校验内部字段，由调用点保证接收者非 nil（`initServices` 仅在服务构造成功后才把 `xxxSvc.CheckAvailability` 注册进 `app.probes` 模块表）；新增调用点必须先判空
- `app.probes` 同时承担可用性判断与探活：key 存在即模块已就绪（`isServiceAvailable`），value 为探活函数（无需探活的本地模块如 `ssh` 注册为 `nil`）；新增依赖外部服务的模块需先在 `optionalModules` 登记，再通过 `app.markReady(module, probe)` 标记就绪（未登记会告警）

**服务构造函数签名按能力分级**：

| 能力 | 签名 | 示例 |
| --- | --- | --- |
| 不会失败 | `NewService() *Service` | `account`、`filer`、`shell`、`overview`、`copilot` |
| 可能失败 | `NewService(...) (*Service, error)` | `docker`、`webssh`、`apisix`、`caddy`、`compose`、`swarm` |
| 需要探活 | 额外接收 `ctx context.Context` 首参 | `apisix`、`caddy`、`swarm` |

**路由的 Module 与 Label**：

- `Module` 驱动服务可用性判断（`isServiceAvailable`），`Label` 用于权限校验失败时的提示文案
- 权限匹配依据是 `METHOD /api/path`（`PermCheck` 内 `routeKey := method + " " + path`），不是 `Label`；创始人（`Founder`）跳过全部权限校验
- 路由条目可能跨行（Handler 为内联闭包时字段在后续行），核查完整性时不能只按单行匹配

**访问级别与审计缺省值**：

- `Access` 缺省为 `AccessPerm`（需具体权限）；匿名必须显式声明 `AccessAnon`（目前仅登录相关接口与 `/overview/bootstrap`）；`AccessAuth` 表示登录即可访问
- `Audit` 缺省为 `AuditByMethod`：审计所有非 GET 请求**以及所有 WebSocket**；高频 GET 轮询天然豁免，需要豁免的写操作必须显式 `AuditIgnore`
- 给 POST 路由设置 `AuditAlways` 是冗余的（POST 在 `AuditByMethod` 下本就会被审计），仅当该接口改为 GET 后仍需审计时才有意义

### 配置结构体与 Provider

- 顶层 `Config`（`server/config/types.go`）当前包含 `Schema`、`Server`、`Password`、`Passkey`、`OIDC`、`THA`、`Copilot`、`Notify`、`Apisix`、`Caddy`、`Docker`、`Monitor`、`Marketplace`、`Links`、`Members`
- `PUT /api/system/config` 支持按分区提交：`AllConfig` 中为 nil 的分区跳过更新，密钥类字段为空表示保留原值
- `Server` 必须作为 `config.Server` 结构体统一访问，禁止重新展开为 `config.Debug`、`config.ListenAddr` 等包级散变量
- 镜像仓库 `DockerRegistry`（含 `Name`、`URL`、`Username`、`Password`、`Description`）
- 远程 Docker：`DockerConfig.TLS`（`DockerTLSConfig`，证书与私钥为 PEM 文本，不使用文件路径，以便随 etcd 配置同步）；`pkgs/docker.TLSConfig.Config(host)` 是唯一的校验与构造入口（`NewDockerService` 与 `system.dockerTLSMerge` 共用）；`DockerService.Remote()` 由 `Host` 地址决定（`tcp://` 且非回环，与 TLS 无关），为真时自身容器识别只用 overlay 路径、不按 IP/主机名匹配，并跳过 bind 源的本机检查
- 顶层配置分区使用指针并带 YAML 标签；配置持久化以 YAML 结构为准，API 脱敏由 `service/system.ConfigAll` 的深拷贝负责

**配置 Provider / cstore 规范（强制）**

- `server/config/provider.go` 负责基于 `CONFIG_PATH` 初始化全局 `cstore.TypedStore[*Config]`、加载/保存配置、监听变更并发送 `ReloadCh`；禁止在业务层绕过 `config.Load/Save` 直接读写配置存储；业务数据（计划任务、SSH 主机/凭据）统一通过 `config.OpenData` 打开，随配置后端切换本地文件或 etcd，日志类数据仍写本地
- 存储适配由 `pkgs/cstore/` 负责：`store.go` 定义统一 `Store` 抽象和 URI 分发，`file.go` 处理本地 YAML 文件，`etcd.go` 处理 etcd URI，`typed.go` 负责 YAML 序列化/反序列化
- `CONFIG_PATH` 是唯一入口：普通路径/`file://` 使用本地 YAML；`etcd://` 使用 etcd；禁止新增 `CONFIG_PROVIDER`、`ETCD_ENDPOINTS`、`ETCD_CONFIG_KEY` 等平行入口
- etcd value 存储完整 `config.yml` 同款 YAML 文本，便于本地 YAML 与 etcd 互迁；禁止改为 JSON，避免敏感字段因 `json:"-"` 丢失
- etcd URI 中的 path 表示完整配置 key，且必须显式提供；系统配置推荐 key 为 `/isrvd/config`，标准形式：`etcd://user:pass@host1:2379,host2:2379/isrvd/config?scheme=http&timeout=5s&fallback=/path/config.yml`
- `fallback` 是本地 YAML 文件路径，且只在 etcd key 不存在时触发：读取该 YAML 后写入 etcd；etcd 连接失败、权限错误、超时、已有值解析失败均不得 fallback
- etcd 认证优先从 URI userinfo 读取；生产场景可用 `ETCD_USERNAME`、`ETCD_PASSWORD` 补充或覆盖；特殊字符必须 URL encode。环境变量由 `server/config/provider.go` 读取并经 `cstore.WithEtcdCredentials` 注入，`pkgs/cstore` 不直接读取环境变量
- etcd watch 只允许做变更检测并发送重载信号；服务重建必须走 `server/app/app.go` 的 reload 流程，禁止在 cstore 层自动 `Apply` 或静默重建 service
- YAML 明文密码迁移属于 `server/config/migrate.go` 的兼容逻辑，禁止放入 `pkgs/cstore` 抽象或 etcd 存储适配

---

## 6) 前端专项规范（Webview）

前端代码位于 `webview/`，详细 Vue/Tailwind/CSS 组件库、状态管理、import 排序、暗黑模式和样式自检规范见 `webview/AGENTS.md`。

修改 `webview/` 时必须同时遵守本文件的通用规则和 `webview/AGENTS.md` 的前端专项规范。

---

## 7) 路由与导航

- `/overview` 概览；本机能力为 `/local/monitor`、`/local/explorer`、`/local/process`、`/local/shell`
- SSH：`/ssh/hosts`、`/ssh/credentials`、`/ssh/host/:id`
- APISIX：`/apisix/routes`、`/apisix/upstreams`、`/apisix/plugin-configs`、`/apisix/ssls`、`/apisix/consumers`、`/apisix/whitelist`
- Caddy：`/caddy/servers`、`/caddy/routes`、`/caddy/certs`、`/caddy/global`、`/caddy/basic-auth`、`/caddy/raw`
- Docker：`/docker/containers`、`/docker/images`、`/docker/networks`、`/docker/volumes`、`/docker/registries` 及对应详情页
- Swarm：`/swarm/nodes`、`/swarm/services`、`/swarm/tasks` 及对应详情/日志页
- 节点管理（仅 `center` 模式的创始人）：`/node`，单个列表页同时展示受管节点与待接入的注册码；节点视角为页面路径 `/n/<节点ID>/`
- 系统模块：`/system/config`（父布局 + 5 个分组子路由 `/system/config/{service,auth,gateway,alert,integrations}`，默认跳转 `service`；侧边栏「系统配置」为折叠子菜单，项名与顺序来自 `webview/src/stores/config.ts` 的 `configGroups`，该常量同时定义分组与后端配置分区的映射）、`/system/audit/logs`；用户管理：`/account/members`；账户设置：`/account/password`、`/account/passkeys`、`/account/apikey`
- 计划任务：`/cron/jobs`；Compose：`/compose/marketplace`、`/compose/deploy`
- 折叠子菜单展开状态跟随当前路由（`@Watch` immediate）
- 侧边栏宽度 `w-16`（折叠）→ `w-64`（展开）
- 桌面端导航与内容分隔线由 `main` 的 `lg:border-l` 提供，侧边栏自身不设置右边框；登录后的主布局容器使用 `pt-16 min-h-screen` 避让全局 header，`main` 使用 `min-h-[calc(100vh-4rem)]` 覆盖剩余视口，避免外边距折叠导致高度偏差
- 移动端：遮罩 + 抽屉式（`-translate-x-full lg:translate-x-0`），`toggleMobileSidebar`/`closeMobileSidebar`/`openMobileSidebar`；窗口 ≥ 1024px 自动关闭

---

## 8) 服务初始化

启动顺序：`main → parse → config.Init → run → app.StartApp`；`run` 按 `--mode` 选择 `server`（直接 `StartApp`）、`center`（先启动网关再 `StartApp`）或 `agent`（先启动隧道客户端再 `StartApp`），详见 `docs/multi-node.md`

可用性检查：由各 `service` 层的 `CheckAvailability(ctx)` 方法负责（`server/service/docker`、`server/service/swarm`、`server/service/apisix`、`server/service/caddy`、`server/service/compose`）。

服务初始化（`server/app/services.go` 的 `initServices()`）：

- `overviewSvc`、`configSvc`、`auditSvc`、`accountSvc`、`filerSvc`、`shellSvc`、`copilotSvc`：直接初始化
- `websshSvc`：初始化自己的主机/凭据存储和 SFTP 客户端，失败时标记不可用
- `apisixSvc`、`caddySvc`：根据可用性检查可选初始化
- `dockerSvc`、`swarmSvc`：根据 Docker 可用性可选初始化
- `composeSvc`：根据 Docker 可用性可选初始化
- `cronSvc`：始终初始化，可选依赖 Docker 原生客户端（用于 DOCKER 类型任务）
- `monitorCollector`：始终创建并启动；reload/退出时与 `websshSvc` 一起释放资源

---

## 9) 安全基线（必须遵守）

1. 禁止硬编码密钥/密码/令牌
2. 配置查询必须深拷贝后清空 JWT、OIDC、Copilot、APISIX、Registry 和 Docker TLS 私钥（`docker.tls.key`）；账户密码、TOTP secret、SSH 密码/私钥继续使用 `json:"-"`
3. 文件系统操作防目录遍历；解压防 Zip Slip
4. WebSocket 必须经过认证链路
5. 关键资源（内置角色等）前后端双重校验
6. SSH 密码/私钥、节点注册码与受管机节点令牌加密落盘（`server/service/webssh/secret.go`、`server/service/node/secret.go`，密钥由 JWT 密钥派生）；节点令牌与领取密钥在中控侧只存哈希

---

## 10) 质量门禁（提交前自检）

```bash
go test ./...                                          # 后端全包编译（仓库当前不提交测试文件）
go vet ./...                                           # Go 静态检查
cd webview && npm run lint                             # 前端类型检查 + ESLint
cd webview && npm run format:check                     # import 排序/格式检查（dry-run，需关注输出）
cd webview && python3 scripts/review-style.py          # 前端样式一致性辅助检查
cd webview && npm run build                            # 前端生产构建
sh -n build.sh                                         # 构建脚本语法检查（修改 build.sh 时）
git diff --check                                       # 空白与冲突标记检查
```

`npm run format` 会直接修复 import/ESLint；仅在需要改写文件时使用。`review-style.py` 只有 ERROR 阻断，WARN 需人工确认。

- [ ] 编译通过；[ ] 无新增 lint 警告；[ ] 关键路径手动验证；[ ] 错误处理与日志符合规范；[ ] 未引入明文敏感信息；[ ] 相关文档已同步更新

仓库策略是不提交 `*_test.*`、`*.test.*`、`*.spec.*` 测试文件；如需临时验证，应放在仓库外或在交付前清理。不能把“无测试文件”表述为行为测试已覆盖。

`build.sh` 会根据最近两个版本标签生成 `RELEASE.md`；生成时过滤 `release...` 以及 GitHub 默认的 `Merge pull request #<n> from ...` 标题，再执行前端构建、OpenAPI 生成、跨平台 Go 编译和分发打包。

---

## 11) Git 约定

提交格式：`<type>(<module>): <subject>`（`feat`/`fix`/`refactor`/`style`/`docs`/`chore`），标题保持简短，详细说明放提交正文

分支：`master`（当前默认/生产）、`dev`（开发）、`feature/<name>`、`fix/<name>`

---

本规范适用于本仓库所有 AI 代理协作与代码改动。如与用户当次明确需求冲突，按"指令优先级"处理并在输出中说明取舍。
