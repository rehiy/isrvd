import type { DockerContainerStats, MonitorContainerRecord } from '@/service/types'

type ContainerChartValues = Pick<DockerContainerStats, 'cpuPercent' | 'memoryPercent' | 'networkRx' | 'networkTx' | 'blockRead' | 'blockWrite'>

export function containerHistoryValues(record: MonitorContainerRecord): ContainerChartValues {
    const data = record.data
    if ('cpuPercent' in data) {
        return data
    }
    const cpu = data.cpu_stats
    const previous = data.precpu_stats
    const cpuDelta = (cpu?.cpu_usage?.total_usage || 0) - (previous?.cpu_usage?.total_usage || 0)
    const systemDelta = (cpu?.system_cpu_usage || 0) - (previous?.system_cpu_usage || 0)
    const cores = cpu?.online_cpus || cpu?.cpu_usage?.percpu_usage?.length || 1
    const memory = data.memory_stats
    const cache = memory?.stats?.total_inactive_file ?? memory?.stats?.inactive_file ?? 0
    const used = Math.max(0, (memory?.usage || 0) - cache)
    const values = {
        cpuPercent: systemDelta > 0 && cpuDelta > 0 ? cpuDelta / systemDelta * cores * 100 : 0,
        memoryPercent: memory?.limit ? used / memory.limit * 100 : 0,
        networkRx: 0, networkTx: 0, blockRead: 0, blockWrite: 0,
    }
    for (const network of Object.values(data.networks || {})) {
        values.networkRx += network.rx_bytes || 0
        values.networkTx += network.tx_bytes || 0
    }
    for (const block of data.blkio_stats?.io_service_bytes_recursive || []) {
        if (block.op.toLowerCase() === 'read') {
            values.blockRead += block.value
        } else if (block.op.toLowerCase() === 'write') {
            values.blockWrite += block.value
        }
    }
    return values
}

interface MonitorWindowHistory {
    ts: number[]
    labels: string[]
}

export const formatMonitorBytes = (bytes: number, rate = false): string => {
    if (!bytes || bytes < 0) {
        return rate ? '0 B/s' : '0 B'
    }
    const units = rate ? ['B/s', 'KB/s', 'MB/s', 'GB/s'] : ['B', 'KB', 'MB', 'GB', 'TB']
    let unitIndex = 0
    let value = bytes
    while (value >= 1024 && unitIndex < units.length - 1) {
        value /= 1024
        unitIndex++
    }
    return `${value.toFixed(1)} ${units[unitIndex]}`
}

export function usagePercent(used: number, total: number): number {
    if (!total) {
        return 0
    }
    return parseFloat(((used / total) * 100).toFixed(1))
}

export function semanticColor(pct: number): string {
    if (pct >= 90) {
        return 'text-red-500'
    }
    if (pct >= 70) {
        return 'text-amber-500'
    }
    return 'text-emerald-500'
}

export function monitorTimeLabel(ts: number): string {
    const t = new Date(ts * 1000)
    return `${t.getHours().toString().padStart(2, '0')}:${t.getMinutes().toString().padStart(2, '0')}:${t.getSeconds().toString().padStart(2, '0')}`
}

export function appendMonitorPoint(
    history: MonitorWindowHistory,
    ts: number,
    rangeSeconds: number,
    appendValues: () => void,
    trimValues: (count: number) => void
): void {
    history.ts.push(ts)
    history.labels.push(monitorTimeLabel(ts))
    appendValues()
    trimMonitorHistory(history, ts, rangeSeconds, trimValues)
}

function trimMonitorHistory(
    history: MonitorWindowHistory,
    latestTs: number,
    rangeSeconds: number,
    trimValues: (count: number) => void
): void {
    if (rangeSeconds <= 0 || history.ts.length === 0) {
        return
    }

    const cutoff = latestTs - rangeSeconds
    let trimCount = 0
    while (trimCount < history.ts.length && history.ts[trimCount] < cutoff) {
        trimCount++
    }
    if (trimCount === 0) {
        return
    }

    history.ts.splice(0, trimCount)
    history.labels.splice(0, trimCount)
    trimValues(trimCount)
}
