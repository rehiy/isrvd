import type { CaddyHandler, CaddyHandlerFileServer, CaddyHandlerHeaders, CaddyHandlerReverseProxy, CaddyHandlerRewrite, CaddyHandlerStaticResponse, CaddyMatchSet, CaddyRoute, CaddyRouteUpsert } from '@/service/types'

// 前端表单内部使用的 handler 类型标识（不提交到后端）
export type CaddyHandlerKind = 'reverse_proxy' | 'file_server' | 'static_response' | 'rewrite' | 'raw'

// 单条请求头/响应头操作（前端表单内部使用）
interface CaddyHeaderOp {
    op: 'set' | 'add' | 'delete'
    field: string
    value: string
}

export interface MatchHeader {
    key: string
    value: string
}

export const defaultFormData = () => ({
    kind: 'reverse_proxy' as CaddyHandlerKind,
    hosts: '',
    paths: '',
    methods: '',
    protocol: '',
    upstreams: '',
    upstreamHost: '',
    upstreamPort: '',
    fastcgi: false,
    fastcgiRoot: '',
    dialTimeout: '',
    readTimeout: '',
    writeTimeout: '',
    proxyRewriteEnabled: false,
    proxyRewriteMethod: '',
    proxyRewriteUri: '',
    proxyStripPathPrefix: '',
    proxyStripPathSuffix: '',
    proxyUriSubstringFind: '',
    proxyUriSubstringReplace: '',
    root: '',
    browse: false,
    statusCode: 200,
    body: '',
    rewriteUri: '',
    stripPathPrefix: '',
    stripPathSuffix: '',
    uriSubstringFind: '',
    uriSubstringReplace: '',
    requestHeaders: [] as CaddyHeaderOp[],
    responseHeaders: [] as CaddyHeaderOp[],
    enableHeaders: false
})

type RouteForm = ReturnType<typeof defaultFormData>

export function buildStructuredRoute(form: RouteForm, match: CaddyRouteUpsert['match']): CaddyRouteUpsert | null {
    const handle = buildHandlers(form)
    if (!handle) {
        return null
    }
    return match ? { match, handle } : { handle }
}

export function readStructuredRoute(nativeRoute: CaddyRouteUpsert) {
    if (!caddyRouteSupportsForm(nativeRoute)) {
        return null
    }
    const form = defaultFormData()
    let matchHeaders: MatchHeader[] = []
    // 仅完整建模的单一 MatchSet 可进入表单模式。
    const matchSet = nativeRoute.match?.[0]
    if (matchSet) {
        form.hosts = (matchSet.host || []).join('\n')
        form.paths = (matchSet.path || []).join('\n')
        form.methods = (matchSet.method || []).join(' ')
        form.protocol = matchSet.protocol || ''
        matchHeaders = headersToList(matchSet.header)
    }
    const handles = nativeRoute.handle || []
    const headersHandler = handles.length === 2 ? handles[0] as CaddyHandlerHeaders : null
    const h = handles[handles.length - 1]
    switch (h.handler) {
        case 'reverse_proxy': {
            form.kind = 'reverse_proxy'
            const rph = h as CaddyHandlerReverseProxy
            form.upstreams = (rph.upstreams || []).map(u => u.dial || '').filter(Boolean).join('\n')
            const transport = rph.transport
            if (transport?.protocol === 'fastcgi') {
                form.fastcgi = true
                form.fastcgiRoot = transport.root || ''
            } else if (transport) {
                form.dialTimeout = transport.dial_timeout || ''
                form.readTimeout = transport.response_header_timeout || ''
                form.writeTimeout = transport.write_timeout || ''
            }
            const rw = rph.rewrite
            if (rw) {
                form.proxyRewriteEnabled = true
                form.proxyRewriteMethod = rw.method || ''
                form.proxyRewriteUri = rw.uri || ''
                form.proxyStripPathPrefix = rw.strip_path_prefix || ''
                form.proxyStripPathSuffix = rw.strip_path_suffix || ''
                if (rw.uri_substring?.[0]) {
                    form.proxyUriSubstringFind = rw.uri_substring[0].find || ''
                    form.proxyUriSubstringReplace = rw.uri_substring[0].replace || ''
                }
            }
            break
        }
        case 'file_server': {
            form.kind = 'file_server'
            const fsh = h as CaddyHandlerFileServer
            form.root = fsh.root || ''
            form.browse = 'browse' in fsh
            break
        }
        case 'static_response': {
            form.kind = 'static_response'
            const srh = h as CaddyHandlerStaticResponse
            const sc = srh.status_code
            form.statusCode = typeof sc === 'number' ? sc : (typeof sc === 'string' ? parseInt(sc) : 200)
            form.body = srh.body || ''
            break
        }
        case 'rewrite': {
            form.kind = 'rewrite'
            const rwh = h as CaddyHandlerRewrite
            form.rewriteUri = rwh.uri || ''
            form.stripPathPrefix = rwh.strip_path_prefix || ''
            form.stripPathSuffix = rwh.strip_path_suffix || ''
            if (rwh.uri_substring?.[0]) {
                form.uriSubstringFind = rwh.uri_substring[0].find || ''
                form.uriSubstringReplace = rwh.uri_substring[0].replace || ''
            }
            break
        }
    }
    if (headersHandler) {
        if (headersHandler.request) {
            form.requestHeaders = parseHeadersOps(headersHandler.request)
        }
        if (headersHandler.response) {
            form.responseHeaders = parseHeadersOps(headersHandler.response)
        }
        form.enableHeaders = true
    }
    const generated = buildStructuredRoute(form, buildRouteMatch(form, matchHeaders))
    return caddyRouteFormMatches(nativeRoute, generated) ? { form, matchHeaders } : null
}

