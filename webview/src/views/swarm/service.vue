<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { SwarmServiceDetail, SwarmTask } from '@/service/types'

import { formatTime } from '@/helper/utils'

import { usePortal } from '@/stores'

@Component
class ServiceInfo extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    serviceData: SwarmServiceDetail | null = null
    formatTime = formatTime
    tasks: SwarmTask[] = []
    loading = false

    get serviceId() {
        return this.$route.params.id as string
    }

    // ─── 方法 ───
    activeTab() {
        return this.$route.name
    }

    switchTab(name: string) {
        this.$router.push({ name, params: { id: this.serviceId } })
    }

    get nodeDistribution() {
        const map = new Map<string, { nodeID: string; nodeName: string; running: number; total: number }>()
        for (const t of this.tasks) {
            const key = t.nodeID || t.nodeName
            if (!map.has(key)) {
                map.set(key, { nodeID: t.nodeID, nodeName: t.nodeName || t.nodeID, running: 0, total: 0 })
            }
            const entry = map.get(key)
            if (!entry) continue
            entry.total++
            if (t.state === 'running') entry.running++
        }
        return [...map.values()].sort((a, b) => b.running - a.running)
    }

    async loadDetail() {
        this.loading = true
        try {
            const [detailRes, tasksRes] = await Promise.all([
                api.swarmService(this.serviceId),
                api.swarmTaskList(this.serviceId),
            ])
            this.serviceData = detailRes.payload ?? null
            this.tasks = tasksRes.payload ?? []
        } catch {
            this.portal.showNotification('error', '获取服务详情失败')
        }
        this.loading = false
    }

    // ─── 生命周期 ───
    mounted() {
        this.loadDetail()
    }
}

export default toNative(ServiceInfo)
</script>

