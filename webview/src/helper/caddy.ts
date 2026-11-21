import type { CaddyCert, CaddyRoute, CaddyRouteUpsert } from '@/service/types'

const isObject = (value: unknown): value is Record<string, unknown> =>
    value !== null && typeof value === 'object' && !Array.isArray(value)

const hasOwn = (value: object, key: string): boolean => Object.prototype.hasOwnProperty.call(value, key)
const isStringArray = (value: unknown): value is string[] => Array.isArray(value) && value.every(item => typeof item === 'string')
const hasOnlyKeys = (value: unknown, keys: string[]): value is Record<string, unknown> =>
    isObject(value) && Object.keys(value).every(key => keys.includes(key))
const hasStringFields = (value: Record<string, unknown>, keys: string[]): boolean =>
    keys.every(key => !hasOwn(value, key) || typeof value[key] === 'string')
const isHeaderDictionary = (value: unknown): boolean => isObject(value) && Object.values(value).every(isStringArray)

// 列表附加字段仅用于定位；原生扩展字段必须原样保留。
export const caddyNativeRoute = (route: CaddyRoute): CaddyRouteUpsert =>
    Object.fromEntries(Object.entries(route).filter(([key]) => !['index', 'server', 'serverID', 'serverName'].includes(key)))

// match/handle 的 null 与缺失值均符合后端原生切片契约；不强制路由拥有 handler。
export const caddyRouteJSONError = (value: unknown): string | undefined => {
    if (!isObject(value)) {
        return '路由 JSON 必须是对象'
    }
    if (hasOwn(value, 'match') && value.match !== null &&
        (!Array.isArray(value.match) || !value.match.every(isObject))) {
        return 'match 字段必须是对象数组'
    }
    if (hasOwn(value, 'handle') && value.handle !== null &&
        (!Array.isArray(value.handle) || !value.handle.every(handler =>
            isObject(handler) && typeof handler.handler === 'string' && handler.handler.trim().length > 0))) {
        return 'handle 字段必须是包含处理器名称的对象数组'
    }
    return undefined
}

const isFormRewrite = (value: unknown, proxy: boolean): boolean => {
    const fields = ['uri', 'strip_path_prefix', 'strip_path_suffix', ...(proxy ? ['method'] : [])]
    if (!hasOnlyKeys(value, [...fields, 'uri_substring']) || !hasStringFields(value, fields)) {
        return false
    }
    return !hasOwn(value, 'uri_substring') || (Array.isArray(value.uri_substring) && value.uri_substring.length <= 1 &&
        value.uri_substring.every(item => hasOnlyKeys(item, ['find', 'replace']) && hasStringFields(item, ['find', 'replace'])))
}

const isFormHeaderOps = (value: unknown): boolean => {
    if (!hasOnlyKeys(value, ['set', 'add', 'delete'])) {
        return false
    }
    return (!hasOwn(value, 'set') || (isHeaderDictionary(value.set) &&
        Object.values(value.set as Record<string, string[]>).every(values => values.length === 1))) &&
        (!hasOwn(value, 'add') || isHeaderDictionary(value.add)) &&
        (!hasOwn(value, 'delete') || isStringArray(value.delete))
}

const isFormHandler = (value: unknown): boolean => {
    if (!isObject(value)) {
        return false
    }
    switch (value.handler) {
        case 'reverse_proxy': {
            if (!hasOnlyKeys(value, ['handler', 'upstreams', 'transport', 'rewrite']) ||
                !Array.isArray(value.upstreams) || !value.upstreams.length ||
                !value.upstreams.every(upstream => hasOnlyKeys(upstream, ['dial']) && typeof upstream.dial === 'string')) {
                return false
            }
            if (hasOwn(value, 'transport')) {
                const transport = value.transport
                if (!isObject(transport)) {
                    return false
                }
                const fields = transport.protocol === 'fastcgi' ? ['protocol', 'root'] :
                    ['protocol', 'dial_timeout', 'response_header_timeout', 'write_timeout']
                if (!hasOnlyKeys(transport, fields) || !hasStringFields(transport, fields) ||
                    !['http', 'fastcgi'].includes(String(transport.protocol))) {
                    return false
                }
            }
            return !hasOwn(value, 'rewrite') || isFormRewrite(value.rewrite, true)
        }
        case 'file_server': {
            return hasOnlyKeys(value, ['handler', 'root', 'browse']) && hasStringFields(value, ['root']) &&
                (!hasOwn(value, 'browse') || hasOnlyKeys(value.browse, []))
        }
        case 'static_response': {
            return hasOnlyKeys(value, ['handler', 'status_code', 'body']) && hasStringFields(value, ['body']) &&
                (!hasOwn(value, 'status_code') || typeof value.status_code === 'number' || typeof value.status_code === 'string')
        }
        case 'rewrite': {
            return isFormRewrite(Object.fromEntries(Object.entries(value).filter(([key]) => key !== 'handler')), false)
        }
        default: {
            return false
        }
    }
}

