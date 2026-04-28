import type {
    ApisixRoute,
    ApisixRouteUpstreamFormNode,
    ApisixRouteUpstreamMode,
    ApisixUpstreamConfig,
    ApisixUpstreamHashOn,
    ApisixUpstreamType,
    ApisixUpstreamNode
} from '@/service/types'

const parseNodeKey = (key: string): ApisixUpstreamNode => {
    if (!key) return { host: '', port: '' }
    const idx = key.lastIndexOf(':')
    if (idx <= 0) return { host: key, port: '' }
    const port = key.slice(idx + 1)
    return {
        host: key.slice(0, idx),
        port: /^\d+$/.test(port) ? Number(port) : port
    }
}

const DEFAULT_UPSTREAM_TYPE: ApisixUpstreamType = 'roundrobin'

export const normalizeUpstreamType = (type?: string): ApisixUpstreamType => {
    if (type === 'chash' || type === 'ewma' || type === 'least_conn') return type
    return DEFAULT_UPSTREAM_TYPE
}

export const normalizeUpstreamNodes = (upstream?: ApisixUpstreamConfig): ApisixUpstreamNode[] => {
    const nodes = upstream?.nodes
    if (!nodes) return []

    if (Array.isArray(nodes)) {
        return nodes
            .map(node => ({
                host: node.host || '',
                port: node.port || '',
                weight: typeof node.weight === 'number' && node.weight >= 0 ? node.weight : 1
            }))
            .filter(node => node.host || node.port)
    }

    if (typeof nodes === 'object') {
        return Object.entries(nodes).map(([key, weight]) => {
            const parsed = parseNodeKey(key)
            return {
                ...parsed,
                weight: typeof weight === 'number' && weight >= 0 ? weight : 1
            }
        })
    }

    return []
}

export const parseUpstreamNode = (upstream?: ApisixUpstreamConfig): { host: string; port: number | string } => {
    const [first] = normalizeUpstreamNodes(upstream)
    return { host: first?.host || '', port: first?.port || '' }
}

export const normalizeUpstreamFormNodes = (upstream?: ApisixUpstreamConfig): ApisixRouteUpstreamFormNode[] => {
    const nodes = normalizeUpstreamNodes(upstream).map(node => ({
        host: String(node.host || ''),
        port: String(node.port || ''),
        weight: typeof node.weight === 'number' && node.weight >= 0 ? node.weight : 1
    }))

    return nodes.length > 0 ? nodes : [{ host: '', port: '', weight: 1 }]
}

export const detectRouteUpstreamMode = (route?: Pick<ApisixRoute, 'upstream_id' | 'upstream'>): ApisixRouteUpstreamMode => {
    if (route?.upstream_id) return 'upstream_id'
    if (normalizeUpstreamNodes(route?.upstream).length > 0) return 'nodes'
    return 'none'
}

export const formatRouteUpstreamSummary = (route: Pick<ApisixRoute, 'upstream_id' | 'upstream'>): string => {
    if (route.upstream_id) return `引用上游 #${route.upstream_id}`

    const nodes = normalizeUpstreamNodes(route.upstream)
    if (nodes.length === 0) return '未配置'

    const upstreamType = normalizeUpstreamType(route.upstream?.type)
    const first = nodes[0]
    const firstLabel = `${first.host || '-'}:${first.port || '-'}`
    if (nodes.length === 1) return `${upstreamType} · ${firstLabel}`
    return `${upstreamType} · ${firstLabel} 等 ${nodes.length} 个节点`
}

/** 仅返回负载均衡策略名称，引用上游时返回 null */
export const formatRouteUpstreamType = (route: Pick<ApisixRoute, 'upstream_id' | 'upstream'>): string | null => {
    if (route.upstream_id) return null
    const nodes = normalizeUpstreamNodes(route.upstream)
    if (nodes.length === 0) return null
    return normalizeUpstreamType(route.upstream?.type)
}

