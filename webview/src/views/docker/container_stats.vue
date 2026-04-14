<script setup>
import { Chart, registerables } from 'chart.js'
import { nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { onBeforeRouteLeave } from 'vue-router'

import api from '@/service/api.js'
import { formatFileSize } from '@/helper/utils.js'

Chart.register(...registerables)

const props = defineProps({
  containerId: { type: String, required: true },
  container: { type: Object, required: true }
})

// 统计状态
const statsData = ref(null)
const statsLoading = ref(true)
let statsTimer = null

// 历史数据
const MAX_POINTS = 60
const labels = []
const cpuData = []
const memData = []
const netRxData = []
const netTxData = []
const blkRData = []
const blkWData = []

// 上一次累计值
let prevNetRx = 0
let prevNetTx = 0
let prevBlkR = 0
let prevBlkW = 0
let prevTime = 0

// 实时速率
const netRxRate = ref(0)
const netTxRate = ref(0)
const blkRRate = ref(0)
const blkWRate = ref(0)

// Chart 实例
let cpuChart = null
let memChart = null
let netChart = null
let blkChart = null

// Canvas refs
const cpuRef = ref(null)
const memRef = ref(null)
const netRef = ref(null)
const blkRef = ref(null)

// ========== 统计定时器 ==========

const startStatsTimer = () => {
  stopStatsTimer()
  statsTimer = setInterval(() => loadStats(), 3000)
}

const stopStatsTimer = () => {
  if (statsTimer) {
    clearInterval(statsTimer)
    statsTimer = null
  }
}

const handleVisibilityChange = () => {
  if (document.hidden) {
    stopStatsTimer()
  } else if (props.container?.state === 'running') {
    startStatsTimer()
  }
}

// ========== 统计数据加载 ==========

const loadStats = async () => {
  try {
    const res = await api.containerStats(props.containerId)
    if (!res.payload) return
    statsData.value = res.payload
    pushPoint(res.payload)
    renderCharts()
  } catch (e) {
    // 静默失败
  }
}

// 追加一个数据点
const pushPoint = (data) => {
  const now = new Date()
  const label = now.getHours().toString().padStart(2, '0') + ':' +
    now.getMinutes().toString().padStart(2, '0') + ':' +
    now.getSeconds().toString().padStart(2, '0')

  const now_ms = Date.now()
  const elapsed = prevTime > 0 ? (now_ms - prevTime) / 1000 : 0

  labels.push(label)
  cpuData.push(+(data.cpuPercent || 0).toFixed(2))
  memData.push(+(data.memoryPercent || 0).toFixed(2))

  if (elapsed > 0) {
    const rxRate = +Math.max(0, ((data.networkRx || 0) - prevNetRx) / elapsed).toFixed(0)
    const txRate = +Math.max(0, ((data.networkTx || 0) - prevNetTx) / elapsed).toFixed(0)
    const brRate = +Math.max(0, ((data.blockRead || 0) - prevBlkR) / elapsed).toFixed(0)
    const bwRate = +Math.max(0, ((data.blockWrite || 0) - prevBlkW) / elapsed).toFixed(0)
    netRxData.push(rxRate)
    netTxData.push(txRate)
    blkRData.push(brRate)
    blkWData.push(bwRate)
    netRxRate.value = rxRate
    netTxRate.value = txRate
    blkRRate.value = brRate
    blkWRate.value = bwRate
  } else {
    netRxData.push(0)
    netTxData.push(0)
    blkRData.push(0)
    blkWData.push(0)
  }

  prevNetRx = data.networkRx || 0
  prevNetTx = data.networkTx || 0
  prevBlkR = data.blockRead || 0
  prevBlkW = data.blockWrite || 0
  prevTime = now_ms

  if (labels.length > MAX_POINTS) {
    labels.shift()
    cpuData.shift()
    memData.shift()
    netRxData.shift()
    netTxData.shift()
    blkRData.shift()
    blkWData.shift()
  }
}

// ========== 折线图 ==========

const baseOptions = (yOptions = {}, tooltipCb = null) => ({
  responsive: true,
  maintainAspectRatio: false,
  animation: false,
  interaction: { intersect: false, mode: 'index' },
  plugins: {
    legend: { display: false },
    tooltip: {
      backgroundColor: 'rgba(15,23,42,0.9)',
      titleFont: { size: 10 },
      bodyFont: { size: 10 },
      padding: 8,
      cornerRadius: 6,
      callbacks: tooltipCb ? { label: tooltipCb } : {}
    }
  },
  scales: {
    x: { display: false },
    y: {
      display: true,
      beginAtZero: true,
      grid: { color: 'rgba(148,163,184,0.08)' },
      border: { display: false },
      ticks: { font: { size: 9 }, color: '#94a3b8', maxTicksLimit: 4, padding: 4 },
      ...yOptions
    }
  },
  elements: {
    point: { radius: 0, hoverRadius: 3 },
    line: { tension: 0.4, borderWidth: 1.5 }
  }
})

const makeDataset = (data, color, label = '') => ({
  label,
  data: [...data],
  borderColor: color,
  backgroundColor: color.replace(')', ', 0.08)').replace('rgb', 'rgba'),
  fill: true
})

const initCharts = () => {
  destroyCharts()

  if (cpuRef.value) {
    cpuChart = new Chart(cpuRef.value, {
      type: 'line',
      data: { labels: [...labels], datasets: [makeDataset(cpuData, '#3b82f6')] },
      options: baseOptions(
        { max: 100, ticks: { font: { size: 9 }, color: '#94a3b8', maxTicksLimit: 4, padding: 4, callback: v => v + '%' } },
        ctx => ctx.parsed.y.toFixed(1) + '%'
      )
    })
  }

  if (memRef.value) {
    memChart = new Chart(memRef.value, {
      type: 'line',
      data: { labels: [...labels], datasets: [makeDataset(memData, '#8b5cf6')] },
      options: baseOptions(
        { max: 100, ticks: { font: { size: 9 }, color: '#94a3b8', maxTicksLimit: 4, padding: 4, callback: v => v + '%' } },
        ctx => ctx.parsed.y.toFixed(1) + '%'
      )
    })
  }

  if (netRef.value) {
    netChart = new Chart(netRef.value, {
      type: 'line',
      data: {
        labels: [...labels],
        datasets: [
          { ...makeDataset(netRxData, '#14b8a6'), label: '接收' },
          { ...makeDataset(netTxData, '#0d9488'), label: '发送' }
        ]
      },
      options: {
        ...baseOptions({}, ctx => ctx.dataset.label + ': ' + formatFileSize(ctx.parsed.y) + '/s'),
        plugins: {
          ...baseOptions().plugins,
          legend: { display: true, position: 'bottom', labels: { boxWidth: 8, padding: 8, font: { size: 10 }, color: '#64748b' } },
          tooltip: {
            backgroundColor: 'rgba(15,23,42,0.9)', titleFont: { size: 10 }, bodyFont: { size: 10 }, padding: 8, cornerRadius: 6,
            callbacks: { label: ctx => ctx.dataset.label + ': ' + formatFileSize(ctx.parsed.y) + '/s' }          }
        }
      }
    })
  }

  if (blkRef.value) {
    blkChart = new Chart(blkRef.value, {
      type: 'line',
      data: {
        labels: [...labels],
        datasets: [
          { ...makeDataset(blkRData, '#f59e0b'), label: '读取' },
          { ...makeDataset(blkWData, '#d97706'), label: '写入' }
        ]
      },
      options: {
        ...baseOptions({}, ctx => ctx.dataset.label + ': ' + formatFileSize(ctx.parsed.y) + '/s'),
        plugins: {
          ...baseOptions().plugins,
          legend: { display: true, position: 'bottom', labels: { boxWidth: 8, padding: 8, font: { size: 10 }, color: '#64748b' } },
          tooltip: {
            backgroundColor: 'rgba(15,23,42,0.9)', titleFont: { size: 10 }, bodyFont: { size: 10 }, padding: 8, cornerRadius: 6,
            callbacks: { label: ctx => ctx.dataset.label + ': ' + formatFileSize(ctx.parsed.y) + '/s' }          }
        }
      }
    })
  }
}

const renderCharts = () => {  if (cpuChart) { cpuChart.data.labels = snap; cpuChart.data.datasets[0].data = [...cpuData]; cpuChart.update('none') }
  if (memChart) { memChart.data.labels = snap; memChart.data.datasets[0].data = [...memData]; memChart.update('none') }
  if (netChart) { netChart.data.labels = snap; netChart.data.datasets[0].data = [...netRxData]; netChart.data.datasets[1].data = [...netTxData]; netChart.update('none') }
  if (blkChart) { blkChart.data.labels = snap; blkChart.data.datasets[0].data = [...blkRData]; blkChart.data.datasets[1].data = [...blkWData]; blkChart.update('none') }
}

const destroyCharts = () => {
  if (cpuChart) { cpuChart.destroy(); cpuChart = null }
  if (memChart) { memChart.destroy(); memChart = null }
  if (netChart) { netChart.destroy(); netChart = null }
  if (blkChart) { blkChart.destroy(); blkChart = null }
}

const clearHistory = () => {
  labels.length = 0; cpuData.length = 0; memData.length = 0
  netRxData.length = 0; netTxData.length = 0; blkRData.length = 0; blkWData.length = 0
  prevNetRx = prevNetTx = prevBlkR = prevBlkW = prevTime = 0
  netRxRate.value = netTxRate.value = blkRRate.value = blkWRate.value = 0
}

watch(statsData, async (val, old) => {
  if (val && !old) {
    await nextTick()
    initCharts()
  }
}, { immediate: false })

onMounted(async () => {
  document.addEventListener('visibilitychange', handleVisibilityChange)
  if (props.container?.state === 'running') {
    await loadStats()
    statsLoading.value = false
    startStatsTimer()
  } else {
    statsLoading.value = false
  }
})

onBeforeRouteLeave(() => {
  stopStatsTimer()
})

onUnmounted(() => {
  stopStatsTimer()
  destroyCharts()
  clearHistory()
  document.removeEventListener('visibilitychange', handleVisibilityChange)
})
</script>

<template>
  <div class="p-6 space-y-4">
    <!-- 加载状态 -->
    <div v-if="statsLoading && !statsData" class="flex flex-col items-center justify-center py-12 gap-3 text-slate-400 text-sm">
      <div class="w-8 h-8 spinner"></div>
      <span>正在采集数据...</span>
    </div>

    <template v-else-if="statsData">
      <!-- 核心指标：CPU 和 内存 -->
      <div class="grid grid-cols-2 gap-4">
        <!-- CPU 使用率 -->
        <div class="bg-slate-50 rounded-2xl p-5 border border-slate-200/60 relative overflow-hidden">
          <div class="absolute top-0 left-0 right-0 h-0.5 bg-gradient-to-r from-blue-400 to-transparent"></div>
          <div class="flex items-center gap-2 mb-1">
            <div class="w-6 h-6 rounded-lg bg-gradient-to-br from-blue-500 to-blue-600 flex items-center justify-center">
              <i class="fas fa-microchip text-white text-[9px]"></i>
            </div>
            <span class="text-sm font-semibold text-slate-700">CPU</span>
            <span class="ml-auto text-lg font-bold font-mono" :class="statsData.cpuPercent > 80 ? 'text-red-500' : statsData.cpuPercent > 60 ? 'text-amber-500' : 'text-blue-600'">
              {{ statsData.cpuPercent }}%
            </span>
          </div>
          <div class="flex items-center gap-3 mb-3 text-[10px] text-slate-400">
            <span v-if="statsData.cpuCores">核心 <span class="text-slate-600 font-medium">{{ statsData.cpuCores }} 核</span></span>
            <span v-if="statsData.cpuFreq">频率 <span class="text-slate-600 font-medium">{{ statsData.cpuFreq.toFixed(0) }} MHz</span></span>
            <span v-if="statsData.cpuThrottled && statsData.cpuThrottled.throttledPeriods > 0" class="text-amber-500">
              <i class="fas fa-bolt"></i> 节流 <span class="font-medium">{{ statsData.cpuThrottled.throttledPeriods }}</span>
            </span>
          </div>
          <div class="h-28"><canvas ref="cpuRef"></canvas></div>
        </div>

        <!-- 内存使用 -->
        <div class="bg-slate-50 rounded-2xl p-5 border border-slate-200/60 relative overflow-hidden">
          <div class="absolute top-0 left-0 right-0 h-0.5 bg-gradient-to-r from-purple-400 to-transparent"></div>
          <div class="flex items-center gap-2 mb-1">
            <div class="w-6 h-6 rounded-lg bg-gradient-to-br from-purple-500 to-purple-600 flex items-center justify-center">
              <i class="fas fa-memory text-white text-[9px]"></i>
            </div>
            <span class="text-sm font-semibold text-slate-700">内存</span>
            <span class="ml-auto text-lg font-bold font-mono" :class="statsData.memoryPercent > 80 ? 'text-red-500' : statsData.memoryPercent > 60 ? 'text-amber-500' : 'text-purple-600'">
              {{ statsData.memoryPercent }}%
            </span>
          </div>
          <div class="flex items-center gap-3 mb-3 text-[10px] text-slate-400">
            <span>内存 <span class="text-slate-600 font-medium">{{ formatFileSize(statsData.memoryUsage) }}</span></span>
            <span>限制 <span class="text-slate-600 font-medium">{{ formatFileSize(statsData.memoryLimit) }}</span></span>
          </div>
          <div class="h-28"><canvas ref="memRef"></canvas></div>
        </div>
      </div>

      <!-- I/O 指标卡片 -->
      <div class="grid grid-cols-2 gap-4">
        <!-- 网络 I/O -->
        <div class="bg-slate-50 rounded-2xl p-5 border border-slate-200/60 relative overflow-hidden">
          <div class="absolute top-0 left-0 right-0 h-0.5 bg-gradient-to-r from-teal-400 to-transparent"></div>
          <div class="flex items-center gap-2 mb-1">
            <div class="w-6 h-6 rounded-lg bg-gradient-to-br from-teal-500 to-teal-600 flex items-center justify-center">
              <i class="fas fa-network-wired text-white text-[9px]"></i>
            </div>
            <span class="text-sm font-semibold text-slate-700">网络</span>
            <span class="ml-auto text-xs font-mono text-teal-600">
              <span class="text-teal-500">↓</span> {{ formatFileSize(netRxRate) }}/s
              <span class="mx-1 text-slate-300">·</span>
              <span class="text-teal-700">↑</span> {{ formatFileSize(netTxRate) }}/s
            </span>
          </div>
          <div class="flex items-center gap-3 mb-3 text-[10px] text-slate-400">
            <span v-if="statsData.networkDetail">网卡 <span class="text-slate-600 font-medium">{{ Object.keys(statsData.networkDetail).length }} 块</span></span>
            <span>累计收 <span class="text-slate-600 font-medium">{{ formatFileSize(statsData.networkRx) }}</span></span>
            <span>累计发 <span class="text-slate-600 font-medium">{{ formatFileSize(statsData.networkTx) }}</span></span>
          </div>
          <div class="h-28"><canvas ref="netRef"></canvas></div>
        </div>

        <!-- 磁盘 I/O -->
        <div class="bg-slate-50 rounded-2xl p-5 border border-slate-200/60 relative overflow-hidden">
          <div class="absolute top-0 left-0 right-0 h-0.5 bg-gradient-to-r from-amber-400 to-transparent"></div>
          <div class="flex items-center gap-2 mb-1">
            <div class="w-6 h-6 rounded-lg bg-gradient-to-br from-amber-500 to-amber-600 flex items-center justify-center">
              <i class="fas fa-hard-drive text-white text-[9px]"></i>
            </div>
            <span class="text-sm font-semibold text-slate-700">磁盘</span>
            <span class="ml-auto text-xs font-mono text-amber-600">
              <span class="text-amber-500">↓</span> {{ formatFileSize(blkRRate) }}/s
              <span class="mx-1 text-slate-300">·</span>
              <span class="text-amber-700">↑</span> {{ formatFileSize(blkWRate) }}/s
            </span>
          </div>
          <div class="flex items-center gap-3 mb-3 text-[10px] text-slate-400">
            <span v-if="statsData.blockDetail">设备 <span class="text-slate-600 font-medium">{{ statsData.blockDetail.length }} 块</span></span>
            <span>累计读 <span class="text-slate-600 font-medium">{{ formatFileSize(statsData.blockRead) }}</span></span>
            <span>累计写 <span class="text-slate-600 font-medium">{{ formatFileSize(statsData.blockWrite) }}</span></span>
          </div>
          <div class="h-28"><canvas ref="blkRef"></canvas></div>
        </div>
      </div>

      <!-- 进程信息 -->
      <div class="bg-slate-50 rounded-2xl p-5 border border-slate-200/60 relative overflow-hidden">
        <div class="absolute top-0 left-0 right-0 h-0.5 bg-gradient-to-r from-rose-400 to-transparent"></div>
        <div class="flex items-center gap-2 mb-4">
          <div class="w-7 h-7 rounded-lg bg-gradient-to-br from-rose-500 to-rose-600 flex items-center justify-center">
            <i class="fas fa-list-ol text-white text-[10px]"></i>
          </div>
          <span class="text-sm font-semibold text-slate-700">进程信息</span>
          <span class="ml-auto text-xs text-slate-400">
            <span class="font-semibold text-slate-700">{{ statsData.pids }}</span> 运行中
            <span v-if="statsData.pidsLimit > 0"> / 限制 {{ statsData.pidsLimit }}</span>
          </span>
          <div v-if="statsData.pidsLimit > 0" class="w-20 h-1.5 bg-slate-200 rounded-full overflow-hidden">
            <div
              class="h-full rounded-full transition-all duration-500"
              :class="statsData.pids / statsData.pidsLimit > 0.9 ? 'bg-red-500' : statsData.pids / statsData.pidsLimit > 0.7 ? 'bg-amber-500' : 'bg-emerald-500'"
              :style="{ width: Math.min(statsData.pids / statsData.pidsLimit * 100, 100) + '%' }"
            ></div>
          </div>
        </div>
        <div v-if="statsData.processList && statsData.processList.processes && statsData.processList.processes.length > 0" class="overflow-x-auto max-h-60 overflow-y-auto">
          <table class="w-full text-xs">
            <thead class="sticky top-0 bg-slate-100">
              <tr>
                <th v-for="title in statsData.processList.titles" :key="title" class="px-3 py-2 text-left text-[10px] font-semibold text-slate-500 uppercase tracking-wider whitespace-nowrap">
                  {{ title }}
                </th>
              </tr>
            </thead>
            <tbody class="bg-white divide-y divide-slate-100">
              <tr v-for="(proc, idx) in statsData.processList.processes" :key="idx" class="hover:bg-slate-50">
                <td v-for="(val, vi) in proc" :key="vi" class="px-3 py-1.5 whitespace-nowrap font-mono text-slate-600">{{ val }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-else class="text-xs text-slate-400">暂无进程信息</div>
      </div>
    </template>
  </div>
</template>
