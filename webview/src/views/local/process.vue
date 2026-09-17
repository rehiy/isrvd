<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { SystemProcessInfo } from '@/service/types'

import { formatFileSize, POLL_INTERVAL } from '@/helper/format'

import PageSearch from '@/component/page-search.vue'

type ProcessSortKey = 'memoryRss' | 'cpuPercent' | 'ioBPS' | 'createTime' | 'pid' | 'name'

interface ProcessTreeRow {
    process: SystemProcessInfo
    depth: number
}

@Component({
    components: { PageSearch }
})
class SystemProcessInfoView extends Vue {
    portal = usePortal()

    processes: SystemProcessInfo[] = []
    loading = false
    searchText = ''
    sortKey: ProcessSortKey = 'cpuPercent'
    sortDescending = true
    showParentTree = false
    private pollTimer: ReturnType<typeof setInterval> | null = null
    private polling = false

    // ─── 计算属性 ───

    get filteredProcesses(): ProcessTreeRow[] {
        const keyword = this.searchText.trim().toLowerCase()
        const processByPID = new Map(this.processes.map(process => [process.pid, process]))
        const visiblePIDs = new Set<number>()

        for (const process of this.processes) {
            if (!keyword || this.matchesProcess(process, keyword)) {
                visiblePIDs.add(process.pid)
                if (this.showParentTree) {
                    let parent = processByPID.get(process.ppid)
                    while (parent && !visiblePIDs.has(parent.pid)) {
                        visiblePIDs.add(parent.pid)
                        parent = processByPID.get(parent.ppid)
                    }
                }
            }
        }

        const visibleProcesses = this.processes.filter(process => visiblePIDs.has(process.pid))
        if (!this.showParentTree) return this.sortProcesses(visibleProcesses).map(process => ({ process, depth: 0 }))
        return this.flattenProcessTree(visibleProcesses)
    }

    get canList() {
        return this.portal.hasPerm('GET /api/local/processes')
    }

    get canKill() {
        return this.portal.founder && this.portal.hasPerm('POST /api/local/process/:pid/kill')
    }

    // ─── 展示辅助 ───

    formatMemory(bytes: number) {
        return bytes ? formatFileSize(bytes) : '-'
    }

    formatIORate(bytes?: number) {
        return bytes === undefined ? '-' : `${formatFileSize(bytes)}/s`
    }

    ioBPS(process: SystemProcessInfo) {
        return (process.ioReadBps || 0) + (process.ioWriteBps || 0)
    }

    formatPercent(value?: number) {
        return value === undefined ? '-' : `${value.toFixed(1)}%`
    }

    formatCPUTime(milliseconds: number) {
        const seconds = Math.floor(milliseconds / 1000)
        if (seconds < 60) return `${seconds}s`
        const minutes = Math.floor(seconds / 60)
        if (minutes < 60) return `${minutes}m ${seconds % 60}s`
        return `${Math.floor(minutes / 60)}h ${minutes % 60}m`
    }

    formatStartTime(ts: number) {
        if (!ts) return '-'
        return new Date(ts).toLocaleString('zh-CN', { hour12: false })
    }

    processIndent(depth: number) {
        return `${Math.min(depth, 4) * 16}px`
    }

    // ─── 进程树与排序 ───

    matchesProcess(process: SystemProcessInfo, keyword: string) {
        return process.name.toLowerCase().includes(keyword) ||
            String(process.pid).includes(keyword) ||
            String(process.ppid).includes(keyword) ||
            process.username.toLowerCase().includes(keyword) ||
            (process.cmdline || '').toLowerCase().includes(keyword)
    }

    flattenProcessTree(processes: SystemProcessInfo[]): ProcessTreeRow[] {
        const processByPID = new Map(processes.map(process => [process.pid, process]))
        const childrenByPPID = new Map<number, SystemProcessInfo[]>()
        const roots: SystemProcessInfo[] = []

        for (const process of processes) {
            if (process.ppid !== process.pid && processByPID.has(process.ppid)) {
                const children = childrenByPPID.get(process.ppid) || []
                children.push(process)
                childrenByPPID.set(process.ppid, children)
            } else {
                roots.push(process)
            }
        }

        const rows: ProcessTreeRow[] = []
        const visited = new Set<number>()
        const appendBranch = (process: SystemProcessInfo, depth: number) => {
            if (visited.has(process.pid)) return
            visited.add(process.pid)
            rows.push({ process, depth })
            for (const child of this.sortProcesses(childrenByPPID.get(process.pid) || [])) {
                appendBranch(child, depth + 1)
            }
        }

        for (const process of this.sortProcesses(roots)) appendBranch(process, 0)
        for (const process of this.sortProcesses(processes)) appendBranch(process, 0)
        return rows
    }

