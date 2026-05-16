// ─── Caddy 相关 ───

export type CaddyHandlerKind = 'reverse_proxy' | 'file_server' | 'static_response' | 'raw'

export interface CaddyMatchForm {
    hosts?: string[]
    paths?: string[]
    methods?: string[]
}

export interface CaddyHandlerForm {
    kind: CaddyHandlerKind
    upstreams?: string[]
    root?: string
    browse?: boolean
    statusCode?: number
    body?: string
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
    hasTls: boolean
    available: boolean
}

// handler kind 卡片
export interface CaddyHandlerKindCard {
    value: CaddyHandlerKind
    title: string
    desc: string
    icon: string
    tone: 'indigo' | 'emerald' | 'amber' | 'slate'
}

// ─── TLS 证书 ───

export type CaddyCertSource = 'file' | 'pem' | 'automate'

export interface CaddyCert {
    key?: string             // 复合主键 <source>-<index>，仅响应使用
    source: CaddyCertSource
    certificate?: string     // file: 路径；pem: PEM 文本
    keyContent?: string      // 私钥（路径或 PEM 文本）
    tags?: string[]
    format?: string          // 仅 file 使用
    subject?: string         // 仅 automate 使用
}

export interface CaddyCertSourceCard {
    value: CaddyCertSource
    title: string
    desc: string
    icon: string
    tone: 'indigo' | 'emerald' | 'amber'
}
