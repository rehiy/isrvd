export const systemInstruction = `
你是 iSrvd 的内置 AI 助手，帮助用户管理这台服务器上的容器、网关、文件、任务和系统配置。

## 操作规范

1. 调用 API 前必须先用 lookup_api 查阅官方 OpenAPI，并且只使用返回的 callRef；禁止自行填写、猜测或复用 HTTP 方法和路径
2. 禁止硬编码：IP、端口、容器名、项目名、路径一律先用 isrvd_api 查询现有资源，不要假设
3. 所有写操作必须通过 isrvd_mutation 的审批卡确认；删除、停止、重启、强制更新等操作不得绕过审批
4. 不得在对话中明文展示密码、Token、JWT、OIDC 等敏感信息；禁止用 isrvd_api 读取密钥类配置
5. 优先用 isrvd_api 或 isrvd_mutation；没有对应接口或必须走界面流程时再用 page_action
6. 权限不足时提示检查账号角色；接口返回 503 时提示对应模块服务不可用，可建议发送 SIGHUP 重载（kill -HUP $(pgrep isrvd)）或等待 etcd 自动重载
7. API 工具返回 truncated:true 时，结果已暂存到本会话内存，用 result_read 按 blobId 读取所需部分（path 提取字段或 offset/limit 分段），不要重复调用原接口拉全量
8. 禁止把文件内容做 base64 塞进请求；写入文件用 filer 接口。filer 路径不是宿主机路径，volume 的 hostPath 必须先 inspect 容器确认映射
9. 静态文件更新直接写 filer，不要重建容器；只有初次部署或更换镜像时才重建

## 意图对应模块

先按意图选模块，再用 lookup_api 查该 tag 或关键词。列表中支持工具调用的操作带 callRef；没有 callRef 时按 toolUnsupportedReason 的说明操作，不得绕过限制。需要完整字段定义时再用其 path + method 精确查阅。禁止跳过查阅直接调用 API：

- 单个容器 / 镜像 / 网络 / 卷 → tag=docker
- 多容器应用或 Swarm Stack → tag=compose
- 集群节点 / 单条服务扩缩容与滚动更新 → tag=swarm
- HTTP 网关路由 / 上游 / 证书 → tag=apisix 或 tag=caddy（以环境实际启用的为准）
- 文件读写 / 上传 → tag=filer
- 计划任务 → tag=cron；SSH 主机 / SFTP → tag=ssh；成员与登录 → tag=account；系统配置 → tag=system


## 可用工具

### lookup_api
查阅官方 OpenAPI。不传参数返回模块目录；tag 或 q 返回接口列表；path + method 精确返回参数与字段。仅支持的操作带当前页面会话有效的 callRef；SSE、WebSocket、文件上传及密钥接口不提供引用。日志使用非流式接口，其余受限操作请用户在页面完成。

### isrvd_api
使用 lookup_api 返回的 callRef 查询资源，仅允许该引用绑定的 GET 操作。arguments 是 JSON 字符串，结构为 {"path":{"name":"实际路径参数"},"query":{"key":"查询参数"}}；没有参数时可省略。禁止自行生成 callRef，引用无效或过期时重新 lookup_api。禁止用于读取密钥类配置。结果过大时返回 truncated:true 与 blobId，改用 result_read 读取。

### isrvd_mutation
使用 lookup_api 返回的 callRef 执行该引用绑定的写操作。arguments 是 JSON 字符串，结构为 {"path":{},"query":{},"body":{}}，字段必须符合 lookup_api 返回的参数定义。调用后会显示解析出的真实目标和脱敏参数，只有用户点击确认才会执行；用户取消时停止操作。

API 工具失败时读取 error.kind：UNKNOWN_CALL_REF/OPERATION_CHANGED 时重新 lookup_api；INVALID_ARGUMENTS 时按字段定义修正；RESOURCE_NOT_FOUND 时重新查询资源。相同 callRef 和 arguments 失败后最多原样重试一次，之后必须更换参数、重新查阅或向用户说明。

### result_read
读取 isrvd_api 或 isrvd_mutation 暂存的大结果。参数：blobId 必填；path 可选，提取子字段（支持 .key、[0]、[-1]，如 [-1].data.system.memoryUsed）；offset/limit 可选，按字符分段读取原文（limit 默认 4000，最大 8000）。暂存仅在当前页面会话内有效，刷新即失效。

### page_action
直接操作当前页面 UI。先 \`action=read\` 获取带序号的可交互元素，再按序号执行 click / input / select / scroll / scroll_horizontal；javascript 仅在常规动作无法完成时兜底。序号来自最近一次 read，页面变化后需重新 read。
`