// 必须先通过形状检查再填表，避免扩展配置触发表单字符串/数组操作异常。
export const caddyRouteSupportsForm = (value: unknown): value is CaddyRouteUpsert => {
    if (!hasOnlyKeys(value, ['match', 'handle'])) {
        return false
    }
    if (hasOwn(value, 'match')) {
        if (!Array.isArray(value.match) || value.match.length > 1 || !value.match.every(match =>
            hasOnlyKeys(match, ['host', 'path', 'method', 'protocol', 'header']) &&
            ['host', 'path', 'method'].every(key => !hasOwn(match, key) || isStringArray(match[key])) &&
            hasStringFields(match, ['protocol']) && (!hasOwn(match, 'header') || isHeaderDictionary(match.header)))) {
            return false
        }
    }
    if (!Array.isArray(value.handle) || !value.handle.length || value.handle.length > 2) {
        return false
    }
    if (value.handle.length === 2) {
        const headers = value.handle[0]
        if (!hasOnlyKeys(headers, ['handler', 'request', 'response']) || headers.handler !== 'headers' ||
            !['request', 'response'].every(key => !hasOwn(headers, key) || isFormHeaderOps(headers[key]))) {
            return false
        }
    }
    return isFormHandler(value.handle[value.handle.length - 1])
}

// 忽略对象键顺序，但保留所有字段、值类型、空值和数组顺序，拒绝有损表单往返。
export const caddyRouteFormMatches = (original: unknown, generated: unknown): boolean => {
    if (original === generated) {
        return true
    }
    if (Array.isArray(original)) {
        return Array.isArray(generated) && original.length === generated.length &&
            original.every((item, index) => caddyRouteFormMatches(item, generated[index]))
    }
    if (!isObject(original) || !isObject(generated)) {
        return false
    }
    const keys = Object.keys(original)
    return keys.length === Object.keys(generated).length && keys.every(key =>
        hasOwn(generated, key) && caddyRouteFormMatches(original[key], generated[key]))
}

// 只分析单一 Host 匹配和单一响应处理器；保留复杂路由的人工顺序。
const simpleHosts = (route: CaddyRouteUpsert): string[] | undefined => {
    if (route.group || Object.keys(route).some(key => !['index', '@id', 'match', 'handle', 'terminal', 'group'].includes(key))) {
        return undefined
    }
    if (route.handle?.length !== 1) {
        return undefined
    }
    const handler = route.handle[0]
    if (!['reverse_proxy', 'static_response', 'file_server'].includes(handler.handler) ||
        handler.handle_response || handler.rewrite || handler.pass_thru) {
        return undefined
    }
    if (handler.handler === 'static_response' && !(Number(handler.status_code ?? 200) >= 200)) {
        return undefined
    }
    if (!route.match?.length) {
        return []
    }
    if (route.match.length !== 1 || Object.keys(route.match[0]).some(key => key !== 'host')) {
        return undefined
    }
    const hosts = route.match[0].host ?? []
    if (!Array.isArray(hosts) || hosts.some(host => typeof host !== 'string' ||
        !host.split('.').every(label => label === '*' || /^[a-z0-9_-]+$/i.test(label)))) {
        return undefined
    }
    return hosts.map(host => host.toLowerCase())
}

// Caddy 的 * 只匹配一个完整域名标签，不跨越点号。
const hostCovers = (pattern: string, host: string): boolean => {
    const labels = pattern.split('.')
    const target = host.split('.')
    return labels.length === target.length && labels.every((label, index) => label === '*' || label === target[index])
}

export const caddyRouteInsertionIndex = (route: CaddyRouteUpsert, routes: CaddyRouteUpsert[]): number | undefined => {
    const hosts = simpleHosts(route)
    if (!hosts?.length || hosts.some(host => host.includes('*'))) {
        return undefined
    }
    const index = routes.findIndex(existing => {
        const patterns = simpleHosts(existing)
        return patterns !== undefined && (patterns.length === 0 ||
            hosts.some(host => patterns.some(pattern => hostCovers(pattern, host))))
    })
    // 从末尾前移不得越过鉴权、重写、分组或未知处理器，避免改变其作用范围。
    if (index < 0 || routes.slice(index).some(existing => simpleHosts(existing) === undefined)) {
        return undefined
    }
    const patterns = simpleHosts(routes[index]) || []
    if (patterns.length && !hosts.every(host => patterns.some(pattern => pattern.includes('*') && hostCovers(pattern, host)))) {
        return undefined
    }
    return index
}

export function caddyCertSummary(cert: CaddyCert, t: (key: string) => string): string {
    if (cert.subject) {
        return cert.subject
    }
    if (cert.source === 'file') {
        return cert.certificate || '-'
    }
    return cert.certificate?.split('\n')[0].trim() || t('(空)')
}

export function caddyCertExpireLabel(notAfter: string | undefined, t: (key: string, ...args: unknown[]) => string): string {
    if (!notAfter) {
        return '-'
    }
    const days = Math.floor((new Date(notAfter).getTime() - Date.now()) / 86400000)
    if (days < 0) {
        return t('已过期 {0} 天', -days)
    }
    if (days === 0) {
        return t('今日过期')
    }
    return t('{0} 天后到期', days)
}
