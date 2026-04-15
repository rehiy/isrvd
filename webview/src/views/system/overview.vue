<script setup>
import { Chart, registerables } from 'chart.js'
import { nextTick, ref, onUnmounted } from 'vue'
import api from '@/service/api.js'

Chart.register(...registerables)

const stat = ref(null)
const loading = ref(false)

// 网络速率历史数据：{ ifName: { labels: [], recv: [], sent: [] } }
const NET_POINTS = 30  // 保留最近30个点
let netHistory = {}   // 原始数据：{ ifName: { labels, recv, sent } }
let lastNetSnapshot = {}  // 上次采集的原始字节数
let pollTimer = null

// Chart 实例：{ ifName: Chart }
let netCharts = {}
// 网络接口容器 refs
const netContainerRef = ref(null)

// 格式化字节
const fmtBytes = (bytes) => {
  if (!bytes) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let v = bytes
  while (v >= 1024 && i < units.length - 1) { v /= 1024; i++ }
  return `${v.toFixed(1)} ${units[i]}`
}

// 格式化速率（字节/秒 -> 可读）
const fmtRate = (bytesPerSec) => {
  if (!bytesPerSec || bytesPerSec < 0) return '0 B/s'
  const units = ['B/s', 'KB/s', 'MB/s', 'GB/s']
  let i = 0
  let v = bytesPerSec
  while (v >= 1024 && i < units.length - 1) { v /= 1024; i++ }
  return `${v.toFixed(1)} ${units[i]}`
}

