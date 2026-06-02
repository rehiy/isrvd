<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { DockerContainerDetail, DockerContainerInfo, DockerVolumeMapping } from '@/service/types'

import { formatTime } from '@/helper/utils'

@Component
class ContainerDetail extends Vue {
    portal = usePortal()
    container: DockerContainerInfo | null = null
    detail: DockerContainerDetail | null = null
    loading = false

    get containerId() {
        return this.$route.params.id as string
    }

    get portEntries() {
        if (!this.detail?.ports) return []
        return Object.entries(this.detail.ports)
    }

    get labelEntries() {
        if (!this.detail?.labels) return []
        return Object.entries(this.detail.labels)
    }

    get envList() {
        return this.detail?.env || []
    }

    get cmdText() {
        return (this.detail?.cmd || []).join(' ')
    }

    formatVolume(v: DockerVolumeMapping) {
        return `${v.source || v.hostPath || '-'} → ${v.containerPath}${v.readOnly ? ' (ro)' : ''}`
    }

    async loadContainer() {
        try {
            const res = await api.dockerContainerList(true)
            this.container = (res.payload || []).find((c: DockerContainerInfo) => c.id === this.containerId) ?? null
            if (!this.container) {
                this.portal.showNotification('error', '容器不存在')
                this.$router.push('/docker/containers')
            }
        } catch {
            this.portal.showNotification('error', '加载容器信息失败')
            this.$router.push('/docker/containers')
        }
    }

    async loadDetail() {
        this.loading = true
        try {
            const res = await api.dockerContainer(this.containerId)
            this.detail = res.payload || null
        } catch {
            this.portal.showNotification('error', '加载容器详情失败')
        }
        this.loading = false
    }

    mounted() {
        this.loadContainer()
        this.loadDetail()
    }

    formatTime = formatTime
}

export default toNative(ContainerDetail)
</script>