    sortProcesses(processes: SystemProcessInfo[]) {
        const direction = this.sortDescending ? -1 : 1
        return [...processes].sort((a, b) => {
            let comparison: number
            if (this.sortKey === 'name') {
                comparison = a.name.localeCompare(b.name, 'zh-CN')
            } else if (this.sortKey === 'ioBPS') {
                comparison = this.ioBPS(a) - this.ioBPS(b)
            } else {
                comparison = (a[this.sortKey] || 0) - (b[this.sortKey] || 0)
            }
            if (comparison !== 0) return comparison * direction
            if (this.sortKey === 'cpuPercent' && a.memoryRss !== b.memoryRss) {
                return (a.memoryRss - b.memoryRss) * direction
            }
            return a.pid - b.pid
        })
    }

    toggleSortDirection() {
        this.sortDescending = !this.sortDescending
    }

    // ─── 数据加载 ───

    async loadProcesses(silent = false) {
        if (!this.canList || this.polling) return

        this.polling = true
        this.loading = true
        try {
            const res = await api.localProcessList()
            this.processes = res.payload?.processes || []
        } catch {
            if (!silent) this.portal.showNotification('error', '获取进程列表失败')
        } finally {
            this.loading = false
            this.polling = false
        }
    }

    startPoll() {
        if (this.pollTimer) return
        this.pollTimer = setInterval(() => this.loadProcesses(true), POLL_INTERVAL)
    }

    stopPoll() {
        if (!this.pollTimer) return
        clearInterval(this.pollTimer)
        this.pollTimer = null
    }

    // ─── 操作 ───

    openKill(proc: SystemProcessInfo, force = false) {
        if (!this.canKill) return

        const tip = force
            ? '强制终止不会给进程清理资源的机会，可能导致数据丢失。'
            : '进程会收到终止信号，允许其自行清理后退出。'
        this.portal.showConfirm({
            title: force ? '强制终止进程' : '终止进程',
            message: `确定要${force ? '强制' : ''}终止 <strong class="text-slate-900">${proc.name}</strong>（PID ${proc.pid}）吗？${tip}`,
            icon: 'fa-skull-crossbones',
            iconColor: 'red',
            confirmText: force ? '强制终止' : '确认终止',
            danger: true,
            onConfirm: async () => {
                try {
                    await api.localProcessKill(proc.pid, force)
                    this.portal.showNotification('success', '已发送终止信号')
                    await this.loadProcesses()
                } catch {
                    this.portal.showNotification('error', '终止进程失败')
                }
            }
        })
    }

    mounted() {
        if (!this.canList) {
            this.$router.replace('/overview')
            return
        }

        this.loadProcesses()
        this.startPoll()
    }

    unmounted() {
        this.stopPoll()
    }
}

export default toNative(SystemProcessInfoView)
</script>

