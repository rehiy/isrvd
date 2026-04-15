<script setup>
import { Chart, registerables } from 'chart.js'
import { nextTick, ref, computed, onUnmounted } from 'vue'
import api from '@/service/api.js'
import { POLL_INTERVAL } from '@/helper/utils.js'

Chart.register(...registerables)

const stat = ref(null)
const loading = ref(false)

// 网络速率历史数据
const NET_POINTS = 45
const MAX_STAT_POINTS = 45
let netHistory = {}
let lastNetSnapshot = {}
let pollTimer = null
let netCharts = {}
const netContainerRef = ref(null)

// 磁盘 IO 历史数据
let diskIOHistory = {}
let lastDiskIOSnapshot = {}
let diskIOCharts = {}
const diskIOContainerRef = ref(null)

// CPU / 内存折线图
const cpuHistory = { labels: [], data: [] }
const memHistory = { labels: [], data: [] }
const cpuCanvasRef = ref(null)
const memCanvasRef = ref(null)
let cpuChart = null
let memChart = null

// ── 计算属性：避免模板中重复调用 ──────────────────────────
const cpuVal = computed(() => stat.value ? cpuPercent(stat.value.system.CpuPercent) : 0)
const memVal = computed(() => stat.value ? memPercent(stat.value.system.MemoryUsed, stat.value.system.MemoryTotal) : 0)

// ── 工具函数 ──────────────────────────────────────────────

// 通用字节格式化（rates=true 时单位加 /s）
const fmtSize = (bytes, rates = false) => {
  if (!bytes || bytes < 0) return rates ? '0 B/s' : '0 B'
  const units = rates
    ? ['B/s', 'KB/s', 'MB/s', 'GB/s']
    : ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0, v = bytes
  while (v >= 1024 && i < units.length - 1) { v /= 1024; i++ }
  return `${v.toFixed(1)} ${units[i]}`
}
const fmtBytes = (b) => fmtSize(b, false)
const fmtRate = (b) => fmtSize(b, true)

// 当前时间标签 HH:MM:SS
const timeLabel = () => {
  const now = new Date()
  return [now.getHours(), now.getMinutes(), now.getSeconds()]
    .map(n => n.toString().padStart(2, '0')).join(':')
}

// 格式化运行时间
const fmtUptime = (seconds) => {
  if (!seconds) return '0s'
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const parts = []
  if (d) parts.push(`${d}d`)
  if (h) parts.push(`${h}h`)
  if (m) parts.push(`${m}m`)
  if (!parts.length) parts.push(`${seconds % 60}s`)
  return parts.join(' ')
}

// CPU 使用率（取各核心平均值）
const cpuPercent = (arr) => {
  if (!arr || !arr.length) return 0
  return (arr.reduce((a, b) => a + b, 0) / arr.length).toFixed(1)
}

// 内存使用率
const memPercent = (used, total) => {
  if (!total) return 0
  return ((used / total) * 100).toFixed(1)
}

// 过滤物理网络接口（排除虚拟接口）
const physicalInterfaces = (list) => {
  if (!list) return []
  const virtualPrefixes = ['lo', 'docker', 'veth', 'br-', 'overlay', 'flannel', 'cni', 'tunl', 'dummy', 'virbr']
  return list.filter(ni => !virtualPrefixes.some(p => ni.Name.startsWith(p)))
}

// 语义颜色（bg- / text- 两用，根据使用率高低返回不同颜色）
const semanticColor = (pct, prefix = 'bg') => {
  const p = parseFloat(pct)
  if (p >= 90) return `${prefix}-red-500`
  if (p >= 70) return `${prefix}-amber-500`
  return `${prefix}-emerald-500`
}
const barColor = (pct) => semanticColor(pct, 'bg')
const textColor = (pct) => semanticColor(pct, 'text')

// GC 时间格式化（Unix 时间戳 → 本地时间字符串）
const fmtGCTime = (ts) => {
  if (!ts) return '从未'
  return new Date(ts * 1000).toLocaleString('zh-CN')
}

