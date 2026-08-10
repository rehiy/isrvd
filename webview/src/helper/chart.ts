import { Chart, registerables } from 'chart.js'

import { hexToRgba } from '@/helper/format'

Chart.register(...registerables)

export function makeLineDataset(data: number[], color: string, label: string) {
    return { label, data: [...data], borderColor: color, backgroundColor: hexToRgba(color, 0.1), fill: true }
}

export default Chart
