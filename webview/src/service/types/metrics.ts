// ─── 监控指标 ───

export type MetricsSourceName = 'caddy' | 'apisix' | 'host' | 'containers'

export interface MetricsSourceStatus {
    name: MetricsSourceName
    available: boolean
    enabled: boolean
    enabledUnknown?: boolean
    scrapable: boolean
    url: string
    error?: string
}