// 获取网卡当前速率（取历史数组最后一个点）
const currentRate = (name, dir) => {
  const h = netHistory[name]
  if (!h || !h[dir] || !h[dir].length) return 0
  return h[dir][h[dir].length - 1]
}

// 获取磁盘 IO 当前速率（取历史数组最后一个点）
const currentDiskRate = (name, dir) => {
  const h = diskIOHistory[name]
  if (!h || !h[dir] || !h[dir].length) return 0
  return h[dir][h[dir].length - 1]
}

// 获取设备短名（去掉 /dev/ 前缀）
const devShortName = (device) => device.split('/').pop()

// 根据设备名（如 /dev/sda1 → sda1 或 sda）查找 diskIO 数据
const diskIOByDevice = (device) => {
  if (!stat.value?.diskIO) return null
  // 设备路径可能是 /dev/sda1，取最后一段
  const devName = device.split('/').pop()
  // 先精确匹配，再尝试去掉末尾数字匹配父设备（如 sda1 → sda）
  return stat.value.diskIO.find(d => d.Name === devName)
    || stat.value.diskIO.find(d => devName.startsWith(d.Name))
    || null
}

// ── 图表配置 ──────────────────────────────────────────────

// CPU/内存背景折线图配置（无坐标轴，铺满卡片）
const bgChartOptions = () => ({
  responsive: true,
  maintainAspectRatio: false,
  animation: false,
  interaction: { intersect: false, mode: 'index' },
  plugins: {
    legend: { display: false },
    tooltip: {
      backgroundColor: 'rgba(15,23,42,0.85)',
      titleFont: { size: 10 },
      bodyFont: { size: 10 },
      padding: 6,
      cornerRadius: 6,
      callbacks: { label: ctx => ctx.parsed.y.toFixed(1) + '%' }
    }
  },
  scales: {
    x: { display: false },
    y: {
      display: false,
      beginAtZero: true,
      max: 100,
      grid: { display: false },
      border: { display: false }
    }
  },
  elements: {
    point: { radius: 0, hoverRadius: 3 },
    line: { tension: 0.4, borderWidth: 2 }
  }
})

// 网络接口折线图配置（显示图例和 Y 轴）
const netChartOptions = () => ({
  responsive: true,
  maintainAspectRatio: false,
  animation: false,
  interaction: { intersect: false, mode: 'index' },
  plugins: {
    legend: {
      display: true,
      position: 'bottom',
      labels: { boxWidth: 8, padding: 8, font: { size: 10 }, color: '#64748b' }
    },
    tooltip: {
      backgroundColor: 'rgba(15,23,42,0.9)',
      titleFont: { size: 10 },
      bodyFont: { size: 10 },
      padding: 8,
      cornerRadius: 6,
      callbacks: { label: ctx => ctx.dataset.label + ': ' + fmtRate(ctx.parsed.y) }
    }
  },
  scales: {
    x: { display: false },
    y: {
      display: true,
      beginAtZero: true,
      grid: { color: 'rgba(148,163,184,0.08)' },
      border: { display: false },
      ticks: {
        font: { size: 9 },
        color: '#94a3b8',
        maxTicksLimit: 4,
        padding: 4,
        callback: v => fmtRate(v)
      }
    }
  },
  elements: {
    point: { radius: 0, hoverRadius: 3 },
    line: { tension: 0.4, borderWidth: 1.5 }
  }
})

// hex 颜色转 rgba（用于图表背景色）
const hexToRgba = (hex, alpha) => {
  const r = parseInt(hex.slice(1, 3), 16)
  const g = parseInt(hex.slice(3, 5), 16)
  const b = parseInt(hex.slice(5, 7), 16)
  return `rgba(${r},${g},${b},${alpha})`
}

// 创建网络图表数据集
const makeDataset = (data, color, label) => ({
  label,
  data: [...data],
  borderColor: color,
  backgroundColor: hexToRgba(color, 0.1),
  fill: true
})

