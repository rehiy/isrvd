// ─── Caddy 相关 ───

export type CaddyHandlerKind = 'reverse_proxy' | 'file_server' | 'static_response' | 'rewrite' | 'raw'

export interface CaddyMatchForm {
    hosts?: string[]
    paths?: string[]
    methods?: string[]
    headers?: Record<string, string[]>  // 按请求头匹配，key 为头字段名，value 为匹配值列表
    protocol?: string                   // 匹配协议：http / https
}

// 单条请求头/响应头操作
export interface CaddyHeaderOp {
    op: 'set' | 'add' | 'delete'  // 操作类型
    field: string                  // 头字段名，如 X-Real-IP
    value: string                  // 值（delete 时留空）
}

export interface CaddyHandlerForm {
    kind: CaddyHandlerKind
    upstreams?: string[]
    fastcgi?: boolean      // 启用 FastCGI 传输（PHP-FPM 等）
    fastcgiRoot?: string   // FastCGI 文档根目录
    dialTimeout?: string   // 连接上游超时，如 10s
    readTimeout?: string   // 读取上游响应头超时，如 30s
    writeTimeout?: string  // 向上游写入请求超时，如 30s
    root?: string
    browse?: boolean
    statusCode?: number
    body?: string
    // rewrite
    rewriteUri?: string          // 完整 URI 替换（支持 Caddy 占位符）
    stripPathPrefix?: string     // 去掉路径前缀
    stripPathSuffix?: string     // 去掉路径后缀
    uriSubstringFind?: string    // 子串查找
    uriSubstringReplace?: string // 子串替换
    // headers
    requestHeaders?: CaddyHeaderOp[]   // 请求头操作列表
    responseHeaders?: CaddyHeaderOp[]  // 响应头操作列表
    raw?: unknown
}

export interface CaddyRoute {
    index: number
    group?: string
    match?: CaddyMatchForm
    handler?: CaddyHandlerForm
    terminal?: boolean
    id?: string
}

// 编辑/创建请求体（不带 index）
export interface CaddyRouteUpsert {
    group?: string
    match?: CaddyMatchForm
    handler: CaddyHandlerForm
    terminal?: boolean
    id?: string
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

// ─── TLS 证书 ───

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