// 当前速率（最后一个点）
const currentRate = (name, dir) => {
  const h = netHistory[name]
  if (!h || !h[dir] || !h[dir].length) return 0
  return h[dir][h[dir].length - 1]
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

// CPU 使用率（取平均）
const cpuPercent = (arr) => {
  if (!arr || !arr.length) return 0
  return (arr.reduce((a, b) => a + b, 0) / arr.length).toFixed(1)
}

// 内存使用率
const memPercent = (used, total) => {
  if (!total) return 0
  return ((used / total) * 100).toFixed(1)
}

// 过滤物理网络接口（排除虚拟/容器接口）
const physicalInterfaces = (list) => {
  if (!list) return []
  const virtualPrefixes = ['lo', 'docker', 'veth', 'br-', 'overlay', 'flannel', 'cni', 'tunl', 'dummy', 'virbr']
  return list.filter(ni => !virtualPrefixes.some(p => ni.Name.startsWith(p)))
}

// 进度条颜色
const barColor = (pct) => {
  const p = parseFloat(pct)
  if (p >= 90) return 'bg-red-500'
  if (p >= 70) return 'bg-amber-500'
  return 'bg-emerald-500'
}

// GC 时间格式化
const fmtGCTime = (ts) => {
  if (!ts) return '从未'
  return new Date(ts * 1000).toLocaleString('zh-CN')
}

// Chart.js 公共配置
const netChartOptions = () => ({
  responsive: true,
  maintainAspectRatio: false,
  animation: false,
  interaction: { intersect: false, mode: 'index' },
  plugins: {
    legend: { display: true, position: 'bottom', labels: { boxWidth: 8, padding: 8, font: { size: 10 }, color: '#64748b' } },
    tooltip: {
      backgroundColor: 'rgba(15,23,42,0.9)',
      titleFont: { size: 10 }, bodyFont: { size: 10 },
      padding: 8, cornerRadius: 6,
      callbacks: { label: ctx => ctx.dataset.label + ': ' + fmtRate(ctx.parsed.y) }
    }
  },
  scales: {
    x: { display: false },
    y: {
      display: true, beginAtZero: true,
      grid: { color: 'rgba(148,163,184,0.08)' },
      border: { display: false },
      ticks: { font: { size: 9 }, color: '#94a3b8', maxTicksLimit: 4, padding: 4, callback: v => fmtRate(v) }
    }
  },
  elements: {
    point: { radius: 0, hoverRadius: 3 },
    line: { tension: 0.4, borderWidth: 1.5 }
  }
})

const hexToRgba = (hex, alpha) => {
  const r = parseInt(hex.slice(1, 3), 16)
  const g = parseInt(hex.slice(3, 5), 16)
  const b = parseInt(hex.slice(5, 7), 16)
  return `rgba(${r},${g},${b},${alpha})`
}

const makeDataset = (data, color, label) => ({
  label,
  data: [...data],
  borderColor: color,
  backgroundColor: hexToRgba(color, 0.1),
  fill: true
})

// 获取某个接口的 canvas 元素
const getNetCanvas = (name) => {
  if (!netContainerRef.value) return null
  return netContainerRef.value.querySelector(`[data-iface="${name}"]`)
}

// 初始化某个接口的 Chart
const initNetChart = (name) => {
  const canvas = getNetCanvas(name)
  if (!canvas) return
  if (netCharts[name]) { netCharts[name].destroy() }
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

// 初始化所有物理接口的图表（load 后调用）
const initAllNetCharts = () => {
  if (!stat.value) return
  const ifaces = physicalInterfaces(stat.value.system.NetInterface)
  ifaces.forEach(ni => {
    if (!netHistory[ni.Name]) {
      netHistory[ni.Name] = { labels: [], recv: [], sent: [] }
    }
    initNetChart(ni.Name)
  })
}

// 更新某个接口的 Chart 数据
const updateNetChart = (name) => {
  const chart = netCharts[name]
  const h = netHistory[name]
  if (!chart || !h) return
  chart.data.labels = [...h.labels]
  chart.data.datasets[0].data = [...h.recv]
  chart.data.datasets[1].data = [...h.sent]
  chart.update('none')
}

// 销毁所有 Chart
const destroyNetCharts = () => {
  Object.values(netCharts).forEach(c => c.destroy())
  netCharts = {}
}

// 更新网络速率历史
const updateNetHistory = async (interfaces, intervalSec) => {
  const snapshot = {}
  interfaces.forEach(ni => {
    snapshot[ni.Name] = { recv: ni.BytesRecv, sent: ni.BytesSent }
  })

  const now = new Date()
  const label = now.getHours().toString().padStart(2, '0') + ':' +
    now.getMinutes().toString().padStart(2, '0') + ':' +
    now.getSeconds().toString().padStart(2, '0')

  if (Object.keys(lastNetSnapshot).length > 0) {
    interfaces.forEach(ni => {
      const prev = lastNetSnapshot[ni.Name]
      if (!prev) return
      const recvRate = Math.max(0, (ni.BytesRecv - prev.recv) / intervalSec)
      const sentRate = Math.max(0, (ni.BytesSent - prev.sent) / intervalSec)

      if (!netHistory[ni.Name]) {
        netHistory[ni.Name] = { labels: [], recv: [], sent: [] }
      }
      const h = netHistory[ni.Name]
      h.labels.push(label)
      h.recv.push(+recvRate.toFixed(0))
      h.sent.push(+sentRate.toFixed(0))
      if (h.labels.length > NET_POINTS) { h.labels.shift(); h.recv.shift(); h.sent.shift() }

      updateNetChart(ni.Name)
    })
  }

  lastNetSnapshot = snapshot
}

const POLL_INTERVAL = 2000  // 2秒

const load = async () => {
  loading.value = true
  // 重置历史数据和图表
  destroyNetCharts()
  netHistory = {}
  lastNetSnapshot = {}
  try {
    const res = await api.systemStat()
    stat.value = res.payload || null
  } catch (e) {
    stat.value = null
  }
  loading.value = false
  // DOM 渲染后初始化空图表
  await nextTick()
  initAllNetCharts()
}

// 轮询（仅更新网络数据，不触发全局 loading）
const pollNet = async () => {
  try {
    const res = await api.systemStat()
    const payload = res.payload
    if (!payload) return
    // 同步更新系统数据
    if (stat.value) {
      stat.value.system.NetInterface = payload.system.NetInterface
      stat.value.system.CpuPercent = payload.system.CpuPercent
      stat.value.system.MemoryUsed = payload.system.MemoryUsed
      stat.value.system.MemoryTotal = payload.system.MemoryTotal
    }
    const ifaces = physicalInterfaces(payload.system.NetInterface)
    await updateNetHistory(ifaces, POLL_INTERVAL / 1000)
  } catch (e) { /* ignore */ }
}

const startPoll = () => {
  pollTimer = setInterval(pollNet, POLL_INTERVAL)
}

const stopPoll = () => {
  if (pollTimer) { clearInterval(pollTimer); pollTimer = null }
}

onUnmounted(() => {
  stopPoll()
  destroyNetCharts()
})

defineExpose({
  load: async () => {
    stopPoll()
    await load()
    startPoll()
  },
  stopPoll
})
</script>

<template>
  <div>
    <div v-if="loading" class="flex items-center justify-center py-10">
      <div class="w-8 h-8 spinner mr-2"></div>
      <span class="text-slate-400 text-sm">加载中...</span>
    </div>

    <div v-else-if="stat" class="space-y-5">
      <!-- 主机基本信息 -->
      <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
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
        <!-- CPU -->
        <div class="rounded-xl border border-slate-200 bg-white p-4">
          <div class="flex items-center justify-between mb-3">
            <div class="flex items-center gap-2">
              <div class="w-7 h-7 rounded-lg bg-blue-500 flex items-center justify-center">
                <i class="fas fa-microchip text-white text-xs"></i>
              </div>
              <span class="text-sm font-semibold text-slate-700">CPU 使用率</span>
            </div>
            <span class="text-lg font-bold text-slate-800">{{ cpuPercent(stat.system.CpuPercent) }}%</span>
          </div>
          <div class="w-full bg-slate-100 rounded-full h-2">
            <div :class="['h-2 rounded-full transition-all', barColor(cpuPercent(stat.system.CpuPercent))]"
              :style="{ width: cpuPercent(stat.system.CpuPercent) + '%' }"></div>
          </div>
          <p v-if="stat.system.CpuModel && stat.system.CpuModel[0]" class="text-xs text-slate-400 mt-2 truncate">
            {{ stat.system.CpuModel[0] }}
          </p>
        </div>

        <!-- 内存 -->
        <div class="rounded-xl border border-slate-200 bg-white p-4">
          <div class="flex items-center justify-between mb-3">
            <div class="flex items-center gap-2">
              <div class="w-7 h-7 rounded-lg bg-indigo-500 flex items-center justify-center">
                <i class="fas fa-memory text-white text-xs"></i>
              </div>
              <span class="text-sm font-semibold text-slate-700">内存使用</span>
            </div>
            <span class="text-lg font-bold text-slate-800">{{ memPercent(stat.system.MemoryUsed, stat.system.MemoryTotal) }}%</span>
          </div>
          <div class="w-full bg-slate-100 rounded-full h-2">
            <div :class="['h-2 rounded-full transition-all', barColor(memPercent(stat.system.MemoryUsed, stat.system.MemoryTotal))]"
              :style="{ width: memPercent(stat.system.MemoryUsed, stat.system.MemoryTotal) + '%' }"></div>
          </div>
          <p class="text-xs text-slate-400 mt-2">
            {{ fmtBytes(stat.system.MemoryUsed) }} / {{ fmtBytes(stat.system.MemoryTotal) }}
          </p>
        </div>
      </div>

      <!-- 磁盘分区 -->
      <div v-if="stat.system.DiskPartition && stat.system.DiskPartition.length" class="rounded-xl border border-slate-200 bg-white overflow-hidden">
        <div class="px-4 py-3 border-b border-slate-100 flex items-center gap-2">
          <div class="w-6 h-6 rounded-md bg-amber-500 flex items-center justify-center">
            <i class="fas fa-hard-drive text-white text-xs"></i>
          </div>
          <span class="text-sm font-semibold text-slate-700">磁盘分区</span>
          <span class="ml-auto text-xs text-slate-400">总计 {{ fmtBytes(stat.system.DiskTotal) }}，已用 {{ fmtBytes(stat.system.DiskUsed) }}</span>
        </div>
        <div class="divide-y divide-slate-50">
          <div v-for="dp in stat.system.DiskPartition" :key="dp.Mountpoint" class="px-4 py-2.5 flex items-center gap-3">
            <div class="w-24 shrink-0">
              <p class="text-xs font-medium text-slate-700 truncate">{{ dp.Mountpoint }}</p>
              <p class="text-xs text-slate-400">{{ dp.Fstype }}</p>
            </div>
            <div class="flex-1">
              <div class="w-full bg-slate-100 rounded-full h-1.5">
                <div :class="['h-1.5 rounded-full', barColor(memPercent(dp.Used, dp.Total))]"
                  :style="{ width: memPercent(dp.Used, dp.Total) + '%' }"></div>
              </div>
            </div>
            <div class="text-right shrink-0 w-28">
              <p class="text-xs text-slate-600">{{ fmtBytes(dp.Used) }} / {{ fmtBytes(dp.Total) }}</p>
              <p class="text-xs text-slate-400">{{ memPercent(dp.Used, dp.Total) }}%</p>
            </div>
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
          <span class="ml-auto text-xs text-slate-400">每 2s 刷新</span>
        </div>
        <div ref="netContainerRef" class="divide-y divide-slate-50">
          <div v-for="ni in physicalInterfaces(stat.system.NetInterface)" :key="ni.Name" class="px-4 py-3">
            <!-- 接口名 + 当前速率 -->
            <div class="flex items-center justify-between mb-2">
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
            <!-- 折线图 -->
            <div class="relative h-20 bg-slate-50 rounded-lg overflow-hidden">
              <canvas :data-iface="ni.Name"></canvas>
              <div v-if="!netHistory[ni.Name]" class="absolute inset-0 flex items-center justify-center">
                <span class="text-xs text-slate-300">等待数据...</span>
              </div>
            </div>
            <!-- 累计流量 -->
            <div class="flex gap-4 mt-1.5 text-xs text-slate-400">
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
        <div class="grid grid-cols-2 md:grid-cols-4 divide-x divide-y divide-slate-100">
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

    <div v-else class="flex items-center gap-3 py-6 px-4 rounded-xl bg-slate-50">
      <i class="fas fa-triangle-exclamation text-2xl text-slate-300"></i>
      <p class="text-sm text-slate-500">获取系统信息失败</p>
    </div>
  </div>
</template>
