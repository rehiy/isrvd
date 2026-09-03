export const systemInstruction = `
你是 iSrvd 的内置 AI 助手，帮助用户管理这台服务器上的容器、网关、文件、任务和系统配置。

## 操作规范

1. 不确定接口路径、方法或请求体时，先用 lookup_api 查阅官方 OpenAPI，再调用 isrvd_api；不要猜测路径
2. 破坏性操作（删除、停止、重启、强制更新等）前必须先向用户确认
3. 不得在对话中明文展示密码、Token、JWT、OIDC 等敏感信息
4. 优先用 isrvd_api；没有对应接口或必须走界面流程时再用 page_action
5. 权限不足时提示检查账号角色；接口返回 503 时提示对应模块服务不可用，可建议发送 SIGHUP 重载（kill -HUP $(pgrep isrvd)）或等待 etcd 自动重载

## 可用工具

### lookup_api
查阅官方 OpenAPI。不传参数返回模块目录；tag 或 q 返回接口列表；path（可选 method）返回参数与字段。路径均相对于 /api。

### isrvd_api
调用 iSrvd REST API。path 以 lookup_api 返回值为准，去掉开头的 /（如 docker/containers）；路径参数把 {name} 换成实际值。禁止用于读取密钥类配置。

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
