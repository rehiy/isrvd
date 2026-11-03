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
