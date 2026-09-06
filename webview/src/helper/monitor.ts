interface MonitorWindowHistory {
    ts: number[]
    labels: string[]
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
    if (rangeSeconds <= 0 || history.ts.length === 0) return

    const cutoff = latestTs - rangeSeconds
    let trimCount = 0
    while (trimCount < history.ts.length && history.ts[trimCount] < cutoff) {
        trimCount++
    }
    if (trimCount === 0) return

    history.ts.splice(0, trimCount)
    history.labels.splice(0, trimCount)
    trimValues(trimCount)
}
