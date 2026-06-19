// ─── 审计日志 ───

export interface AuditLog {
    timestamp: string
    username: string
    method: string
    uri: string
    body: string
    ip: string
    statusCode: number
    success: boolean
    duration: number
}

// ─── 系统配置（复用后端结构） ───

export interface ServerConfig {
    listenAddr: string
    rootDirectory: string
    maxUploadSize: number
    allowedOrigins: string[]
    // 写入时为空表示保留原值（不通过 JSON 返回）
    jwtSecret?: string
    jwtExpiration: number
    openapi: boolean
    debug: boolean
}

export interface THAConfig {
    enabled: boolean
    headerName: string
    trustedCIDRs: string[]
}

export interface OIDCConfig {
    enabled: boolean
    only: boolean
    issuerUrl: string
    clientId: string
    // 写入时为空表示保留原值（不通过 JSON 返回）
    clientSecret?: string
    redirectUrl: string
    usernameClaim: string
    scopes: string[]
    // OIDC 登录按钮自定义名称，留空时使用默认文案
    loginLabel?: string
}

export interface PasskeyConfig {
    enabled: boolean
    rpName: string
    rpId: string
    rpOrigins: string[]
    timeout: number
}

export interface AgentConfig {
    model: string
    baseUrl: string
    // 写入时为空表示保留原值（不通过 JSON 返回）
    apiKey?: string
}

export interface ApisixConfig {
    adminUrl: string
    // 写入时为空表示保留原值（不通过 JSON 返回）
    adminKey?: string
}

export interface CaddyConfig {
    adminUrl: string
}

export interface DockerRegistry {
    name: string
    description?: string
    url: string
    username?: string
    // 写入时为空表示保留原值（不通过 JSON 返回）
    password?: string
}

export interface DockerConfig {
    host: string
    containerRoot: string
    registries?: DockerRegistry[]
}

export interface MonitorConfig {
    interval: number  // 采集间隔（秒），合法值：5/15/30/60；其他值均视为禁用
}

export interface MarketplaceConfig {
    url: string
}

export interface LinkConfig {
    label: string
    url: string
    icon?: string
}

export interface AllConfig {
    server: ServerConfig
    tha: THAConfig
    oidc: OIDCConfig
    passkey: PasskeyConfig
    agent: AgentConfig
    apisix: ApisixConfig
    caddy: CaddyConfig
    docker: DockerConfig
    monitor: MonitorConfig
    marketplace: MarketplaceConfig
    links: LinkConfig[]
}