<template>
  <div>
    <div class="card mb-4">
      <!-- Toolbar -->
      <div class="card-toolbar">
        <!-- 桌面端 -->
        <div class="hidden md:flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="page-icon bg-emerald-500">
              <i class="fas fa-cubes text-white"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">{{ serviceData?.name || '服务详情' }}</h1>
              <p class="text-xs text-slate-600 font-mono truncate max-w-xs">{{ serviceId }}</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <div class="flex gap-1 bg-slate-100 p-1 rounded-lg">
              <button v-if="portal.hasPerm('GET /api/swarm/service/:id')" :class="['tab-btn', activeTab() === 'swarm-service' ? 'tab-btn-active text-emerald-600' : 'tab-btn-inactive']" @click="switchTab('swarm-service')">
                <i class="fas fa-circle-info"></i><span>详情</span>
              </button>
              <button v-if="portal.hasPerm('GET /api/swarm/service/:id/logs')" :class="['tab-btn', activeTab() === 'swarm-service-logs' ? 'tab-btn-active text-emerald-600' : 'tab-btn-inactive']" @click="switchTab('swarm-service-logs')">
                <i class="fas fa-file-lines"></i><span>日志</span>
              </button>
            </div>
            <button class="btn btn-secondary" @click="loadDetail()">
              <i class="fas fa-rotate"></i>刷新
            </button>
          </div>
        </div>
        <!-- 移动端 -->
        <div class="block md:hidden">
          <div class="flex items-center justify-between mb-3">
            <div class="flex items-center gap-3 min-w-0 flex-1">
              <div class="page-icon bg-emerald-500">
                <i class="fas fa-cubes text-white"></i>
              </div>
              <div class="min-w-0">
                <h1 class="text-lg font-semibold text-slate-800 truncate">{{ serviceData?.name || '服务详情' }}</h1>
                <p class="text-xs text-slate-600 font-mono truncate">{{ serviceId.slice(0, 12) }}</p>
              </div>
            </div>
            <button class="btn btn-secondary w-9 h-9 !px-0" title="刷新" @click="loadDetail()">
              <i class="fas fa-rotate text-sm"></i>
            </button>
          </div>
          <div class="flex justify-center gap-1 bg-slate-100 p-1 rounded-lg">
            <button v-if="portal.hasPerm('GET /api/swarm/service/:id')" :class="['tab-btn', activeTab() === 'swarm-service' ? 'tab-btn-active text-emerald-600' : 'tab-btn-inactive']" @click="switchTab('swarm-service')">
              <i class="fas fa-circle-info"></i><span>详情</span>
            </button>
            <button v-if="portal.hasPerm('GET /api/swarm/service/:id/logs')" :class="['tab-btn', activeTab() === 'swarm-service-logs' ? 'tab-btn-active text-emerald-600' : 'tab-btn-inactive']" @click="switchTab('swarm-service-logs')">
              <i class="fas fa-file-lines"></i><span>日志</span>
            </button>
          </div>
        </div>
      </div>

      <!-- Loading -->
      <div v-if="loading" class="empty-state">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>

      <!-- 未找到 -->
      <div v-else-if="!serviceData" class="empty-state">
        <div class="empty-state-icon">
          <i class="fas fa-cubes text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium">未找到服务详情</p>
      </div>

      <!-- 内容 -->
      <div v-else class="p-4 md:p-6 space-y-6 text-sm">
        <!-- 基本信息 -->
        <div>
          <h2 class="section-title">基本信息</h2>
          <div class="grid grid-cols-2 gap-3">
            <div class="col-span-2">
              <label class="form-label">服务 ID</label>
              <code class="block px-3 py-2 bg-slate-50 rounded-lg text-xs font-mono text-slate-700 break-all">{{ serviceData.id }}</code>
            </div>
            <div class="col-span-2">
              <label class="form-label">镜像</label>
              <code class="block px-3 py-2 bg-slate-50 rounded-lg text-xs font-mono text-slate-700 break-all">{{ serviceData.image }}</code>
            </div>
            <div>
              <label class="form-label">模式</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700 capitalize">{{ serviceData.mode }}</div>
            </div>
            <div>
              <label class="form-label">副本</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700">
                <span class="text-emerald-600 font-medium">{{ serviceData.runningTasks }}</span>
                <span v-if="serviceData.mode === 'replicated'" class="text-slate-400"> / {{ serviceData.replicas ?? '?' }}</span>
                <span v-else class="text-slate-400"> 运行中</span>
              </div>
            </div>
            <div>
              <label class="form-label">创建时间</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700">{{ formatTime(serviceData.createdAt) }}</div>
            </div>
            <div>
              <label class="form-label">更新时间</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700">{{ formatTime(serviceData.updatedAt) }}</div>
            </div>
          </div>
        </div>

        <!-- 网络 -->
        <div v-if="serviceData.networks && serviceData.networks.length > 0">
          <h2 class="section-title">网络</h2>
          <div class="flex flex-wrap gap-1.5">
            <span v-for="n in serviceData.networks" :key="n" class="inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-medium bg-blue-50 text-blue-700">{{ n }}</span>
          </div>
        </div>

        <!-- 端口 -->
        <div v-if="serviceData.ports && serviceData.ports.length > 0">
          <h2 class="section-title">端口映射</h2>
          <div class="border border-slate-200 rounded-lg divide-y divide-slate-100">
            <div v-for="(p, idx) in serviceData.ports" :key="idx" class="px-3 py-2 flex items-center gap-3">
              <code class="text-xs font-mono text-emerald-700 font-medium">{{ p.publishedPort }}</code>
              <i class="fas fa-arrow-right text-slate-300 text-xs"></i>
              <code class="text-xs font-mono text-slate-600">{{ p.targetPort }}/{{ p.protocol }}</code>
              <span class="ml-auto text-xs text-slate-400 capitalize">{{ p.publishMode }}</span>
            </div>
          </div>
        </div>

        <!-- 挂载 -->
        <div v-if="serviceData.mounts && serviceData.mounts.length > 0">
          <h2 class="section-title">挂载</h2>
          <div class="border border-slate-200 rounded-lg divide-y divide-slate-100">
            <div v-for="(mt, idx) in serviceData.mounts" :key="idx" class="px-3 py-2">
              <div class="flex items-center gap-2 text-xs font-mono flex-wrap">
                <span class="text-slate-400 capitalize">{{ mt.type }}</span>
                <code class="text-slate-700">{{ mt.source || '(匿名)' }}</code>
                <i class="fas fa-arrow-right text-slate-300"></i>
                <code class="text-slate-700">{{ mt.target }}</code>
                <span v-if="mt.readOnly" class="ml-auto text-amber-600">只读</span>
              </div>
            </div>
          </div>
        </div>

        <!-- 环境变量 -->
        <div v-if="serviceData.env && serviceData.env.length > 0">
          <h2 class="section-title">环境变量</h2>
          <div class="border border-slate-200 rounded-lg divide-y divide-slate-100">
            <div v-for="(env, idx) in serviceData.env" :key="idx" class="px-3 py-1.5">
              <code class="text-xs font-mono text-slate-600">{{ env }}</code>
            </div>
          </div>
        </div>

        <!-- 启动参数 -->
        <div v-if="serviceData.args && serviceData.args.length > 0">
          <h2 class="section-title">启动参数</h2>
          <code class="block px-3 py-2 bg-slate-50 rounded-lg text-xs font-mono text-slate-700">{{ serviceData.args.join(' ') }}</code>
        </div>

        <!-- 约束 -->
        <div v-if="serviceData.constraints && serviceData.constraints.length > 0">
          <h2 class="section-title">调度约束</h2>
          <div class="border border-slate-200 rounded-lg divide-y divide-slate-100">
            <div v-for="(c, idx) in serviceData.constraints" :key="idx" class="px-3 py-1.5">
              <code class="text-xs font-mono text-slate-600">{{ c }}</code>
            </div>
          </div>
        </div>

        <!-- 节点分布 -->
        <div v-if="nodeDistribution.length > 0 && portal.hasPerm('GET /api/swarm/node/:id')">
          <h2 class="section-title">节点分布</h2>
          <div class="border border-slate-200 rounded-lg divide-y divide-slate-100">
            <div v-for="node in nodeDistribution" :key="node.nodeName" class="px-3 py-2 flex items-center gap-3 transition-colors cursor-pointer hover:bg-slate-50" @click="$router.push({ name: 'swarm-node', params: { id: node.nodeID } })">
              <i class="fas fa-server text-slate-400 text-xs w-3"></i>
              <span class="text-xs font-mono text-slate-700 flex-1 truncate">{{ node.nodeName }}</span>
              <span class="text-xs">
                <span class="text-emerald-600 font-medium">{{ node.running }}</span>
                <span class="text-slate-400"> / {{ node.total }} 任务</span>
              </span>
              <span class="text-xs text-slate-500">{{ node.running > 0 ? '运行中' : '空闲' }}</span>
            </div>
          </div>
        </div>

        <!-- Labels -->
        <div v-if="serviceData.labels && Object.keys(serviceData.labels).length > 0">
          <h2 class="section-title">Labels</h2>
          <div class="border border-slate-200 rounded-lg divide-y divide-slate-100">
            <div v-for="(val, key) in serviceData.labels" :key="key" class="px-3 py-1.5 flex gap-2 flex-wrap">
              <code class="text-xs font-mono text-blue-600 shrink-0">{{ key }}</code>
              <span class="text-slate-400">=</span>
              <code class="text-xs font-mono text-slate-600 break-all">{{ val }}</code>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
