<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { SwarmNodeDetail } from '@/service/types'

import { formatFileSize, formatTime } from '@/helper/utils'

@Component
class NodeDetail extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    nodeData: SwarmNodeDetail | null = null
    loading = false
    formatFileSize = formatFileSize
    formatTime = formatTime

    get nodeId() {
        return this.$route.params.id as string
    }

    // ─── 方法 ───
    nodeStateClass(state: string) {
        if (state === 'ready') return 'bg-emerald-100 text-emerald-700'
        if (state === 'down') return 'bg-red-100 text-red-700'
        return 'bg-slate-100 text-slate-600'
    }

    availabilityClass(avail: string) {
        if (avail === 'active') return 'bg-emerald-100 text-emerald-700'
        if (avail === 'drain') return 'bg-amber-100 text-amber-700'
        if (avail === 'pause') return 'bg-slate-100 text-slate-600'
        return 'bg-slate-100 text-slate-500'
    }

    async loadDetail() {
        this.loading = true
        try {
            const res = await api.swarmNode(this.nodeId)
            this.nodeData = res.payload ?? null
        } catch {
            this.portal.showNotification('error', '获取节点详情失败')
        }
        this.loading = false
    }

    // ─── 生命周期 ───
    mounted() {
        this.loadDetail()
    }
}

export default toNative(NodeDetail)
</script>

<template>
  <div class="card">
    <!-- Toolbar -->
    <div class="card-toolbar">
      <!-- 桌面端 -->
      <div class="hidden md:flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="page-icon bg-blue-500">
            <i class="fas fa-server text-white"></i>
          </div>
          <div>
            <h1 class="text-lg font-semibold text-slate-800">节点详情</h1>
            <p class="text-xs text-slate-600 font-mono truncate max-w-xs">Node ID: {{ nodeId }}</p>
          </div>
        </div>
        <div class="flex items-center gap-2">
          <button class="btn btn-secondary" @click="loadDetail()">
            <i class="fas fa-rotate"></i>刷新
          </button>
        </div>
      </div>
      <!-- 移动端 -->
      <div class="flex md:hidden items-center justify-between">
        <div class="flex items-center gap-3 min-w-0 flex-1">
          <div class="page-icon bg-blue-500">
            <i class="fas fa-server text-white"></i>
          </div>
          <div class="min-w-0">
            <h1 class="text-lg font-semibold text-slate-800 truncate">节点详情</h1>
            <p class="text-xs text-slate-600 font-mono truncate">{{ nodeId }}</p>
          </div>
        </div>
        <button class="btn btn-secondary w-9 h-9 !px-0" title="刷新" @click="loadDetail()">
          <i class="fas fa-rotate text-sm"></i>
        </button>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="card-body">
      <div class="empty-state">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>
    </div>

    <!-- Detail -->
    <div v-else-if="nodeData" class="card-body space-y-4 text-sm">
      <!-- 基本信息 -->
      <div class="detail-card detail-card-blue">
        <div class="detail-card-bar detail-card-bar-blue"></div>
        <h2 class="detail-card-title"><div class="detail-card-icon bg-gradient-to-br from-blue-500 to-blue-600"><i class="fas fa-info-circle text-white text-[9px]"></i></div>基本信息</h2>
        <div class="detail-card-body">
          <div class="detail-row"><span class="detail-label">节点 ID</span><code class="detail-value code">{{ nodeData.id }}</code></div>
            <div class="detail-row"><span class="detail-label">主机名</span><span class="detail-value">{{ nodeData.hostname }} <span v-if="nodeData.leader" class="text-xs text-indigo-600"><i class="fas fa-crown mr-1"></i>Leader</span></span></div>
            <div class="detail-row"><span class="detail-label">地址</span><span class="detail-value font-mono">{{ nodeData.addr || '-' }}</span></div>
            <div class="detail-row"><span class="detail-label">角色</span><span class="detail-value capitalize">{{ nodeData.role }}</span></div>
            <div class="detail-row"><span class="detail-label">状态</span><span class="detail-value capitalize">{{ nodeData.state }}</span></div>
            <div class="detail-row"><span class="detail-label">可用性</span><span class="detail-value capitalize">{{ nodeData.availability }}</span></div>
            <div class="detail-row"><span class="detail-label">引擎版本</span><span class="detail-value">{{ nodeData.engineVersion || '-' }}</span></div>
        </div>
      </div>

      <!-- 硬件资源 -->
      <div class="detail-card detail-card-amber">
        <div class="detail-card-bar detail-card-bar-amber"></div>
        <h2 class="detail-card-title"><div class="detail-card-icon bg-gradient-to-br from-amber-500 to-amber-600"><i class="fas fa-microchip text-white text-[9px]"></i></div>硬件资源</h2>
        <div class="detail-card-body">
          <div class="detail-row"><span class="detail-label">操作系统</span><span class="detail-value capitalize">{{ nodeData.os || '-' }}</span></div>
          <div class="detail-row"><span class="detail-label">架构</span><span class="detail-value">{{ nodeData.architecture || '-' }}</span></div>
          <div class="detail-row"><span class="detail-label">CPU 核数</span><span class="detail-value">{{ nodeData.cpus || '-' }}</span></div>
          <div class="detail-row"><span class="detail-label">内存</span><span class="detail-value">{{ nodeData.memoryBytes ? formatFileSize(nodeData.memoryBytes) : '-' }}</span></div>
        </div>
      </div>

      <!-- 时间信息 -->
      <div class="detail-card detail-card-slate">
        <div class="detail-card-bar detail-card-bar-slate"></div>
        <h2 class="detail-card-title"><div class="detail-card-icon bg-gradient-to-br from-slate-500 to-slate-600"><i class="fas fa-clock text-white text-[9px]"></i></div>时间信息</h2>
        <div class="detail-card-body">
          <div class="detail-row"><span class="detail-label">创建时间</span><span class="detail-value">{{ formatTime(nodeData.createdAt) }}</span></div>
          <div class="detail-row"><span class="detail-label">更新时间</span><span class="detail-value">{{ formatTime(nodeData.updatedAt) }}</span></div>
        </div>
      </div>

      <!-- Labels -->
      <div v-if="nodeData.labels && Object.keys(nodeData.labels).length > 0" class="detail-card detail-card-purple">
        <div class="detail-card-bar detail-card-bar-purple"></div>
        <h2 class="detail-card-title"><div class="detail-card-icon bg-gradient-to-br from-purple-500 to-purple-600"><i class="fas fa-tags text-white text-[9px]"></i></div>Labels</h2>
        <div class="detail-grid">
          <div v-for="(val, key) in nodeData.labels" :key="key" class="detail-grid-item">
            <code class="text-blue-600">{{ key }}</code><span class="text-slate-400 mx-1">=</span><code class="text-slate-600">{{ val }}</code>
          </div>
        </div>
      </div>
    </div>

    <!-- Empty -->
    <div v-else class="card-body">
      <div class="empty-state">
        <div class="empty-state-icon">
          <i class="fas fa-server text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium">未找到节点详情</p>
      </div>
    </div>
  </div>
</template>