<template>
  <div class="card">
    <div class="card-toolbar">
      <!-- 桌面端 -->
      <div class="hidden md:flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div :class="['page-icon', container?.state === 'running' ? 'bg-emerald-400' : 'bg-slate-400']">
            <i class="fas fa-cube text-white text-sm"></i>
          </div>
          <div class="min-w-0">
            <h1 class="text-lg font-semibold text-slate-800 truncate">{{ container ? (container.name || container.id) : '加载中...' }}</h1>
            <p class="text-xs text-slate-600 font-mono truncate">{{ container?.image }}</p>
          </div>
        </div>
        <div class="flex items-center gap-2">
          <button class="btn btn-secondary" :disabled="loading" @click="loadDetail">
            <i class="fas fa-rotate"></i>刷新
          </button>
        </div>
      </div>
      <!-- 移动端 -->
      <div class="flex md:hidden items-center justify-between">
        <div class="flex items-center gap-3 min-w-0 flex-1">
          <div :class="['page-icon flex-shrink-0', container?.state === 'running' ? 'bg-emerald-400' : 'bg-slate-400']">
            <i class="fas fa-cube text-white text-sm"></i>
          </div>
          <div class="min-w-0">
            <h1 class="text-lg font-semibold text-slate-800 truncate">{{ container ? (container.name || container.id) : '加载中...' }}</h1>
            <p class="text-xs text-slate-600 font-mono truncate">{{ container?.image }}</p>
          </div>
        </div>
        <div class="flex items-center gap-1 flex-shrink-0">
          <button class="btn btn-secondary w-9 h-9 !px-0" title="刷新" :disabled="loading" @click="loadDetail">
            <i class="fas fa-rotate text-sm"></i>
          </button>
        </div>
      </div>
    </div>

    <div v-if="loading" class="empty-state">
      <div class="w-12 h-12 spinner mb-3"></div>
      <p class="text-slate-500">加载中...</p>
    </div>

    <div v-else-if="detail" class="card-body space-y-4">
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <!-- 基本信息 -->
        <div class="detail-card detail-card-emerald">
          <div class="detail-card-bar detail-card-bar-emerald"></div>
          <h2 class="detail-card-title"><div class="detail-card-icon bg-gradient-to-br from-emerald-500 to-emerald-600"><i class="fas fa-circle-info text-white text-[9px]"></i></div>基本信息</h2>
          <div class="detail-card-body">
            <div class="detail-row"><span class="detail-label">名称</span><span class="detail-value">{{ detail.name }}</span></div>
            <div class="detail-row"><span class="detail-label">ID</span><code class="detail-value code">{{ detail.id }}</code></div>
            <div class="detail-row"><span class="detail-label">状态</span><span :class="detail.state === 'running' ? 'text-emerald-600 font-medium' : 'text-slate-500'">{{ detail.state }}</span></div>
            <div class="detail-row"><span class="detail-label">创建时间</span><span class="detail-value">{{ formatTime(detail.createdAt) }}</span></div>
            <div class="detail-row"><span class="detail-label">镜像</span><code class="detail-value code">{{ detail.image }}</code></div>
            <div class="detail-row"><span class="detail-label">重启策略</span><span class="detail-value">{{ detail.restart || 'no' }}</span></div>
          </div>
        </div>

        <!-- 运行配置 -->
        <div class="detail-card detail-card-blue">
          <div class="detail-card-bar detail-card-bar-blue"></div>
          <h2 class="detail-card-title"><div class="detail-card-icon bg-gradient-to-br from-blue-500 to-blue-600"><i class="fas fa-sliders text-white text-[9px]"></i></div>运行配置</h2>
          <div class="detail-card-body">
            <div class="detail-row"><span class="detail-label">网络</span><span class="detail-value">{{ detail.network || '-' }}</span></div>
            <div class="detail-row"><span class="detail-label">工作目录</span><span class="detail-value">{{ detail.workdir || '-' }}</span></div>
            <div class="detail-row"><span class="detail-label">用户</span><span class="detail-value">{{ detail.user || '-' }}</span></div>
            <div class="detail-row"><span class="detail-label">主机名</span><span class="detail-value">{{ detail.hostname || '-' }}</span></div>
            <div class="detail-row"><span class="detail-label">特权模式</span><span class="detail-value">{{ detail.privileged ? '是' : '否' }}</span></div>
            <div class="detail-row"><span class="detail-label">资源限制</span><span class="detail-value">{{ detail.memory || 0 }} MB / {{ detail.cpus || 0 }} CPU</span></div>
          </div>
        </div>
      </div>

      <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <!-- 端口映射 -->
        <div class="detail-card detail-card-indigo">
          <div class="detail-card-bar detail-card-bar-indigo"></div>
          <h2 class="detail-card-title"><div class="detail-card-icon bg-gradient-to-br from-indigo-500 to-indigo-600"><i class="fas fa-network-wired text-white text-[9px]"></i></div>端口映射</h2>
          <div v-if="portEntries.length" class="space-y-2">
            <div v-for="[host, target] in portEntries" :key="host" class="detail-list-item">
              <code class="text-slate-700">{{ host }}</code><i class="fas fa-arrow-right text-slate-300"></i><code class="text-slate-700">{{ target }}</code>
            </div>
          </div>
          <p v-else class="detail-empty">无端口映射</p>
        </div>

        <!-- 挂载 -->
        <div class="detail-card detail-card-amber">
          <div class="detail-card-bar detail-card-bar-amber"></div>
          <h2 class="detail-card-title"><div class="detail-card-icon bg-gradient-to-br from-amber-500 to-amber-600"><i class="fas fa-hard-drive text-white text-[9px]"></i></div>挂载</h2>
          <div v-if="detail.volumes?.length" class="space-y-2">
            <code v-for="vol in detail.volumes" :key="formatVolume(vol)" class="detail-code-block">{{ formatVolume(vol) }}</code>
          </div>
          <p v-else class="detail-empty">无挂载</p>
        </div>
      </div>

      <!-- 命令与环境变量 -->
      <div class="detail-card detail-card-slate">
        <div class="detail-card-bar detail-card-bar-slate"></div>
        <h2 class="detail-card-title"><div class="detail-card-icon bg-gradient-to-br from-slate-500 to-slate-600"><i class="fas fa-terminal text-white text-[9px]"></i></div>命令与环境变量</h2>
        <div class="space-y-4">
          <div>
            <div class="text-xs text-slate-500 mb-1">启动命令</div>
            <code class="detail-code-block">{{ cmdText || '-' }}</code>
          </div>
          <div>
            <div class="text-xs text-slate-500 mb-1">环境变量</div>
            <div v-if="envList.length" class="detail-grid">
              <code v-for="env in envList" :key="env" class="detail-grid-item">{{ env }}</code>
            </div>
            <p v-else class="detail-empty">无环境变量</p>
          </div>
        </div>
      </div>

      <!-- Labels -->
      <div class="detail-card detail-card-purple">
        <div class="detail-card-bar detail-card-bar-purple"></div>
        <h2 class="detail-card-title"><div class="detail-card-icon bg-gradient-to-br from-purple-500 to-purple-600"><i class="fas fa-tags text-white text-[9px]"></i></div>Labels</h2>
        <div v-if="labelEntries.length" class="detail-grid">
          <div v-for="[key, value] in labelEntries" :key="key" class="detail-grid-item">
            <span class="text-slate-500">{{ key }}</span><span class="text-slate-300 mx-1">=</span><span class="text-slate-700">{{ value }}</span>
          </div>
        </div>
        <p v-else class="detail-empty">无标签</p>
      </div>
    </div>

    <div v-else class="empty-state">
      <div class="empty-state-icon">
        <i class="fas fa-cube text-4xl text-slate-300"></i>
      </div>
      <p class="text-slate-600 font-medium">未找到容器详情</p>
    </div>
  </div>
</template>