export function buildRouteMatch(form: RouteForm, matchHeaders: MatchHeader[]): CaddyRouteUpsert['match'] {
    const hosts = form.hosts.split('\n').map(t => t.trim()).filter(Boolean)
    const paths = form.paths.split('\n').map(t => t.trim()).filter(Boolean)
    const methods = form.methods.split(/\s+/).map(t => t.trim().toUpperCase()).filter(Boolean)
    const headers = listToHeaders(matchHeaders)
    const protocol = form.protocol || undefined
    const matchSet: CaddyMatchSet = {}
    if (hosts.length) {
        matchSet.host = hosts
    }
    if (paths.length) {
        matchSet.path = paths
    }
    if (methods.length) {
        matchSet.method = methods
    }
    if (headers) {
        matchSet.header = headers
    }
    if (protocol) {
        matchSet.protocol = protocol
    }
    return Object.keys(matchSet).length ? [matchSet] : undefined
}

function buildHandlers(f: RouteForm): CaddyHandler[] | null {
    let terminalHandler: CaddyHandler | null = null
    switch (f.kind) {
        case 'reverse_proxy': {
            const upstreams = f.upstreams.split('\n').map(t => t.trim()).filter(Boolean)
            if (upstreams.length) {
                terminalHandler = {
                    handler: 'reverse_proxy',
                    upstreams: upstreams.map(d => ({ dial: d }))
                }
                if (f.fastcgi) {
                    const transport: Record<string, unknown> = { protocol: 'fastcgi' }
                    if (f.fastcgiRoot.trim()) {
                        transport.root = f.fastcgiRoot.trim()
                    }
                    terminalHandler.transport = transport
                } else if (f.dialTimeout.trim() || f.readTimeout.trim() || f.writeTimeout.trim()) {
                    const transport: Record<string, unknown> = { protocol: 'http' }
                    if (f.dialTimeout.trim()) {
                        transport.dial_timeout = f.dialTimeout.trim()
                    }
                    if (f.readTimeout.trim()) {
                        transport.response_header_timeout = f.readTimeout.trim()
                    }
                    if (f.writeTimeout.trim()) {
                        transport.write_timeout = f.writeTimeout.trim()
                    }
                    terminalHandler.transport = transport
                }
                const rewrite = buildProxyRewrite(f)
                if (rewrite) {
                    terminalHandler.rewrite = rewrite
                }
            }
            break
        }
        case 'file_server': {
            terminalHandler = { handler: 'file_server' }
            if (f.root.trim()) {
                terminalHandler.root = f.root.trim()
            }
            if (f.browse) {
                terminalHandler.browse = {}
            }
            break
        }
        case 'static_response': {
            terminalHandler = { handler: 'static_response' }
            if (f.statusCode) {
                terminalHandler.status_code = f.statusCode
            }
            if (f.body) {
                terminalHandler.body = f.body
            }
            break
        }
        case 'rewrite': {
            terminalHandler = { handler: 'rewrite' }
            if (f.rewriteUri.trim()) {
                terminalHandler.uri = f.rewriteUri.trim()
            }
            if (f.stripPathPrefix.trim()) {
                terminalHandler.strip_path_prefix = f.stripPathPrefix.trim()
            }
            if (f.stripPathSuffix.trim()) {
                terminalHandler.strip_path_suffix = f.stripPathSuffix.trim()
            }
            if (f.uriSubstringFind.trim()) {
                terminalHandler.uri_substring = [{ find: f.uriSubstringFind.trim(), replace: f.uriSubstringReplace.trim() }]
            }
            break
        }
    }
    if (!terminalHandler) {
        return null
    }
    const buildOps = (ops: CaddyHeaderOp[]) => {
        const set: Record<string, string[]> = Object.create(null)
        const add: Record<string, string[]> = Object.create(null)
        const del: string[] = []
        ops.forEach(op => {
            if (op.op === 'set') {
                set[op.field] = [op.value]
            } else if (op.op === 'add') {
                if (!Object.prototype.hasOwnProperty.call(add, op.field)) {
                    add[op.field] = []
                }
                add[op.field].push(op.value)
            } else if (op.op === 'delete') {
                del.push(op.field)
            }
        })
        const out: Record<string, unknown> = {}
        if (Object.keys(set).length) {
            out.set = set
        }
        if (Object.keys(add).length) {
            out.add = add
        }
        if (del.length) {
            out.delete = del
        }
        return out
    }
    const handles: CaddyHandler[] = []
    // 只有当启用header操作时才添加headers中间件
    if (f.enableHeaders) {
        const reqOps = f.requestHeaders.filter(op => op.field.trim())
        const respOps = f.responseHeaders.filter(op => op.field.trim())
        if (reqOps.length || respOps.length) {
            const hh: CaddyHandler = { handler: 'headers' }
            if (reqOps.length) {
                hh.request = buildOps(reqOps)
            }
            if (respOps.length) {
                hh.response = buildOps(respOps)
            }
            handles.push(hh)
        }
    }
    handles.push(terminalHandler)
    return handles
}

