// ─── Apisix 相关 ───

export type ApisixRouteUpstreamMode = 'none' | 'nodes' | 'upstream_id'

export type ApisixUpstreamType = 'roundrobin' | 'chash' | 'ewma' | 'least_conn'

export type ApisixUpstreamHashOn = 'vars' | 'header' | 'cookie' | 'consumer' | 'vars_combinations'

export interface ApisixUpstreamNode {
    host?: string
    port?: number | string
    weight?: number
}

export interface ApisixUpstreamConfig {
    type?: ApisixUpstreamType | string
    nodes?: ApisixUpstreamNode[] | Record<string, number>
    hash_on?: ApisixUpstreamHashOn
    key?: string
    timeout?: ApisixRouteTimeout
    [key: string]: unknown
}

export interface ApisixRouteTimeout {
    connect?: number
    send?: number
    read?: number
}

export interface ApisixRouteUpstreamFormNode {
    host: string
    port: string
    weight: number
}

export interface ApisixRouteUpstreamModeCard {
    value: ApisixRouteUpstreamMode
    title: string
    desc: string
    icon: string
    tone: 'indigo' | 'emerald' | 'slate'
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
    plugins?: Record<string, unknown>
}

export interface ApisixUpdateConsumerRequest {
    desc?: string
    plugins?: Record<string, unknown>
}

export interface ApisixPluginConfig {
    id: string
    desc: string
    plugins?: Record<string, unknown>
    create_time: number
    update_time: number
}

export interface ApisixUpstream {
    id?: string
    name: string
    desc?: string
    type: ApisixUpstreamType | string
    nodes?: ApisixUpstreamNode[] | Record<string, number>
    hash_on?: ApisixUpstreamHashOn
    key?: string
    scheme?: string
    pass_host?: string
    upstream_host?: string
    retries?: number
    retry_timeout?: number
    timeout?: ApisixRouteTimeout
    create_time?: number
    update_time?: number
    [key: string]: unknown
}

export type ApisixCreateUpstreamRequest = ApisixUpstream

export type ApisixUpdateUpstreamRequest = ApisixUpstream

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
