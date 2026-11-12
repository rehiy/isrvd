import type { CaddyRouteUpsert } from '@/service/types'

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
