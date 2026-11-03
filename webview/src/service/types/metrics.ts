// ─── 监控指标与网关访问统计 ───

export type MetricsGatewayName = 'caddy' | 'apisix'
export type MetricsSourceName = MetricsGatewayName | 'host' | 'containers'

export interface MetricsSourceStatus {
    name: MetricsSourceName
    available: boolean
    enabled: boolean
    enabledUnknown?: boolean
    scrapable: boolean
    url: string
    error?: string
}

export interface MetricsStatsPoint {
    ts: number
    rps: number
    s2xx: number
    s3xx: number
    s4xx: number
    s5xx: number
    avg: number
}

export interface MetricsStatsSummary {
    requests: number
    s2xx: number
    s3xx: number
    s4xx: number
    s5xx: number
    errorRate: number
    rps: number
    avg: number
    bytesOut: number
}

export interface MetricsSiteStat {
    site: string
    requests: number
    s4xx: number
    s5xx: number
    avg: number
    bytesOut: number
}

export interface MetricsGatewayStats {
    name: MetricsGatewayName
    error?: string
    summary: MetricsStatsSummary
    series: MetricsStatsPoint[]
    sites: MetricsSiteStat[]
}
