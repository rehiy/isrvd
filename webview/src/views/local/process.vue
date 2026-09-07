<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { SystemProcessInfo } from '@/service/types'

import { formatFileSize } from '@/helper/format'

import PageSearch from '@/component/page-search.vue'

@Component({
    components: { PageSearch }
})
class SystemProcessInfoView extends Vue {
    portal = usePortal()

    processes: SystemProcessInfo[] = []
    loading = false
    searchText = ''

    // ─── 计算属性 ───

    get filteredProcesses() {
        const keyword = this.searchText.trim().toLowerCase()
        if (!keyword) return this.processes
        return this.processes.filter(p =>
            p.name.toLowerCase().includes(keyword) ||
            String(p.pid).includes(keyword) ||
            p.username.toLowerCase().includes(keyword) ||
            (p.cmdline || '').toLowerCase().includes(keyword)
        )
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

    formatPercent(value: number) {
        return `${value.toFixed(1)}%`
    }

    formatStartTime(ts: number) {
        if (!ts) return '-'
        return new Date(ts).toLocaleString('zh-CN', { hour12: false })
    }

    // ─── 数据加载 ───

    async loadProcesses() {
        if (!this.canList) return

        this.loading = true
        try {
            const res = await api.localProcessList()
            this.processes = res.payload?.processes || []
        } catch {
            this.portal.showNotification('error', '获取进程列表失败')
        } finally {
            this.loading = false
        }
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
            <p class="text-xs text-slate-500">查看本机运行中的进程，并可按需终止</p>
          </div>
        </div>
        <div class="action-group">
          <PageSearch v-model="searchText" search-key="local-process" placeholder="搜索进程名、PID、用户、命令行..." aria-label="搜索本机进程" type-to-search />
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
              <th class="w-20 th">PID</th>
              <th class="th">进程</th>
              <th class="w-24 th">用户</th>
              <th class="w-20 th">CPU</th>
              <th class="w-24 th">内存</th>
              <th class="w-36 th">启动时间</th>
              <th v-if="canKill" class="w-40 th-right">操作</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <tr v-for="proc in filteredProcesses" :key="proc.pid" class="hover:bg-slate-50 transition-colors">
              <td class="px-4 py-3 font-mono text-xs text-slate-600">{{ proc.pid }}</td>
              <td class="px-4 py-3 max-w-[320px]">
                <div class="inline-info">
                  <div class="row-icon bg-primary-400">
                    <i class="fas fa-microchip text-white text-sm"></i>
                  </div>
                  <div class="min-w-0">
                    <span class="item-title">{{ proc.name || '-' }}</span>
                    <span v-if="proc.cmdline" class="item-subtitle truncate">{{ proc.cmdline }}</span>
                  </div>
                </div>
              </td>
              <td class="td-text">{{ proc.username || '-' }}</td>
              <td class="td-text">{{ formatPercent(proc.cpuPercent) }}</td>
              <td class="td-text">
                {{ formatMemory(proc.memoryRss) }}
                <span class="text-xs text-slate-400">({{ formatPercent(proc.memoryPercent) }})</span>
              </td>
              <td class="td-text">{{ formatStartTime(proc.createTime) }}</td>
              <td v-if="canKill" class="px-4 py-3">
                <div class="table-actions">
                  <button class="btn-icon btn-icon-amber" title="终止进程" @click="openKill(proc)">
                    <i class="fas fa-stop text-xs"></i>
                  </button>
                  <button class="btn-icon btn-icon-red" title="强制终止" @click="openKill(proc, true)">
                    <i class="fas fa-skull-crossbones text-xs"></i>
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="card-body md:hidden space-y-3">
        <div v-for="proc in filteredProcesses" :key="proc.pid" class="card-interactive">
          <div class="card-info-row">
            <div class="list-icon bg-primary-400">
              <i class="fas fa-microchip text-white text-base"></i>
            </div>
            <div class="min-w-0">
              <span class="item-title-sm">{{ proc.name || '-' }}</span>
              <span class="item-subtitle">PID {{ proc.pid }} · {{ proc.username || '-' }}</span>
            </div>
          </div>
          <div class="card-prop-row">
            <span class="text-xs text-slate-400 flex-shrink-0">CPU</span>
            <span class="text-xs text-slate-500">{{ formatPercent(proc.cpuPercent) }}</span>
          </div>
          <div class="card-prop-row">
            <span class="text-xs text-slate-400 flex-shrink-0">内存</span>
            <span class="text-xs text-slate-500">{{ formatMemory(proc.memoryRss) }}（{{ formatPercent(proc.memoryPercent) }}）</span>
          </div>
          <div class="card-prop-row">
            <span class="text-xs text-slate-400 flex-shrink-0">启动</span>
            <span class="text-xs text-slate-500">{{ formatStartTime(proc.createTime) }}</span>
          </div>
          <div v-if="canKill" class="card-actions">
            <button class="btn-icon btn-icon-amber" title="终止进程" @click="openKill(proc)">
              <i class="fas fa-stop text-xs"></i><span class="text-xs ml-1">终止</span>
            </button>
            <button class="btn-icon btn-icon-red" title="强制终止" @click="openKill(proc, true)">
              <i class="fas fa-skull-crossbones text-xs"></i><span class="text-xs ml-1">强制终止</span>
            </button>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>
