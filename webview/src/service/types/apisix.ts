// ─── Apisix 相关 ───

export interface ApisixUpstreamNode {
    host?: string
    port?: number | string
    weight?: number
}

export interface ApisixUpstreamConfig {
    type?: string
    nodes?: ApisixUpstreamNode[] | Record<string, number>
    [key: string]: unknown
}

export interface ApisixRouteTimeout {
    connect?: number
    send?: number
    read?: number
}

export interface ApisixRoute {
    id?: string
    name: string
    uri?: string
    uris?: string[]
    host?: string
    hosts?: string[]
    desc?: string
    status: number
    priority: number
    enable_websocket: boolean
    plugin_config_id?: string
    upstream_id?: string
    upstream?: ApisixUpstreamConfig
    plugins?: Record<string, unknown>
    timeout?: ApisixRouteTimeout
    consumers?: string[]
    create_time?: number
    update_time?: number
}

export interface ApisixConsumer {
    username: string
    desc: string
    plugins?: Record<string, unknown>
    create_time: number
    update_time: number
}

export interface ApisixCreateConsumerRequest {
    username: string
    desc?: string
}

export interface ApisixUpdateConsumerRequest {
    desc?: string
}

export interface ApisixPluginConfig {
    id: string
    desc: string
    plugins?: Record<string, unknown>
    create_time: number
    update_time: number
}

export interface ApisixUpstream {
    id: string
    name: string
    desc: string
    type: string
    create_time: number
    update_time: number
}

export interface ApisixRevokeWhitelistRequest {
    routeId: string
    consumer: string
}

// Apisix 概览统计
export interface ApisixInfo {
    routes: number
    consumers: number
    whitelist: number
    [key: string]: number
}
