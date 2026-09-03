# CopilotKit 切换可行性评估

> 评估日期：2026-09-02　分支：`feature/copilotkit`
> 评估对象：将内置 AI 助手从 `page-agent` 切换为 [CopilotKit](https://github.com/copilotkit/copilotkit)

## 结论

**可行，已实现。** 采用「CopilotKit 负责编排与会话 UI，page-controller 提供 DOM 操作能力」的混合方案。

关键理由：两者的能力模型不同 —— `page-agent` 让 LLM **自主浏览并操作当前页面 DOM**，而 CopilotKit 要求开发者**预先声明工具（`useFrontendTool`）**，LLM 只能调用已声明的工具，其本身**不提供**页面 DOM 自动操作能力。因此把 `@page-agent/page-controller` 包装成一个 `page_action` 前端工具，让 Copilot 既能调用 REST API，也能直接操作页面。

---

## 1. 现状：page-agent 方案

| 项 | 值 |
| --- | --- |
| 依赖 | `page-agent: ^1.8.2`（`webview/package.json:44`） |
| lockfile 解析版本 | `1.10.0`（**与 `node_modules` 实际安装的 `1.8.2` 不一致**，`npm ci` 会跳版本） |
| 周下载量 | ~20.8k |
| 运行时形态 | 纯前端 DOM agent，无需后端浏览器 |
| 操作方式 | 抽取可交互元素树 → 编号高亮 → LLM 选序号执行（`clickElement` / `inputText` / `selectOption` / `scroll` / `executeJavascript`） |
| 心智模型 | reflection-before-action |

**优势**：零后端改造，开箱即能操作系统里任意 UI（55 个路由全部自动可用）。
**问题**：行为不可枚举、不可审计；LLM 直接点击 DOM，破坏性操作难以拦截；提示词里写「执行前先确认」只能靠模型自律。

## 2. CopilotKit 方案

### 2.1 Vue 支持：官方且可用（此前预估的最大风险已排除）

`@copilotkit/vue` 是 **CopilotKit 官方仓库 `packages/vue`**，非社区移植：

| 项 | 值 |
| --- | --- |
| 最新版 | `1.70.0`（2026-08-31 发布，与 `@copilotkit/runtime` 同版本同步） |
| 首版发布 | 2026-05-13 |
| 周下载量 | ~10.7k（React 版 `@copilotkit/react-core` 为 ~452k） |
| peerDependencies | **仅 `vue >=3.3.0`，不含 React** |
| 依赖树 | `@copilotkit/core` + `@ag-ui/client` + `lucide-vue-next` + `streamdown-vue` 等，**无 React / ReactDOM** |

与同类方案对比：

| 方案 | 周下载 | 官方维护 | 说明 |
| --- | --- | --- | --- |
| `@copilotkit/vue` | ~10.7k | ✅ CopilotKit 官方 | 与 React 版同版本号、同 CHANGELOG 发布 |
| `vue-copilotkit` | 未见数据 | ❌ 第三方 | 基于 React UI 库的 Vue 移植，不推荐 |

**此前「必须引入 React 桥接」的判断不成立** —— `@copilotkit/vue` 依赖树中 React 出现 0 次，无需 `veaury` 之类的桥接层。

风险点：Vue 版相对 React 版用户量小（约 1/42），官方 `PARITY.md` 自述处于持续对齐阶段，文档「feature complete 但 docs 仍在补齐」。属于官方维护但生态较小的取舍。

### 2.2 后端：Go 直连 AG-UI，无需 Node 运行时

官方 quickstart 用 Node 的 `@copilotkit/runtime` 托管 `/api/copilotkit`，但**并非强制**。`@copilotkit/core` 暴露了两个可绕过它的入口：

```ts
agents__unsafe_dev_only?: Record<string, AbstractAgent>  // 注释：仅开发用
selfManagedAgents?: Record<string, AbstractAgent>        // 生产可用
```

两者都接受 `AbstractAgent`，而 `@ag-ui/client` 导出的 `HttpAgent` 正是其实现。前端可直接指向任意 AG-UI 端点：

```ts
new HttpAgent({ url: "/api/agui", headers: { Authorization: `Bearer ${jwt}` } })
```

`HttpAgent` 的通信契约（读 `@ag-ui/client@0.0.59` 源码确认）：**单个 URL，`POST`，`Accept: text/event-stream`**，body 为 `RunAgentInput`（`threadId`/`runId`/`messages`/`tools`/`context`/`state`/`forwardedProps`），响应为 AG-UI SSE 流。这正好是一个 Go handler 能提供的形态。

**已端到端验证**：官方 Go SDK `github.com/ag-ui-protocol/ag-ui/sdks/community/go` 可直接产出标准 AG-UI SSE 帧：

```
id: RUN_STARTED_1788352937073
data: {"type":"RUN_STARTED","timestamp":1788352937073,"threadId":"t1","runId":"r1"}

id: TEXT_MESSAGE_CONTENT_1788352937073
data: {"type":"TEXT_MESSAGE_CONTENT","timestamp":1788352937073,"messageId":"m1","delta":"你好"}
```

即 **Go 侧无需 Node，也无需自己拼 SSE 帧格式**。

⚠️ 但该 SDK 目前只有伪版本 `v0.0.0-20260902075100-e929f557f59b`，**无正式 release**。事件结构本身很小（约 17 种事件、JSON 编码），自写编码器约 150 行即可，可规避 0.0.0 依赖。两种选法都成立，见「建议」。

### 2.3 现有 Go 侧改动面很小

| 项 | 现状 | 迁移成本 |
| --- | --- | --- |
| Agent 代码 | `ctrl_agent.go` + `service/agent/service.go`，约 110 行 | 低 |
| 性质 | **透明反向代理**，非 agent runtime；已具备鉴权注入、model 重写、流式拷贝（10 分钟超时） | 低 |
| 路由注册 | `Route{Method,Path,Handler,Module,Label,Access,Audit,QueryToken}`，中间件自动带权限与审计 | 低 |
| SSE | `httpd.NewEventWriter(c.Writer)` 已在 `ctrl_docker.go:164`、`ctrl_swarm.go:144` 使用 | 低（可直接复用） |
| JWT 逃生通道 | `Route.QueryToken` 专为「SSE 等无法携带 Header 的场景」设计（`app.go:82`） | 低（AG-UI 流可直接用） |
| 配置 | `config.Agent{Model,BaseURL,APIKey}` + 配置页 + 脱敏，均已完备 | 无 |

新增一个 `POST /api/agui` 路由即可，权限、审计、鉴权全部由现有中间件免费提供。

---

## 3. 核心差异与风险

### 3.1 能力模型差异（最需关注）

| | page-agent | CopilotKit |
| --- | --- | --- |
| 能力来源 | LLM 自主扫描 DOM 树并操作 | 开发者预先用 `useFrontendTool` 声明 |
| 覆盖范围 | 全部 55 个路由自动可用 | 只覆盖已声明工具的功能 |
| 页面上下文 | 自动（`getBrowserState`） | 需 `useCopilotReadable` 显式喂给模型 |
| 破坏性操作 | 提示词约束，靠模型自律 | 工具 handler 内可硬拦截；另有 `useHumanInTheLoop` |
| 可审计性 | 弱 | 强（工具调用即为审计单元） |
| 生成式 UI | 无 | 有（A2UI / Generative UI / 建议 pills / 工具渲染插槽） |

**结论**：CopilotKit 在可控性、可审计性、交互表现上明显更强，代价是**能力必须由人声明**，无法自动覆盖全站 UI。

### 3.2 风险清单

1. **功能回退**：直接替换后，未声明工具对应的页面操作能力会消失。55 个路由的全自动覆盖不会自动继承。
2. **Vue 版生态较小**：官方维护但用户量约为 React 版 1/42，遇到边缘问题可参考资料少。
3. **样式体系冲突**：`AGENTS.md:348` 规定暗色模式只放在 `webview/src/assets/dark.css`、**组件内禁止 `dark:` 前缀**。CopilotKit 自带 `styles.css` 与独立主题变量，不跟随本仓库设计系统，需要额外适配。
4. **包体积**：引入 `@copilotkit/vue` 会带进 `katex`、`@a2ui/web_core`、`@jetbrains/websandbox` 等较重的传递依赖。
5. **依赖版本漂移**：现有 `page-agent` lockfile 记录 `1.10.0` 而实际安装 `1.8.2`，说明前端依赖未严格执行 `npm ci`，切换时需一并收敛。

---

## 4. 建议方案

采用**混合方案**：CopilotKit 负责会话编排与 UI，把 page-agent 的 DOM 引擎降级为 CopilotKit 的一个前端工具。

```
CopilotKitProvider (selfManagedAgents + HttpAgent → /api/agui)
├── CopilotSidebar / CopilotPopup      ← 复用官方会话 UI
├── useCopilotReadable                 ← 复用现有 instructions.ts（按路由注入上下文）
├── useFrontendTool({ name: "isrvd_api" })   ← 调 REST API，走结构化工具（推荐主路径）
└── useFrontendTool({ name: "page_action" }) ← 调 @page-agent/page-controller 兜底
```

要点：

- **保留 `@page-agent/page-controller`**：它唯一依赖是 `ai-motion`，可独立引入，不受 `page-agent` 主包版本漂移影响。把它包成一个 `page_action` 工具，即保留了「自动操作任意 UI」的兜底能力。
- **优先结构化工具**：当前 `instructions.ts` 已写清各模块 REST API 路径规则，直接映射为工具比让 LLM 点 DOM 更稳、更可审计。
- **保留 `/api/agent/*` 代理**：不破坏现有 OpenAI 兼容用法，新增 `/api/agui` 并行提供 AG-UI 流。
- **SSE 编码选型**：优先自写约 150 行编码器（事件类型少、结构稳定），规避 0.0.0 伪版本依赖；若后续官方发布正式版再切换。

## 5. 落地状态

- [x] 分支 `feature/copilotkit`
- [x] Go：新增 `POST /api/agui`，将 `config.Agent` 指向的 OpenAI 兼容流式响应翻译为 AG-UI 事件
- [x] 前端：`@copilotkit/vue` 的 `CopilotKitProvider` + `CopilotSidebar` 替换 `page-agent.vue`
- [x] 前端：`instructions.ts` 作为 `useCopilotReadable` 数据源（按 `router.currentRoute` 注入）
- [x] 前端：`useFrontendTool` 暴露 `isrvd_api`（REST）与 `page_action`（DOM 操作）
- [x] 文档：`docs/references/agent.md`、`docs/SKILL.md`、README 技术栈

### 工具分工

| 工具 | 用途 | 说明 |
| --- | --- | --- |
| `isrvd_api` | 调 REST API | 有接口支撑时优先使用，稳定且可审计 |
| `page_action` | 直接操作页面 | 无对应接口或需走界面流程时使用；由 `@page-agent/page-controller` 驱动 |

`page_action` 的用法与 page-agent 一致：先 `read` 拿到带序号的可交互元素列表，再按序号 `click` / `input` / `select` / `scroll`，`javascript` 作为兜底。

### UI 形态

- 右侧 `CopilotSidebar`（480px，`position: fixed`），对话形式，回复经 AG-UI 事件流逐块渲染
- 开合按钮在顶栏（经 `toggle-button` 插槽 + Pinia store 桥接），侧栏默认关闭
- 侧栏从顶栏（4rem）下方开始，不遮挡 header 右侧的用户菜单；主内容不让位（已禁用侧栏注入的 `body margin-inline-end`）
- Provider 在应用根**同步**挂载：composable 的 provide/inject 依赖它，异步加载会导致子树注入失败整页崩溃
- JWT 经 Provider 的 `headers` prop 注入每个 agent 请求

### 验证

- Go 侧：用真实 `@ag-ui/client` 打 `/api/agui`，文本流与工具调用两条路径的事件序列、参数分片拼接、消息累积均正确
- 前端：在真实 Chrome（puppeteer）中验证 `page_action` 全链路 —— `read` 返回带序号元素树、`input` 写入文本、`click` 触发页面真实响应、再次 `read` 能看到更新后的内容
- 集成：isrvd 完整二进制（前端 embed）+ 模拟 OpenAI 上游，浏览器实测「顶栏按钮开合 → 发消息 → 流式回复显示」全程通过，侧栏几何（top=64px、480px 宽、不让位）符合设计，console 无错误
- 门禁：`go build` / `go vet` / `vue-tsc` / eslint / style review / 生产构建全部通过

### 排障记录（供后来者）

1. **`data: data: {"ty"...` 双重前缀**：`httpd.NewEventWriter` 一次 `Write` 自动加 `data:` 前缀，Encoder 不能自己再写 `data:`，只输出 JSON 载荷即可
2. **`useCopilotKit must be used within CopilotKitProvider`**：Provider 与 composable 必须是不同组件（Vue inject 不能取到自己 provide 的值）；且 Provider 要同步挂载，不能 `defineAsyncComponent`
3. **public/assets 陈旧产物**：Go 二进制 embed `public/`，改前端后必须重新 `go build`，否则浏览器加载旧 bundle；调试时先 `curl /` 看引用的 hash 与磁盘是否一致，并 `pkill` 掉所有旧 isrvd 进程
4. **styles.css 未引入**：`@copilotkit/vue/styles.css` 必须在 `main.ts` import，否则侧栏宽度/定位样式缺失（表现为全宽覆盖）
5. **`Failed to construct 'URL'`**：`getPageInstruction` 只接受完整 URL；hash 路由下 `new URL(href).pathname` 恒为 `/`，页面匹配实际发生在 hash 段（已修复兼容两种入参）

### 遗留事项

- CopilotKit 自带样式不跟随仓库设计系统（`AGENTS.md` 规定暗色模式只放 `dark.css`、组件内禁用 `dark:` 前缀），需额外适配
- `page_action` 的 `javascript` 动作允许 LLM 在页面执行任意 JS。调用者已通过 JWT 与 `agent` 权限校验，且本身具备 Web Shell 等等效能力，不构成提权；指令中已限制其仅作兜底

> 仓库规范禁止提交 `*_test.*` 文件，故迁移验证以手工冒烟为主。