/** 仅返回节点信息，引用上游时返回完整摘要 */
export const formatRouteUpstreamNodes = (route: Pick<ApisixRoute, 'upstream_id' | 'upstream'>): string => {
    if (route.upstream_id) return `引用上游 #${route.upstream_id}`
    const nodes = normalizeUpstreamNodes(route.upstream)
    if (nodes.length === 0) return '未配置'
    const first = nodes[0]
    const firstLabel = `${first.host || '-'}:${first.port || '-'}`
    if (nodes.length === 1) return firstLabel
    return `${firstLabel} 等 ${nodes.length} 个节点`
}

interface RouteFormData {
    name: string
    desc: string
    status: number
    priority?: number
    enable_websocket: boolean
    plugin_config_id?: string
    plugins?: Record<string, unknown>
    uris: string
    hosts: string
    upstream_mode: ApisixRouteUpstreamMode
    upstream_type?: ApisixUpstreamType
    upstream_id?: string
    upstream_nodes: ApisixRouteUpstreamFormNode[]
    upstream_hash_on?: ApisixUpstreamHashOn
    upstream_key?: string
    timeout_connect?: string | number
    timeout_send?: string | number
    timeout_read?: string | number
}

const buildInlineUpstream = (
    nodes: ApisixRouteUpstreamFormNode[],
    baseUpstream?: ApisixUpstreamConfig | null,
    hashOn?: ApisixUpstreamHashOn,
    key?: string
): ApisixUpstreamConfig | undefined => {
    const normalizedNodes: { host: string; port: number; weight: number }[] = []
    for (const node of nodes) {
        const host = node.host.trim()
        const port = String(node.port).trim()
        if (host && port) normalizedNodes.push({ host, port: Number(port), weight: Number(node.weight) >= 0 ? Number(node.weight) : 1 })
    }
    if (!normalizedNodes.length) return undefined

    const type = String(baseUpstream?.type || 'roundrobin')
    const result: ApisixUpstreamConfig = { ...(baseUpstream || {}), type, nodes: normalizedNodes }

    if (type === 'chash') {
        result.hash_on = hashOn || 'vars'
        result.key = key || 'remote_addr'
    } else {
        delete result.hash_on
        delete result.key
    }

    return result
}

export const buildRoutePayload = (formData: RouteFormData, baseUpstream?: ApisixUpstreamConfig | null): ApisixRoute => {
    const payload: ApisixRoute = {
        name: formData.name.trim(),
        desc: formData.desc.trim(),
        status: formData.status,
        priority: formData.priority ?? 0,
        enable_websocket: formData.enable_websocket,
        plugin_config_id: formData.plugin_config_id || '',
        plugins: formData.plugins || {}
    }
    const urisArr = formData.uris.split('\n').map((s: string) => s.trim()).filter(Boolean)
    if (urisArr.length > 1) payload.uris = urisArr
    else if (urisArr.length === 1) payload.uri = urisArr[0]
    const hostsArr = formData.hosts.split('\n').map((s: string) => s.trim()).filter(Boolean)
    if (hostsArr.length > 1) payload.hosts = hostsArr
    else if (hostsArr.length === 1) payload.host = hostsArr[0]

    if (formData.upstream_mode === 'upstream_id' && formData.upstream_id?.trim()) {
        payload.upstream_id = formData.upstream_id.trim()
    }

    if (formData.upstream_mode === 'nodes') {
        const inlineUpstream = buildInlineUpstream(
            formData.upstream_nodes,
            {
                ...(baseUpstream || {}),
                type: normalizeUpstreamType(formData.upstream_type || baseUpstream?.type)
            },
            formData.upstream_hash_on,
            formData.upstream_key
        )
        if (inlineUpstream) payload.upstream = inlineUpstream
    }

    const connect = Number(formData.timeout_connect) || 0
    const send = Number(formData.timeout_send) || 0
    const read = Number(formData.timeout_read) || 0
    if (connect > 0 || send > 0 || read > 0) {
        payload.timeout = { connect: connect || undefined, send: send || undefined, read: read || undefined }
    }

    return payload
}
