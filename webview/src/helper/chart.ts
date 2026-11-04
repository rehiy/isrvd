import { Chart, registerables } from 'chart.js'
import type { ChartDataset, ChartOptions, TooltipItem } from 'chart.js'

import { hexToRgba } from '@/helper/format'
import { formatMonitorBytes } from '@/helper/monitor'

Chart.register(...registerables)

type MonitorScales = NonNullable<ChartOptions<'line'>['scales']>
type MonitorScale = Extract<NonNullable<MonitorScales[string]>, { type?: 'linear' }>
type MonitorUnit = 'percent' | 'bytes' | 'rate' | 'count' | 'power'

export function makeMonitorScale(unit: MonitorUnit = 'count', options: MonitorScale = {}): MonitorScale {
    const callback = (value: string | number) => {
        if (unit === 'percent') {
            return Number(value) + '%'
        }
        if (unit === 'bytes' || unit === 'rate') {
            return formatMonitorBytes(Number(value), unit === 'rate')
        }
        return unit === 'power' ? Number(value) + ' W' : Number(value)
    }
    return {
        type: 'linear', display: true, beginAtZero: true,
        ...(unit === 'percent' ? { max: 100 } : {}),
        ...options,
        grid: { color: 'rgba(148,163,184,0.08)', ...options.grid },
        border: { display: false, ...options.border },
        ticks: {
            font: { size: 9 }, color: '#94a3b8', padding: 4,
            ...(unit === 'percent' ? { stepSize: 25 } : { maxTicksLimit: 4 }),
            ...(unit === 'count' ? {} : { callback }), ...options.ticks
        }
    }
}

export function makeMonitorChartOptions(scales: MonitorScales, label: (ctx: TooltipItem<'line'>) => string): ChartOptions<'line'> {
    return {
        responsive: true, maintainAspectRatio: false, animation: false,
        interaction: { intersect: false, mode: 'index' },
        plugins: {
            legend: { display: false },
            tooltip: {
                backgroundColor: 'rgba(15,23,42,0.9)',
                titleFont: { size: 10 }, bodyFont: { size: 10 }, padding: 8, cornerRadius: 6,
                callbacks: { label }
            }
        },
        scales: { x: { display: false }, ...scales },
        elements: { point: { radius: 0, hoverRadius: 3 }, line: { tension: 0.4, borderWidth: 2 } }
    }
}

export function makeLineDataset(data: (number | null)[], color: string, label = '', options: Partial<ChartDataset<'line', (number | null)[]>> & { opacity?: number; fillOpacity?: number } = {}) {
    const { opacity = 1, fillOpacity = 0.1, ...datasetOptions } = options
    return {
        label, data: [...data], borderColor: opacity === 1 ? color : hexToRgba(color, opacity),
        backgroundColor: hexToRgba(color, fillOpacity), fill: true, ...datasetOptions
    }
}

export function findMonitorCanvas(container: HTMLElement | null | undefined, attribute: string, key: string): HTMLCanvasElement | null {
    if (!container?.isConnected) {
        return null
    }
    return Array.from(container?.querySelectorAll<HTMLCanvasElement>('canvas') ?? [])
        .find(canvas => canvas.getAttribute(attribute) === key) ?? null
}

export function syncMonitorCharts(charts: Record<string, Chart<'line'>>, container: HTMLElement | null | undefined, attribute: string, keys: string[], init: (key: string) => void, update: (key: string) => void): void {
    const active = new Set(keys)
    for (const [key, chart] of Object.entries(charts)) {
        if (!active.has(key) || chart.canvas !== findMonitorCanvas(container, attribute, key)) {
            chart.destroy()
            delete charts[key]
        }
    }
    for (const key of active) {
        if (charts[key]) {
            update(key)
        } else {
            init(key)
        }
    }
}

export default Chart