// 磁盘 IO 图表配置（与网络图表相同）
const diskChartOptions = () => ({
  responsive: true,
  maintainAspectRatio: false,
  animation: false,
  interaction: { intersect: false, mode: 'index' },
  plugins: {
    legend: {
      display: true,
      position: 'bottom',
      labels: { boxWidth: 8, padding: 8, font: { size: 10 }, color: '#64748b' }
    },
    tooltip: {
      backgroundColor: 'rgba(15,23,42,0.9)',
      titleFont: { size: 10 },
      bodyFont: { size: 10 },
      padding: 8,
      cornerRadius: 6,
      callbacks: { label: ctx => ctx.dataset.label + ': ' + fmtRate(ctx.parsed.y) }
    }
  },
  scales: {
    x: { display: false },
    y: {
      display: true,
      beginAtZero: true,
      grid: { color: 'rgba(148,163,184,0.08)' },
      border: { display: false },
      ticks: {
        font: { size: 9 },
        color: '#94a3b8',
        maxTicksLimit: 4,
        padding: 4,
        callback: v => fmtRate(v)
      }
    }
  },
  elements: {
    point: { radius: 0, hoverRadius: 3 },
    line: { tension: 0.4, borderWidth: 1.5 }
  }
})

// ── CPU/内存图表 ──────────────────────────────────────────

// 通用：创建单个背景折线图
const makeStatChart = (canvas, history, borderColor, bgColor) => {
  if (!canvas) return null
  return new Chart(canvas, {
    type: 'line',
    data: {
      labels: [...history.labels],
      datasets: [{ data: [...history.data], borderColor, backgroundColor: bgColor, fill: true }]
    },
    options: bgChartOptions()
  })
}

const initStatCharts = () => {
  if (cpuChart) { cpuChart.destroy(); cpuChart = null }
  if (memChart) { memChart.destroy(); memChart = null }
  cpuChart = makeStatChart(cpuCanvasRef.value, cpuHistory, 'rgba(59,130,246,0.6)', 'rgba(59,130,246,0.08)')
  memChart = makeStatChart(memCanvasRef.value, memHistory, 'rgba(99,102,241,0.6)', 'rgba(99,102,241,0.08)')
}

// 向历史数组追加一个时间点
const pushStatPoint = (cpuV, memV) => {
  const label = timeLabel()
  cpuHistory.labels.push(label)
  cpuHistory.data.push(+parseFloat(cpuV).toFixed(1))
  memHistory.labels.push(label)
  memHistory.data.push(+parseFloat(memV).toFixed(1))
  if (cpuHistory.labels.length > MAX_STAT_POINTS) {
    cpuHistory.labels.shift()
    cpuHistory.data.shift()
    memHistory.labels.shift()
    memHistory.data.shift()
  }
}

// 将最新历史数据同步到图表
const updateStatCharts = () => {
  for (const [chart, history] of [[cpuChart, cpuHistory], [memChart, memHistory]]) {
    if (!chart) continue
    chart.data.labels = [...history.labels]
    chart.data.datasets[0].data = [...history.data]
    chart.update('none')
  }
}

const destroyStatCharts = () => {
  cpuChart?.destroy()
  cpuChart = null
  memChart?.destroy()
  memChart = null
}

// ── 网络图表 ──────────────────────────────────────────────

const getNetCanvas = (name) =>
  netContainerRef.value?.querySelector(`[data-iface="${name}"]`) ?? null

const initNetChart = (name) => {
  const canvas = getNetCanvas(name)
  if (!canvas) return
  netCharts[name]?.destroy()
  const h = netHistory[name] || { labels: [], recv: [], sent: [] }
  netCharts[name] = new Chart(canvas, {
    type: 'line',
    data: {
      labels: [...h.labels],
      datasets: [
        makeDataset(h.recv, '#10b981', '下行'),
        makeDataset(h.sent, '#3b82f6', '上行')
      ]
    },
    options: netChartOptions()
  })
}