<template>
  <div class="page">
    <div class="page-toolbar">
      <div class="toolbar-desktop">
        <div class="flex items-center gap-3">
          <div class="page-icon bg-primary-500">
            <i class="fas fa-microchip text-white"></i>
          </div>
          <div>
            <h1 class="title-text">进程管理</h1>
            <p class="text-xs text-slate-500">查看本机进程与资源占用，并可按需终止</p>
          </div>
        </div>
        <div class="action-group">
          <PageSearch v-model="searchText" search-key="local-process" placeholder="搜索进程名、PID、父 PID、用户、命令行..." aria-label="搜索本机进程" type-to-search />
          <select v-model="showParentTree" class="select-sm min-w-[110px]" aria-label="展示方式">
            <option :value="true">父子树</option>
            <option :value="false">平铺列表</option>
          </select>
          <select v-model="sortKey" class="select-sm min-w-[130px]" aria-label="排序字段">
            <option value="memoryRss">内存占用</option>
            <option value="cpuPercent">CPU 占用</option>
            <option value="ioBPS">I/O 速率</option>
            <option value="createTime">启动时间</option>
            <option value="pid">进程 ID</option>
            <option value="name">进程名称</option>
          </select>
          <button class="btn btn-secondary btn-square" :title="sortDescending ? '降序排列' : '升序排列'" @click="toggleSortDirection()">
            <i :class="['fas', sortDescending ? 'fa-arrow-down' : 'fa-arrow-up', 'text-sm']"></i>
          </button>
          <button class="btn btn-secondary" @click="loadProcesses()">
            <i class="fas fa-rotate"></i>刷新
          </button>
        </div>
      </div>

      <div class="block md:hidden">
        <div class="flex items-center justify-between">
          <div class="title-group">
            <div class="page-icon bg-primary-500">
              <i class="fas fa-microchip text-white"></i>
            </div>
            <div class="min-w-0">
              <h1 class="title-text">进程管理</h1>
              <p class="text-xs text-slate-500 truncate">本机进程</p>
            </div>
          </div>
          <div class="action-group-sm">
            <select v-model="showParentTree" class="w-20 select-sm" aria-label="展示方式">
              <option :value="true">父子树</option>
              <option :value="false">平铺</option>
            </select>
            <select v-model="sortKey" class="w-24 select-sm" aria-label="排序字段">
              <option value="memoryRss">内存</option>
              <option value="cpuPercent">CPU</option>
              <option value="ioBPS">I/O</option>
              <option value="createTime">启动</option>
              <option value="pid">PID</option>
              <option value="name">名称</option>
            </select>
            <button class="btn btn-secondary btn-square" :title="sortDescending ? '降序排列' : '升序排列'" @click="toggleSortDirection()">
              <i :class="['fas', sortDescending ? 'fa-arrow-down' : 'fa-arrow-up', 'text-sm']"></i>
            </button>
            <button class="btn btn-secondary btn-square" title="刷新" @click="loadProcesses()">
              <i class="fas fa-rotate text-sm"></i>
            </button>
          </div>
        </div>
      </div>
    </div>
    <div class="mobile-search">
      <PageSearch v-model="searchText" search-key="local-process" placeholder="搜索进程..." aria-label="搜索本机进程" width-class="w-full" />
    </div>

    <div v-if="loading && processes.length === 0" class="card-body">
      <div class="empty-state">
        <div class="spinner-lg"></div>
        <p class="text-slate-500">加载中...</p>
      </div>
    </div>

    <div v-else-if="filteredProcesses.length === 0" class="card-body">
      <div class="empty-state">
        <div class="empty-state-icon">
          <i class="fas fa-microchip text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">{{ processes.length === 0 ? '未获取到进程' : '未找到匹配进程' }}</p>
        <p class="text-sm text-slate-400">{{ processes.length === 0 ? '请刷新重试' : '尝试更换关键词或清空搜索条件' }}</p>
      </div>
    </div>

    <template v-else>
      <div class="card-table hidden md:block">
        <table class="w-full border-collapse">
          <thead>
            <tr class="bg-slate-100 border-b border-slate-200">
              <th class="w-28 th">{{ showParentTree ? 'PID / PPID' : 'PID' }}</th>
              <th class="th">{{ showParentTree ? '进程树' : '进程' }}</th>
              <th class="w-24 th">用户</th>
              <th class="w-20 th">CPU</th>
              <th class="w-36 th">内存</th>
              <th class="w-36 th">I/O</th>
              <th class="w-36 th">启动时间</th>
              <th v-if="canKill" class="w-40 th-right">操作</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <tr v-for="row in filteredProcesses" :key="row.process.pid" class="hover:bg-slate-50 transition-colors">
              <td class="px-4 py-3 font-mono text-xs text-slate-600">
                <span class="block">{{ row.process.pid }}</span>
                <span v-if="showParentTree && row.process.ppid > 0" class="text-slate-400">PPID {{ row.process.ppid }}</span>
              </td>
              <td class="px-4 py-3 max-w-[320px]">
                <div class="inline-info" :style="{ marginLeft: showParentTree ? processIndent(row.depth) : '0' }">
                  <i v-if="showParentTree && row.depth" class="fas fa-turn-down text-xs text-slate-300 flex-shrink-0" aria-hidden="true"></i>
                  <div class="row-icon bg-primary-400">
                    <i class="fas fa-microchip text-white text-sm"></i>
                  </div>
                  <div class="min-w-0">
                    <span class="item-title">{{ row.process.name || '-' }}</span>
                    <span v-if="row.process.cmdline" class="item-subtitle truncate">{{ row.process.cmdline }}</span>
                  </div>
                </div>
              </td>
              <td class="td-text">{{ row.process.username || '-' }}</td>
              <td class="td-text whitespace-nowrap">
                <span class="block">{{ formatCPUTime(row.process.cpuMillis) }}</span>
                <span class="block text-xs text-slate-400">{{ formatPercent(row.process.cpuPercent) }}</span>
              </td>
              <td class="td-text whitespace-nowrap">
                <span class="block">{{ formatMemory(row.process.memoryRss) }}</span>
                <span class="block text-xs text-slate-400">{{ formatPercent(row.process.memoryPercent) }}</span>
              </td>
              <td class="td-text">
                <span class="block">读 {{ formatIORate(row.process.ioReadBps) }}</span>
                <span class="block text-xs text-slate-400">写 {{ formatIORate(row.process.ioWriteBps) }}</span>
              </td>
              <td class="td-text">{{ formatStartTime(row.process.createTime) }}</td>
              <td v-if="canKill" class="px-4 py-3">
                <div class="table-actions">
                  <button class="btn-icon btn-icon-amber" title="终止进程" @click="openKill(row.process)">
                    <i class="fas fa-stop text-xs"></i>
                  </button>
                  <button class="btn-icon btn-icon-red" title="强制终止" @click="openKill(row.process, true)">
                    <i class="fas fa-skull-crossbones text-xs"></i>
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="card-body md:hidden space-y-3">
        <div v-for="row in filteredProcesses" :key="row.process.pid" class="card-interactive">
          <div class="card-info-row" :style="{ marginLeft: showParentTree ? processIndent(row.depth) : '0' }">
            <i v-if="showParentTree && row.depth" class="fas fa-turn-down text-xs text-slate-300 flex-shrink-0" aria-hidden="true"></i>
            <div class="list-icon bg-primary-400">
              <i class="fas fa-microchip text-white text-base"></i>
            </div>
            <div class="min-w-0">
              <span class="item-title-sm">{{ row.process.name || '-' }}</span>
              <span class="item-subtitle">PID {{ row.process.pid }}<template v-if="showParentTree && row.process.ppid > 0"> · PPID {{ row.process.ppid }}</template> · {{ row.process.username || '-' }}</span>
            </div>
          </div>
          <div class="card-prop-row-start">
            <span class="prop-label-start">CPU</span>
            <span class="text-xs text-slate-500">
              <span class="block">{{ formatCPUTime(row.process.cpuMillis) }}</span>
              <span class="block text-slate-400 mt-0.5">{{ formatPercent(row.process.cpuPercent) }}</span>
            </span>
          </div>
          <div class="card-prop-row-start">
            <span class="prop-label-start">内存</span>
            <span class="text-xs text-slate-500">
              <span class="block">{{ formatMemory(row.process.memoryRss) }}</span>
              <span class="block text-slate-400 mt-0.5">{{ formatPercent(row.process.memoryPercent) }}</span>
            </span>
          </div>
          <div class="card-prop-row">
            <span class="text-xs text-slate-400 flex-shrink-0">I/O</span>
            <span class="text-xs text-slate-500">读 {{ formatIORate(row.process.ioReadBps) }} · 写 {{ formatIORate(row.process.ioWriteBps) }}</span>
          </div>
          <div class="card-prop-row">
            <span class="text-xs text-slate-400 flex-shrink-0">启动</span>
            <span class="text-xs text-slate-500">{{ formatStartTime(row.process.createTime) }}</span>
          </div>
          <div v-if="canKill" class="card-actions">
            <button class="btn-icon btn-icon-amber" title="终止进程" @click="openKill(row.process)">
              <i class="fas fa-stop text-xs"></i><span class="text-xs ml-1">终止</span>
            </button>
            <button class="btn-icon btn-icon-red" title="强制终止" @click="openKill(row.process, true)">
              <i class="fas fa-skull-crossbones text-xs"></i><span class="text-xs ml-1">强制终止</span>
            </button>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>
