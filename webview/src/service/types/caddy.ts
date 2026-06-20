// ─── Caddy 相关 ───

// 前端表单内部使用的 handler 类型标识（不提交到后端）
export type CaddyHandlerKind = 'reverse_proxy' | 'file_server' | 'static_response' | 'rewrite' | 'raw'

// 单条请求头/响应头操作（前端表单内部使用）
export interface CaddyHeaderOp {
    op: 'set' | 'add' | 'delete'
    field: string
    value: string
}

// Caddy 原生 MatchSet（对应 routes[i].match[] 的单个元素）
export interface CaddyMatchSet {
    host?: string[]
    path?: string[]
    method?: string[]
    header?: Record<string, string[]>
    protocol?: string
    [key: string]: unknown   // 允许其他 Caddy 扩展匹配器
}

// Caddy 原生 Handler（对应 routes[i].handle[] 的单个元素）
export interface CaddyHandler {
    handler: string          // 模块名，如 reverse_proxy / file_server / headers 等
    [key: string]: unknown   // 各模块自有字段
}

// reverse_proxy transport 子结构
export interface CaddyTransport {
    protocol?: string        // http / fastcgi
    root?: string            // fastcgi 文档根目录
    dial_timeout?: string
    response_header_timeout?: string
    write_timeout?: string
}

// reverse_proxy 内嵌 rewrite 子结构
export interface CaddyProxyRewrite {
    method?: string
    uri?: string
    strip_path_prefix?: string
    strip_path_suffix?: string
    uri_substring?: Array<{ find: string; replace: string }>
}

// reverse_proxy handler
export interface CaddyHandlerReverseProxy extends CaddyHandler {
    handler: 'reverse_proxy'
    upstreams?: Array<{ dial: string }>
    transport?: CaddyTransport
    rewrite?: CaddyProxyRewrite
}

// file_server handler
export interface CaddyHandlerFileServer extends CaddyHandler {
    handler: 'file_server'
    root?: string
    browse?: Record<string, unknown>
}

// static_response handler
export interface CaddyHandlerStaticResponse extends CaddyHandler {
    handler: 'static_response'
    status_code?: number | string
    body?: string
}

// rewrite handler
export interface CaddyHandlerRewrite extends CaddyHandler {
    handler: 'rewrite'
    uri?: string
    strip_path_prefix?: string
    strip_path_suffix?: string
    uri_substring?: Array<{ find: string; replace: string }>
}

// headers 中间件 handler
export interface CaddyHandlerHeaders extends CaddyHandler {
    handler: 'headers'
    request?: Record<string, unknown>
    response?: Record<string, unknown>
}

// CaddyRoute 路由视图（后端返回，含 index）
export interface CaddyRoute {
    index: number
    group?: string
    match?: CaddyMatchSet[]
    handle?: CaddyHandler[]
    terminal?: boolean
    '@id'?: string
}

// CaddyRouteUpsert 创建/更新请求体（直接提交 Caddy 原生 JSON，不含 index）
export interface CaddyRouteUpsert {
    group?: string
    match?: CaddyMatchSet[]
    handle?: CaddyHandler[]
    terminal?: boolean
    '@id'?: string
}

// 概览
export interface CaddyInfo {
    adminUrl: string
    servers: number
    routes: number
    certs: number
    hasTls: boolean
    available: boolean
}

// handler kind 卡片
export interface CaddyHandlerKindCard {
    value: CaddyHandlerKind
    title: string
    desc: string
    icon: string
    tone: 'indigo' | 'emerald' | 'amber' | 'violet' | 'rose' | 'slate'
}

// ─── Basic Auth ───

export interface CaddyBasicAuthUser {
    username: string
}

export interface CaddyBasicAuthRoute {
    index: number             // 路由下标（主键）
    name: string              // 路由 @id（展示用）
    realm: string             // HTTP Basic realm
    forwardHeader: string     // 传递用户名的请求头；空表示未开启
    users: CaddyBasicAuthUser[]
    handlers: CaddyHandler[]  // 其余 handler（只读展示）
}

export interface CaddyBasicAuthUserCreate {
    username: string
    password: string
    realm?: string
    forwardHeader?: string    // 非空时注入 X-Remote-User 等 header
}

export interface CaddyBasicAuthConfigUpdate {
    realm: string
    forwardHeader: string
}

// ─── 全局选项 ───

export interface CaddyGlobal {
    logLevel?: string       // 全局日志级别：DEBUG / INFO / WARN / ERROR
    logFormat?: string      // 日志格式：json / console
    email?: string          // ACME 注册邮箱
    acmeCA?: string         // 自定义 ACME 目录 URL，留空使用 Let's Encrypt
    localCerts?: boolean    // 使用本地自签证书（internal issuer），不走 ACME
    onDemandTLS?: boolean   // 启用 on_demand TLS（连接时动态申请证书）
    onDemandAsk?: string    // ask 鉴权端点 URL，Caddy v2.8+ 必须配置以防滥用
    autoHttpsDisable?: boolean          // 全局禁用自动 HTTPS
    autoHttpsDisableRedirects?: boolean // 禁用 HTTP→HTTPS 自动跳转
    gracePeriod?: string    // 优雅关闭等待时间，例如 10s
}

// ─── SSL 证书 ───

export type CaddyCertSource = 'file' | 'pem' | 'automate' | 'cached'

export interface CaddyCert {
    key?: string             // 复合主键 <source>-<index>，仅响应使用（cached 无此字段）
    source: CaddyCertSource
    subject?: string         // 证书域名：automate/cached 为目标域名，file/pem 从证书 CN 解析
    certificate?: string     // file：证书文件路径；pem：证书 PEM 文本
    keyContent?: string      // file：私钥文件路径；pem：私钥 PEM 文本（响应时不返回）
    tags?: string[]          // Caddy 内部标签（file/pem 可选）
    format?: string          // 证书格式，仅 file 使用（默认 PEM）
    // 由证书内容解析填充，automate 类型无证书文件故留空
    issuer?: string          // 签发机构 Common Name
    notBefore?: string       // 证书生效时间
    notAfter?: string        // 证书过期时间
    sans?: string[]          // Subject Alternative Names（DNS）
}

export interface CaddyCertSourceCard {
    value: CaddyCertSource
    title: string
    desc: string
    icon: string
    tone: 'indigo' | 'emerald' | 'amber' | 'cyan'
}