const initAllNetCharts = () => {
  if (!stat.value) return
  physicalInterfaces(stat.value.system.NetInterface).forEach(ni => {
    if (!netHistory[ni.Name]) netHistory[ni.Name] = { labels: [], recv: [], sent: [] }
    initNetChart(ni.Name)
  })
}

const updateNetChart = (name) => {
  const chart = netCharts[name]
  const h = netHistory[name]
  if (!chart || !h) return
  chart.data.labels = [...h.labels]
  chart.data.datasets[0].data = [...h.recv]
  chart.data.datasets[1].data = [...h.sent]
  chart.update('none')
}

const destroyNetCharts = () => {
  Object.values(netCharts).forEach(c => c.destroy())
  netCharts = {}
}

// ── 磁盘 IO 图表 ──────────────────────────────────────────

const getDiskCanvas = (name) =>
  diskIOContainerRef.value?.querySelector(`[data-disk="${name}"]`) ?? null

const initDiskChart = (name) => {
  const canvas = getDiskCanvas(name)
  if (!canvas) return
  diskIOCharts[name]?.destroy()
  const h = diskIOHistory[name] || { labels: [], read: [], write: [] }
  diskIOCharts[name] = new Chart(canvas, {
    type: 'line',
    data: {
      labels: [...h.labels],
      datasets: [
        makeDataset(h.read, '#f59e0b', '读取'),
        makeDataset(h.write, '#8b5cf6', '写入')
      ]
    },
    options: diskChartOptions()
  })
}

const initAllDiskCharts = () => {
  if (!stat.value?.system?.DiskPartition) return
  // 以分区设备名为 key 初始化图表
  stat.value.system.DiskPartition.forEach(dp => {
    const devName = dp.Device.split('/').pop()
    if (!diskIOHistory[devName]) diskIOHistory[devName] = { labels: [], read: [], write: [] }
    initDiskChart(devName)
  })
}

const updateDiskChart = (name) => {
  const chart = diskIOCharts[name]
  const h = diskIOHistory[name]
  if (!chart || !h) return
  chart.data.labels = [...h.labels]
  chart.data.datasets[0].data = [...h.read]
  chart.data.datasets[1].data = [...h.write]
  chart.update('none')
}

const destroyDiskCharts = () => {
  Object.values(diskIOCharts).forEach(c => c.destroy())
  diskIOCharts = {}
}

// 根据前后两次快照计算磁盘 IO 速率，并追加到历史数组
// diskList 中的 Name 是 gopsutil 返回的设备名（如 sda、sda1）
const updateDiskIOHistory = (diskList, intervalSec) => {
  const snapshot = {}
  diskList.forEach(d => {
    snapshot[d.Name] = { read: d.ReadBytes, write: d.WriteBytes }
  })

  if (Object.keys(lastDiskIOSnapshot).length > 0) {
    const label = timeLabel()
    diskList.forEach(d => {
      const prev = lastDiskIOSnapshot[d.Name]
      if (!prev) return
      // 确保 history 存在（可能在 initAllDiskCharts 之后才有新设备）
      if (!diskIOHistory[d.Name]) diskIOHistory[d.Name] = { labels: [], read: [], write: [] }
      const h = diskIOHistory[d.Name]
      h.labels.push(label)
      h.read.push(+Math.max(0, (d.ReadBytes - prev.read) / intervalSec).toFixed(0))
      h.write.push(+Math.max(0, (d.WriteBytes - prev.write) / intervalSec).toFixed(0))
      if (h.labels.length > NET_POINTS) {
        h.labels.shift()
        h.read.shift()
        h.write.shift()
      }
      updateDiskChart(d.Name)
    })
  }

  lastDiskIOSnapshot = snapshot
}