function buildProxyRewrite(f: RouteForm): Record<string, unknown> | null {
    if (!f.proxyRewriteEnabled) {
        return null
    }
    const rewrite: Record<string, unknown> = {}
    if (f.proxyRewriteMethod.trim()) {
        rewrite.method = f.proxyRewriteMethod.trim().toUpperCase()
    }
    if (f.proxyRewriteUri.trim()) {
        rewrite.uri = f.proxyRewriteUri.trim()
    }
    if (f.proxyStripPathPrefix.trim()) {
        rewrite.strip_path_prefix = f.proxyStripPathPrefix.trim()
    }
    if (f.proxyStripPathSuffix.trim()) {
        rewrite.strip_path_suffix = f.proxyStripPathSuffix.trim()
    }
    if (f.proxyUriSubstringFind.trim()) {
        rewrite.uri_substring = [{ find: f.proxyUriSubstringFind.trim(), replace: f.proxyUriSubstringReplace.trim() }]
    }
    return Object.keys(rewrite).length ? rewrite : null
}

function headersToList(headers: Record<string, string[]> | undefined): MatchHeader[] {
    const rows: MatchHeader[] = []
    for (const [k, vals] of Object.entries(headers || {})) {
        if (vals.length === 0) {
            rows.push({ key: k, value: '' })
        } else {
            for (const v of vals) {
                rows.push({ key: k, value: v })
            }
        }
    }
    return rows
}

function listToHeaders(list: MatchHeader[]): Record<string, string[]> | undefined {
    const headers: Record<string, string[]> = Object.create(null)
    for (const row of list) {
        const k = row.key.trim()
        if (!k) {
            continue
        }
        if (!Object.prototype.hasOwnProperty.call(headers, k)) {
            headers[k] = []
        }
        if (row.value.trim()) {
            headers[k].push(row.value.trim())
        }
    }
    return Object.keys(headers).length ? headers : undefined
}

// parseHeadersOps 从 Caddy headers handler 的 set/add/delete 结构解析为 CaddyHeaderOp 列表
function parseHeadersOps(m: Record<string, unknown>): CaddyHeaderOp[] {
    const ops: CaddyHeaderOp[] = []
    const setMap = m.set as Record<string, string[]> | undefined
    const addMap = m.add as Record<string, string[]> | undefined
    const delArr = m.delete as string[] | undefined
    if (setMap) {
        for (const [k, v] of Object.entries(setMap)) {
            ops.push({ op: 'set', field: k, value: Array.isArray(v) ? v[0] || '' : String(v) })
        }
    }
    if (addMap) {
        for (const [k, v] of Object.entries(addMap)) {
            for (const val of (Array.isArray(v) ? v : [v])) {
                ops.push({ op: 'add', field: k, value: String(val) })
            }
        }
    }
    if (delArr) {
        for (const f of delArr) {
            ops.push({ op: 'delete', field: f, value: '' })
        }
    }
    return ops
}

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
const caddyRouteSupportsForm = (value: unknown): value is CaddyRouteUpsert => {
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
const caddyRouteFormMatches = (original: unknown, generated: unknown): boolean => {
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
