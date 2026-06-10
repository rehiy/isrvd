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
            this.$router.push('/docker/containers')
        }
    }

    async loadDetail() {
        this.loading = true
        try {
            const res = await api.dockerContainerInspect(this.containerId)
            this.detail = res.payload || null
        } finally {
            this.loading = false
        }
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

    <div v-if="loading" class="card-body">
      <div class="empty-state">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>
    </div>

    <div v-else-if="detail" class="card-body space-y-4 text-sm">
      <!-- 基本信息 -->
      <div>
        <h2 class="section-title">基本信息</h2>
        <div class="grid grid-cols-2 gap-3">
          <div class="col-span-2">
            <label class="form-label">名称</label>
            <div class="detail-value text-slate-700 break-all">{{ detail.name }}</div>
          </div>
          <div class="col-span-2">
            <label class="form-label">ID</label>
            <code class="detail-value-mono">{{ detail.id }}</code>
          </div>
          <div>
            <label class="form-label">状态</label>
            <div :class="['detail-value font-medium', detail.state === 'running' ? 'text-emerald-600' : 'text-slate-500']">{{ detail.state }}</div>
          </div>
          <div>
            <label class="form-label">重启策略</label>
            <div class="detail-value text-slate-700">{{ detail.restart || 'no' }}</div>
          </div>
          <div>
            <label class="form-label">创建时间</label>
            <div class="detail-value text-slate-700">{{ formatTime(detail.createdAt) }}</div>
          </div>
          <div>
            <label class="form-label">镜像</label>
            <code class="detail-value-mono">{{ detail.image }}</code>
          </div>
        </div>
      </div>

      <!-- 运行配置 -->
      <div>
        <h2 class="section-title">运行配置</h2>
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="form-label">网络</label>
            <div class="detail-value text-slate-700">{{ detail.network || '-' }}</div>
          </div>
          <div>
            <label class="form-label">主机名</label>
            <div class="detail-value text-slate-700">{{ detail.hostname || '-' }}</div>
          </div>
          <div>
            <label class="form-label">工作目录</label>
            <div class="detail-value font-mono text-slate-700">{{ detail.workdir || '-' }}</div>
          </div>
          <div>
            <label class="form-label">用户</label>
            <div class="detail-value text-slate-700">{{ detail.user || '-' }}</div>
          </div>
          <div>
            <label class="form-label">特权模式</label>
            <div class="detail-value text-slate-700">{{ detail.privileged ? '是' : '否' }}</div>
          </div>
          <div>
            <label class="form-label">资源限制</label>
            <div class="detail-value text-slate-700">{{ detail.memory || 0 }} MB / {{ detail.cpus || 0 }} CPU</div>
          </div>
        </div>
      </div>

      <!-- 端口映射 -->
      <div>
        <h2 class="section-title section-title-table">端口映射</h2>
        <div v-if="portEntries.length" class="border-x border-b border-slate-200 rounded-b-xl overflow-hidden">
          <table class="w-full">
            <thead>
              <tr class="bg-slate-50 border-b border-slate-200">
                <th class="th-sm">主机端口</th>
                <th class="th-sm">容器端口</th>
              </tr>
            </thead>
            <tbody class="bg-white divide-y divide-slate-100">
              <tr v-for="[host, target] in portEntries" :key="host" class="hover:bg-slate-50 transition-colors">
                <td class="px-3 py-2 font-mono text-xs text-slate-700">{{ host }}</td>
                <td class="px-3 py-2 font-mono text-xs text-slate-700">{{ target }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-else class="text-sm text-slate-400 py-6 text-center bg-slate-50 rounded-xl">无端口映射</div>
      </div>

      <!-- 挂载 -->
      <div>
        <h2 class="section-title">挂载</h2>
        <div v-if="detail.volumes?.length" class="space-y-2">
          <code v-for="vol in detail.volumes" :key="formatVolume(vol)" class="detail-value-mono">{{ formatVolume(vol) }}</code>
        </div>
        <div v-else class="text-sm text-slate-400 py-6 text-center bg-slate-50 rounded-xl">无挂载</div>
      </div>

      <!-- 命令与环境变量 -->
      <div>
        <h2 class="section-title">命令与环境变量</h2>
        <div class="space-y-3">
          <div>
            <label class="form-label">启动命令</label>
            <code class="detail-value-mono">{{ cmdText || '-' }}</code>
          </div>
          <div>
            <label class="form-label">环境变量</label>
            <div v-if="envList.length" class="space-y-1">
              <code v-for="env in envList" :key="env" class="detail-value-mono">{{ env }}</code>
            </div>
            <div v-else class="text-sm text-slate-400 py-6 text-center bg-slate-50 rounded-xl">无环境变量</div>
          </div>
        </div>
      </div>

      <!-- Labels -->
      <div>
        <h2 class="section-title section-title-table">Labels</h2>
        <div v-if="labelEntries.length" class="border-x border-b border-slate-200 rounded-b-xl overflow-hidden">
          <table class="w-full">
            <thead>
              <tr class="bg-slate-50 border-b border-slate-200">
                <th class="th-sm">Key</th>
                <th class="th-sm">Value</th>
              </tr>
            </thead>
            <tbody class="bg-white divide-y divide-slate-100">
              <tr v-for="[key, value] in labelEntries" :key="key" class="hover:bg-slate-50 transition-colors">
                <td class="px-3 py-2 font-mono text-xs text-slate-500">{{ key }}</td>
                <td class="px-3 py-2 font-mono text-xs text-slate-700">{{ value }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-else class="text-sm text-slate-400 py-6 text-center bg-slate-50 rounded-xl">无标签</div>
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
