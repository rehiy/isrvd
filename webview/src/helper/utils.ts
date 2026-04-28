import type {
    ApisixRoute,
    ApisixRouteUpstreamFormNode,
    ApisixRouteUpstreamMode,
    ApisixUpstreamConfig,
    ApisixUpstreamHashOn,
    ApisixUpstreamType,
    ApisixUpstreamNode
} from '@/service/types'

// 全局自动刷新间隔（毫秒），所有轮询定时器统一使用此常量
export const POLL_INTERVAL = 3000

export const TEXT_EXTENSIONS: string[] = [
    'txt', 'md', 'js', 'css', 'html', 'htm', 'json', 'xml', 'csv',
    'log', 'conf', 'ini', 'cfg', 'yaml', 'yml', 'php', 'py', 'go',
    'java', 'cpp', 'c', 'h', 'sql', 'sh', 'bat', 'env'
]

export const FILE_ICON_MAP: Record<string, string> = {
    'txt': 'fas fa-file-alt text-secondary',
    'md': 'fab fa-markdown text-dark',
    'pdf': 'fas fa-file-pdf text-danger',
    'doc': 'fas fa-file-word text-primary',
    'docx': 'fas fa-file-word text-primary',
    'xls': 'fas fa-file-excel text-success',
    'xlsx': 'fas fa-file-excel text-success',
    'ppt': 'fas fa-file-powerpoint text-warning',
    'pptx': 'fas fa-file-powerpoint text-warning',
    'zip': 'fas fa-file-archive text-warning',
    'rar': 'fas fa-file-archive text-warning',
    '7z': 'fas fa-file-archive text-warning',
    'tar': 'fas fa-file-archive text-warning',
    'gz': 'fas fa-file-archive text-warning',
    'jpg': 'fas fa-file-image text-info',
    'jpeg': 'fas fa-file-image text-info',
    'png': 'fas fa-file-image text-info',
    'gif': 'fas fa-file-image text-info',
    'bmp': 'fas fa-file-image text-info',
    'svg': 'fas fa-file-image text-info',
    'mp3': 'fas fa-file-audio text-success',
    'wav': 'fas fa-file-audio text-success',
    'mp4': 'fas fa-file-video text-danger',
    'avi': 'fas fa-file-video text-danger',
    'mov': 'fas fa-file-video text-danger',
    'js': 'fab fa-js-square text-warning',
    'html': 'fab fa-html5 text-danger',
    'css': 'fab fa-css3-alt text-primary',
    'php': 'fab fa-php text-purple',
    'py': 'fab fa-python text-info',
    'java': 'fab fa-java text-danger',
    'cpp': 'fas fa-file-code text-info',
    'c': 'fas fa-file-code text-info',
    'go': 'fas fa-file-code text-primary',
    'sql': 'fas fa-database text-secondary'
}

export const isEditableFile = (filename: string): boolean => {
    if (!filename) return false
    const ext = filename.split('.').pop()?.toLowerCase() ?? ''
    return TEXT_EXTENSIONS.includes(ext)
}

export const getFileIcon = (file: { isDir: boolean; name: string }): string => {
    if (file.isDir) {
        return 'fas fa-folder text-warning'
    }
    const ext = file.name.split('.').pop()?.toLowerCase() ?? ''
    return FILE_ICON_MAP[ext] || 'fas fa-file text-secondary'
}

export const formatFileSize = (bytes: number): string => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

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

export const formatTime = (timeString: string): string => {
    const date = new Date(timeString)
    return date.toLocaleDateString('zh-CN') + ' ' + date.toLocaleTimeString('zh-CN', { hour12: false })
}

export const hexToRgba = (hex: string, alpha: number): string => {
    const r = parseInt(hex.slice(1, 3), 16)
    const g = parseInt(hex.slice(3, 5), 16)
    const b = parseInt(hex.slice(5, 7), 16)
    return `rgba(${r},${g},${b},${alpha})`
}

export const downloadFile = (filename: string, data: BlobPart): void => {
    const url = window.URL.createObjectURL(new Blob([data]))
    const link = document.createElement('a')
    link.href = url
    link.download = filename
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(url)
}

/**
 * 复制文本到剪贴板
 * 优先使用 Clipboard API，降级到 execCommand
 * @returns 是否复制成功
 */
export const copyToClipboard = async (text: string): Promise<boolean> => {
    try {
        if (navigator.clipboard?.writeText) {
            await navigator.clipboard.writeText(text)
            return true
        }
        // 降级方案
        const el = document.createElement('textarea')
        el.value = text
        el.style.cssText = 'position:fixed;top:-9999px;left:-9999px;opacity:0'
        document.body.appendChild(el)
        el.select()
        const ok = document.execCommand('copy')
        document.body.removeChild(el)
        return ok
    } catch {
        return false
    }
}
