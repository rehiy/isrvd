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
  <div>
    <div class="card mb-4">
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
      <div v-if="loading" class="empty-state">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>

      <!-- Detail -->
      <div v-else-if="nodeData" class="p-4 md:p-6 space-y-6 text-sm">
        <!-- 基本信息 -->
        <div>
          <h2 class="section-title">基本信息</h2>
          <div class="grid grid-cols-2 gap-3">
            <div class="col-span-2">
              <label class="form-label">节点 ID</label>
              <code class="block px-3 py-2 bg-slate-50 rounded-lg text-xs font-mono text-slate-700 break-all">{{ nodeData.id }}</code>
            </div>
            <div>
              <label class="form-label">主机名</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700 flex items-center gap-2">
                {{ nodeData.hostname }}
                <span v-if="nodeData.leader" class="text-xs text-indigo-600"><i class="fas fa-crown mr-1"></i>Leader</span>
              </div>
            </div>
            <div>
              <label class="form-label">地址</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg font-mono text-slate-700">{{ nodeData.addr || '-' }}</div>
            </div>
            <div>
              <label class="form-label">角色</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700 capitalize">{{ nodeData.role }}</div>
            </div>
            <div>
              <label class="form-label">状态</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700 capitalize">{{ nodeData.state }}</div>
            </div>
            <div>
              <label class="form-label">可用性</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700 capitalize">{{ nodeData.availability }}</div>
            </div>
            <div>
              <label class="form-label">引擎版本</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700">{{ nodeData.engineVersion || '-' }}</div>
            </div>
          </div>
        </div>

        <!-- 硬件资源 -->
        <div>
          <h2 class="section-title">硬件资源</h2>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="form-label">操作系统</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700 capitalize">{{ nodeData.os || '-' }}</div>
            </div>
            <div>
              <label class="form-label">架构</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700">{{ nodeData.architecture || '-' }}</div>
            </div>
            <div>
              <label class="form-label">CPU 核数</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700">{{ nodeData.cpus || '-' }}</div>
            </div>
            <div>
              <label class="form-label">内存</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700">{{ nodeData.memoryBytes ? formatFileSize(nodeData.memoryBytes) : '-' }}</div>
            </div>
          </div>
        </div>

        <!-- 时间信息 -->
        <div>
          <h2 class="section-title">时间信息</h2>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="form-label">创建时间</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700">{{ formatTime(nodeData.createdAt) }}</div>
            </div>
            <div>
              <label class="form-label">更新时间</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700">{{ formatTime(nodeData.updatedAt) }}</div>
            </div>
          </div>
        </div>

        <!-- Labels -->
        <div v-if="nodeData.labels && Object.keys(nodeData.labels).length > 0">
          <h2 class="section-title">Labels</h2>
          <div class="border border-slate-200 rounded-lg divide-y divide-slate-100">
            <div v-for="(val, key) in nodeData.labels" :key="key" class="px-3 py-1.5 flex gap-2">
              <code class="text-xs font-mono text-blue-600 shrink-0">{{ key }}</code>
              <span class="text-slate-400">=</span>
              <code class="text-xs font-mono text-slate-600 break-all">{{ val }}</code>
            </div>
          </div>
        </div>
      </div>

      <!-- Empty -->
      <div v-else class="empty-state">
        <div class="empty-state-icon">
          <i class="fas fa-server text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium">未找到节点详情</p>
      </div>
    </div>
  </div>
</template>
