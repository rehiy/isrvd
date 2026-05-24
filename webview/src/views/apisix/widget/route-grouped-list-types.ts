import type { ApisixRoute } from '@/service/types'

export interface ApisixRouteGroupEntry {
    key: string
    route: ApisixRoute
}

export interface ApisixRouteGroupTarget {
    key: string
    label: string
    labelClass?: string
}

export interface ApisixRouteGroupStat {
    key: string
    label: string
    className: string
}

export interface ApisixRouteGroup {
    key: string
    label: string
    labelClass?: string
    summary: string
    preview: string
    entries: ApisixRouteGroupEntry[]
    stats: ApisixRouteGroupStat[]
}