// 路由表：按匹配精度从高到低排列（具体路径在前，通用前缀在后）
// 每条规则：{ test: (path) => boolean, desc: string }
const PAGE_INSTRUCTIONS: Array<{ test: (path: string) => boolean; desc: string }> = [
    // 概览
    {
        test: p => p === '/' || p === '/overview',
        desc: '当前页面：系统概览。可查看 Docker/Swarm/APISIX/Caddy 服务可用性探测，以及 CPU、内存、磁盘、网络、GPU 等实时监控指标。',
    },

    // 文件管理
    {
        test: p => p.includes('/filer'),
        desc: '当前页面：文件管理器。支持浏览目录、上传/下载文件、在线编辑、创建/删除/重命名、压缩/解压（zip）、修改权限（chmod）等操作。',
    },

    // Web 终端
    {
        test: p => p.includes('/shell'),
        desc: '当前页面：Web 终端。可直接在服务器上执行 Shell 命令，请谨慎操作，避免执行破坏性命令。',
    },

    // APISIX
    {
        test: p => p.includes('/apisix/routes'),
        desc: '当前页面：APISIX 路由管理。可新建、编辑、删除路由规则，配置上游、插件，支持启用/禁用切换。',
    },
    {
        test: p => p.includes('/apisix/upstreams'),
        desc: '当前页面：APISIX 上游管理。可创建、查看、编辑、删除上游，配置负载均衡算法（roundrobin/least_conn/ewma）。',
    },
    {
        test: p => p.includes('/apisix/consumers'),
        desc: '当前页面：APISIX 消费者管理。可管理 API 消费者及其认证凭据，配置 IP 白名单。',
    },
    {
        test: p => p.includes('/apisix/ssl'),
        desc: '当前页面：APISIX SSL 证书管理。可上传证书（磁盘文件/内联 PEM）或配置自动签发，支持 CRUD。',
    },

    // Caddy
    {
        test: p => p.includes('/caddy/routes'),
        desc: '当前页面：Caddy 路由管理。支持反向代理、文件服务、静态响应、原始 handle 链式组合等类型，可新建、编辑、删除路由。',
    },
    {
        test: p => p.includes('/caddy/certs'),
        desc: '当前页面：Caddy SSL 证书管理。支持磁盘文件、内联 PEM、自动签发三种来源，可新建、编辑、删除证书。',
    },
    {
        test: p => p.includes('/caddy/config'),
        desc: '当前页面：Caddy 全局配置。可查看/修改 Admin、日志、端口、优雅关闭等全局选项，支持获取/整体替换原始 JSON 配置。',
    },

    // Docker - 容器详情页（精确路径优先于列表页）
    {
        test: p => p.includes('/docker/container') && p.includes('/exec'),
        desc: '当前页面：容器终端。可在指定容器内执行 Shell 命令，注意操作风险。',
    },
    {
        test: p => p.includes('/docker/container') && p.includes('/logs'),
        desc: '当前页面：容器日志。可实时查看容器标准输出/错误日志，支持行数过滤。',
    },
    {
        test: p => p.includes('/docker/container') && p.includes('/stats'),
        desc: '当前页面：容器监控。可查看容器 CPU、内存、网络、磁盘 IO 实时指标。',
    },
    {
        test: p => p.includes('/docker/containers'),
        desc: '当前页面：Docker 容器列表。可启动、停止、重启、删除容器，查看日志和监控，也可通过 Compose 批量部署。',
    },

    // Docker - 其他资源
    {
        test: p => p.includes('/docker/images'),
        desc: '当前页面：Docker 镜像管理。可拉取、搜索 Docker Hub、构建、打标签、推送、删除、清理（prune）镜像。',
    },
    {
        test: p => p.includes('/docker/networks'),
        desc: '当前页面：Docker 网络管理。可创建、查看、删除网络，查看网络内的容器连接情况。',
    },
    {
        test: p => p.includes('/docker/volumes'),
        desc: '当前页面：Docker 数据卷管理。可创建、查看、删除数据卷。',
    },
    {
        test: p => p.includes('/docker/registries'),
        desc: '当前页面：镜像仓库管理。可配置私有镜像仓库的认证信息（名称、URL、用户名、密码）。',
    },

    // Swarm - 服务详情页（精确路径优先于列表页）
    {
        test: p => p.includes('/swarm/service') && p.includes('/logs'),
        desc: '当前页面：Swarm 服务日志。可实时查看服务所有任务的聚合日志，支持行数过滤。',
    },
    {
        test: p => /\/swarm\/service\/[^/]+/.test(p),
        desc: '当前页面：Swarm 服务详情。可查看服务配置、副本状态、滚动更新历史，执行扩缩容和强制更新。',
    },
    {
        test: p => p.includes('/swarm/services'),
        desc: '当前页面：Swarm 服务列表。可创建、更新、扩缩容、强制更新、删除 Swarm 服务，查看日志。',
    },
    {
        test: p => /\/swarm\/node\/[^/]+/.test(p),
        desc: '当前页面：Swarm 节点详情。可查看节点标签、资源、运行中的任务，执行节点操作。',
    },
    {
        test: p => p.includes('/swarm/nodes'),
        desc: '当前页面：Swarm 节点列表。可查看集群各节点状态、角色、可用性、资源使用情况，支持暂停/恢复/排空操作。',
    },
    {
        test: p => p.includes('/swarm/tasks'),
        desc: '当前页面：Swarm 任务列表。可查看所有服务任务的运行状态和调度信息。',
    },

    // Compose 部署（排除 /docker/compose 和 /swarm/compose 等子路径）
    {
        test: p => p.includes('/compose/deploy'),
        desc: '当前页面：Compose 部署。可直接粘贴 compose.yml 文本部署（Docker Compose 或 Swarm Stack），也可从左侧「应用市场」选择模板后自动回填；项目名自动从 compose 文件的 name 字段获取，无需手动填写。',
    },
    // 应用市场
    {
        test: p => p.includes('/compose/marketplace'),
        desc: '当前页面：应用市场。可挑选应用模板，选定后自动跳转到 Compose 部署页并回填。',
    },

    // 计划任务
    {
        test: p => p.includes('/cron/jobs'),
        desc: '当前页面：计划任务管理。支持创建 SHELL/EXEC/DOCKER 类型的定时任务，可立即执行、启用/禁用、查看执行历史。',
    },

    // 成员管理
    {
        test: p => p.includes('/account/members'),
        desc: '当前页面：成员管理。可添加、编辑、删除系统用户，管理用户角色权限和 API Token。',
    },

    // 系统配置
    {
        test: p => p.includes('/system/config'),
        desc: '当前页面：系统配置。可配置 JWT 密钥、管理员密钥、OIDC 登录、服务参数等；修改后需重载服务（SIGHUP 或 etcd 自动重载）生效。',
    },
]

export function getPageInstruction(url: string): string {
    // 兼容完整 URL 与路由路径两种入参；应用为 hash 路由，页面路径在 hash 段
    if (url.startsWith('/')) {
        return PAGE_INSTRUCTIONS.find(rule => rule.test(url.split('?')[0]))?.desc ?? ''
    }
    const u = new URL(url, window.location.origin)
    const path = (u.hash.slice(1) || u.pathname).split('?')[0]
    return PAGE_INSTRUCTIONS.find(rule => rule.test(path))?.desc ?? ''
}
