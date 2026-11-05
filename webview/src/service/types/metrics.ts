// ─── 监控指标 ───

export type MetricsGatewayName = 'caddy' | 'apisix'
export type MetricsLocalName = 'host' | 'containers'
export type MetricsSourceName = MetricsGatewayName | MetricsLocalName

export interface MetricsSourceStatus {
    name: MetricsSourceName
    configured: boolean
    available: boolean
    enabled: boolean
    enabledUnknown?: boolean
    scrapable: boolean
    url: string
    error?: string
}