// 根据前后两次快照计算速率，并追加到历史数组
const updateNetHistory = (interfaces, intervalSec) => {
  const snapshot = {}
  interfaces.forEach(ni => {
    snapshot[ni.Name] = { recv: ni.BytesRecv, sent: ni.BytesSent }
  })

  if (Object.keys(lastNetSnapshot).length > 0) {
    const label = timeLabel()
    interfaces.forEach(ni => {
      const prev = lastNetSnapshot[ni.Name]
      if (!prev) return
      if (!netHistory[ni.Name]) netHistory[ni.Name] = { labels: [], recv: [], sent: [] }
      const h = netHistory[ni.Name]
      h.labels.push(label)
      h.recv.push(+Math.max(0, (ni.BytesRecv - prev.recv) / intervalSec).toFixed(0))
      h.sent.push(+Math.max(0, (ni.BytesSent - prev.sent) / intervalSec).toFixed(0))
      if (h.labels.length > NET_POINTS) {
        h.labels.shift()
        h.recv.shift()
        h.sent.shift()
      }
      updateNetChart(ni.Name)
    })
  }

  lastNetSnapshot = snapshot
}

// ── 数据加载 & 轮询 ───────────────────────────────────────

// 首次加载：重置所有状态，拉取数据，初始化图表
const load = async () => {
  loading.value = true
  destroyNetCharts()
  destroyStatCharts()
  destroyDiskCharts()
  netHistory = {}
  lastNetSnapshot = {}
  diskIOHistory = {}
  lastDiskIOSnapshot = {}
  cpuHistory.labels.length = 0
  cpuHistory.data.length = 0
  memHistory.labels.length = 0
  memHistory.data.length = 0
  try {
    const res = await api.systemStat()
    stat.value = res.payload || null
  } catch (e) {
    stat.value = null
  }
  loading.value = false
  await nextTick()
  initAllNetCharts()
  initAllDiskCharts()
  initStatCharts()
}

// 轮询：更新 CPU、内存、网络数据及图表
const pollNet = async () => {
  try {
    const res = await api.systemStat()
    const payload = res.payload
    if (!payload || !stat.value) return
    stat.value.system.NetInterface = payload.system.NetInterface
    stat.value.system.CpuPercent = payload.system.CpuPercent
    stat.value.system.MemoryUsed = payload.system.MemoryUsed
    stat.value.system.MemoryTotal = payload.system.MemoryTotal
    pushStatPoint(
      cpuPercent(payload.system.CpuPercent),
      memPercent(payload.system.MemoryUsed, payload.system.MemoryTotal)
    )
    updateStatCharts()
    updateNetHistory(physicalInterfaces(payload.system.NetInterface), POLL_INTERVAL / 1000)
    if (payload.diskIO?.length) {
      stat.value.diskIO = payload.diskIO
      updateDiskIOHistory(payload.diskIO, POLL_INTERVAL / 1000)
    }
  } catch (e) { /* ignore */ }
}

const startPoll = () => { pollTimer = setInterval(pollNet, POLL_INTERVAL) }
const stopPoll = () => {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

onUnmounted(() => {
  stopPoll()
  destroyNetCharts()
  destroyDiskCharts()
  destroyStatCharts()
})

defineExpose({
  load: async () => { stopPoll(); await load(); startPoll() },
  stopPoll
})
</script>

<template>
  <div>
    <!-- 加载中 -->
    <div v-if="loading" class="flex items-center justify-center py-10">
      <div class="w-8 h-8 spinner mr-2"></div>
      <span class="text-slate-400 text-sm">加载中...</span>
    </div>

    <div v-else-if="stat" class="space-y-5">
      <!-- 主机基本信息 -->
      <div class="grid grid-cols-1 xs:grid-cols-2 md:grid-cols-4 gap-3">
        <div class="rounded-xl border border-slate-200 bg-white p-4">
          <p class="text-xs text-slate-400 mb-1">主机名</p>
          <p class="text-sm font-semibold text-slate-800 truncate">{{ stat.system.HostName }}</p>
        </div>
        <div class="rounded-xl border border-slate-200 bg-white p-4">
          <p class="text-xs text-slate-400 mb-1">操作系统</p>
          <p class="text-sm font-semibold text-slate-800 truncate">{{ stat.system.Platform }} / {{ stat.system.KernelArch }}</p>
        </div>
        <div class="rounded-xl border border-slate-200 bg-white p-4">
          <p class="text-xs text-slate-400 mb-1">运行时间</p>
          <p class="text-sm font-semibold text-slate-800">{{ fmtUptime(stat.system.Uptime) }}</p>
        </div>
        <div class="rounded-xl border border-slate-200 bg-white p-4">
          <p class="text-xs text-slate-400 mb-1">CPU 核心</p>
          <p class="text-sm font-semibold text-slate-800">{{ stat.system.CpuCore }} 物理 / {{ stat.system.CpuCoreLogic }} 逻辑</p>
        </div>
      </div>

      <!-- CPU & 内存 -->
      <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
        <!-- CPU 使用率卡片：背景折线图 + 顶部进度条 -->
        <div class="relative rounded-xl border border-slate-200 bg-white overflow-hidden">
          <div class="absolute inset-0 pointer-events-none">
            <canvas ref="cpuCanvasRef"></canvas>
          </div>
          <div class="relative p-4">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <div class="w-7 h-7 rounded-lg bg-blue-500/90 flex items-center justify-center">
                  <i class="fas fa-microchip text-white text-xs"></i>
                </div>
                <span class="text-sm font-semibold text-slate-700">CPU 使用率</span>
              </div>
              <span :class="['text-2xl font-bold tabular-nums', textColor(cpuVal)]">
                {{ cpuVal }}<span class="text-sm font-medium ml-0.5">%</span>
              </span>
            </div>
            <p v-if="stat.system.CpuModel?.[0]" class="text-xs text-slate-400 mt-3 truncate">
              {{ stat.system.CpuModel[0] }}
            </p>
            <p v-else class="mt-3 text-xs text-slate-300">—</p>
          </div>
        </div>

        <!-- 内存使用卡片：背景折线图 + 顶部进度条 -->
        <div class="relative rounded-xl border border-slate-200 bg-white overflow-hidden">
          <div class="absolute inset-0 pointer-events-none">
            <canvas ref="memCanvasRef"></canvas>
          </div>
          <div class="relative p-4">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <div class="w-7 h-7 rounded-lg bg-indigo-500/90 flex items-center justify-center">
                  <i class="fas fa-memory text-white text-xs"></i>
                </div>
                <span class="text-sm font-semibold text-slate-700">内存使用</span>
              </div>
              <span :class="['text-2xl font-bold tabular-nums', textColor(memVal)]">
                {{ memVal }}<span class="text-sm font-medium ml-0.5">%</span>
              </span>
            </div>
            <p class="text-xs text-slate-400 mt-3">
              {{ fmtBytes(stat.system.MemoryUsed) }} / {{ fmtBytes(stat.system.MemoryTotal) }}
            </p>
          </div>
        </div>
      </div>

      <!-- 磁盘（容量 + IO 融合卡片） -->
      <div v-if="stat.system.DiskPartition?.length" class="rounded-xl border border-slate-200 bg-white overflow-hidden">
        <div class="px-4 py-3 border-b border-slate-100 flex items-center gap-2">
          <div class="w-6 h-6 rounded-md bg-amber-500 flex items-center justify-center">
            <i class="fas fa-hard-drive text-white text-xs"></i>
          </div>
          <span class="text-sm font-semibold text-slate-700">磁盘</span>
          <span class="ml-auto text-xs text-slate-400">
            总计 {{ fmtBytes(stat.system.DiskTotal) }}，已用 {{ fmtBytes(stat.system.DiskUsed) }}
          </span>
        </div>
        <div ref="diskIOContainerRef" class="divide-y divide-slate-50">
          <div
            v-for="dp in stat.system.DiskPartition"
            :key="dp.Mountpoint"
            class="px-4 py-3"
          >
            <!-- 第一行：挂载点 + 容量 -->
            <div class="flex flex-col sm:flex-row sm:items-center gap-2 sm:gap-3 mb-2">
              <div class="w-full sm:w-24 shrink-0">
                <p class="text-xs font-semibold text-slate-700 truncate">{{ dp.Mountpoint }}</p>
                <p class="text-xs text-slate-400">{{ dp.Device }} · {{ dp.Fstype }}</p>
              </div>
              <div class="flex-1">
                <div class="w-full bg-slate-100 rounded-full h-1.5">
                  <div
                    :class="['h-1.5 rounded-full', barColor(memPercent(dp.Used, dp.Total))]"
                    :style="{ width: memPercent(dp.Used, dp.Total) + '%' }"
                  ></div>
                </div>
              </div>
              <div class="text-right shrink-0 w-full sm:w-28">
                <p class="text-xs text-slate-600">{{ fmtBytes(dp.Used) }} / {{ fmtBytes(dp.Total) }}</p>
                <p class="text-xs text-slate-400">{{ memPercent(dp.Used, dp.Total) }}%</p>
              </div>
            </div>
            <!-- 第二行：IO 速率折线图（仅当有对应设备 IO 数据时显示） -->
            <template v-if="diskIOByDevice(dp.Device)">
              <div class="flex flex-col sm:flex-row sm:items-center justify-between mb-1.5 gap-1">
                <span class="text-xs text-slate-400">IO 速率</span>
                <div class="flex items-center gap-4 text-xs">
                  <span class="flex items-center gap-1">
                    <i class="fas fa-arrow-down text-amber-500"></i>
                    <span class="font-mono text-slate-600 w-20 text-right">{{ fmtRate(currentDiskRate(devShortName(dp.Device), 'read')) }}</span>
                  </span>
                  <span class="flex items-center gap-1">
                    <i class="fas fa-arrow-up text-violet-500"></i>
                    <span class="font-mono text-slate-600 w-20 text-right">{{ fmtRate(currentDiskRate(devShortName(dp.Device), 'write')) }}</span>
                  </span>
                </div>
              </div>
              <div class="relative h-16 bg-slate-50 rounded-lg overflow-hidden">
                <canvas :data-disk="devShortName(dp.Device)"></canvas>
                <div v-if="!diskIOHistory[devShortName(dp.Device)]?.read?.length" class="absolute inset-0 flex items-center justify-center">
                  <span class="text-xs text-slate-300">等待数据...</span>
                </div>
              </div>
              <div class="flex flex-col sm:flex-row gap-2 sm:gap-4 mt-1.5 text-xs text-slate-400">
                <span>累计读: {{ fmtBytes(diskIOByDevice(dp.Device).ReadBytes) }}</span>
                <span>累计写: {{ fmtBytes(diskIOByDevice(dp.Device).WriteBytes) }}</span>
              </div>
            </template>
          </div>
        </div>
      </div>

      <!-- 网络接口 -->
      <div v-if="physicalInterfaces(stat.system.NetInterface).length" class="rounded-xl border border-slate-200 bg-white overflow-hidden">
        <div class="px-4 py-3 border-b border-slate-100 flex items-center gap-2">
          <div class="w-6 h-6 rounded-md bg-cyan-500 flex items-center justify-center">
            <i class="fas fa-network-wired text-white text-xs"></i>
          </div>
          <span class="text-sm font-semibold text-slate-700">网络接口</span>
        </div>
        <div ref="netContainerRef" class="divide-y divide-slate-50">
          <div
            v-for="ni in physicalInterfaces(stat.system.NetInterface)"
            :key="ni.Name"
            class="px-4 py-3"
          >
            <!-- 接口名 + 当前速率 -->
            <div class="flex flex-col sm:flex-row sm:items-center justify-between mb-2 gap-1">
              <p class="text-xs font-semibold text-slate-700">{{ ni.Name }}</p>
              <div class="flex items-center gap-4 text-xs">
                <span class="flex items-center gap-1">
                  <i class="fas fa-arrow-down text-emerald-500"></i>
                  <span class="font-mono text-slate-600 w-20 text-right">{{ fmtRate(currentRate(ni.Name, 'recv')) }}</span>
                </span>
                <span class="flex items-center gap-1">
                  <i class="fas fa-arrow-up text-blue-500"></i>
                  <span class="font-mono text-slate-600 w-20 text-right">{{ fmtRate(currentRate(ni.Name, 'sent')) }}</span>
                </span>
              </div>
            </div>
            <!-- 速率折线图 -->
            <div class="relative h-20 bg-slate-50 rounded-lg overflow-hidden">
              <canvas :data-iface="ni.Name"></canvas>
              <div v-if="!netHistory[ni.Name]" class="absolute inset-0 flex items-center justify-center">
                <span class="text-xs text-slate-300">等待数据...</span>
              </div>
            </div>
            <!-- 累计流量 -->
            <div class="flex flex-col sm:flex-row gap-2 sm:gap-4 mt-1.5 text-xs text-slate-400">
              <span>累计收: {{ fmtBytes(ni.BytesRecv) }}</span>
              <span>累计发: {{ fmtBytes(ni.BytesSent) }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Go 运行态 -->
      <div class="rounded-xl border border-slate-200 bg-white overflow-hidden">
        <div class="px-4 py-3 border-b border-slate-100 flex items-center gap-2">
          <div class="w-6 h-6 rounded-md bg-sky-500 flex items-center justify-center">
            <i class="fas fa-code text-white text-xs"></i>
          </div>
          <span class="text-sm font-semibold text-slate-700">Go 运行态</span>
          <span class="ml-2 text-xs text-slate-400 font-mono">{{ stat.go.version }}</span>
        </div>
        <div class="grid grid-cols-1 xs:grid-cols-2 md:grid-cols-4 divide-x divide-y divide-slate-100">
          <div class="px-4 py-3">
            <p class="text-xs text-slate-400 mb-1">Goroutine 数</p>
            <p class="text-sm font-bold text-slate-800">{{ stat.go.numGoroutine }}</p>
          </div>
          <div class="px-4 py-3">
            <p class="text-xs text-slate-400 mb-1">堆已分配</p>
            <p class="text-sm font-bold text-slate-800">{{ fmtBytes(stat.go.HeapAlloc) }}</p>
          </div>
          <div class="px-4 py-3">
            <p class="text-xs text-slate-400 mb-1">堆使用中</p>
            <p class="text-sm font-bold text-slate-800">{{ fmtBytes(stat.go.HeapInuse) }}</p>
          </div>
          <div class="px-4 py-3">
            <p class="text-xs text-slate-400 mb-1">系统申请</p>
            <p class="text-sm font-bold text-slate-800">{{ fmtBytes(stat.go.Sys) }}</p>
          </div>
          <div class="px-4 py-3">
            <p class="text-xs text-slate-400 mb-1">栈使用中</p>
            <p class="text-sm font-bold text-slate-800">{{ fmtBytes(stat.go.StackInuse) }}</p>
          </div>
          <div class="px-4 py-3">
            <p class="text-xs text-slate-400 mb-1">累计分配</p>
            <p class="text-sm font-bold text-slate-800">{{ fmtBytes(stat.go.TotalAlloc) }}</p>
          </div>
          <div class="px-4 py-3">
            <p class="text-xs text-slate-400 mb-1">GC 次数</p>
            <p class="text-sm font-bold text-slate-800">{{ stat.go.NumGC }}</p>
          </div>
          <div class="px-4 py-3">
            <p class="text-xs text-slate-400 mb-1">最后 GC</p>
            <p class="text-sm font-bold text-slate-800">{{ fmtGCTime(stat.go.LastGC) }}</p>
          </div>
        </div>
      </div>
    </div>

    <!-- 加载失败 -->
    <div v-else class="flex items-center gap-3 py-6 px-4 rounded-xl bg-slate-50">
      <i class="fas fa-triangle-exclamation text-2xl text-slate-300"></i>
      <p class="text-sm text-slate-500">获取系统信息失败</p>
    </div>
  </div>
</template>
