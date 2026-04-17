<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import { formatTime } from '@/helper/utils'
import api from '@/service/api'
import type { SwarmServiceInspect } from '@/service/types'
import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

@Component
class ServiceInfo extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    // ─── 数据属性 ───
    serviceData: SwarmServiceInspect | null = null
    loading = false
    formatTime = formatTime

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

    async loadDetail() {
        this.loading = true
        try {
            const res = await api.swarmInspectService(this.serviceId)
            this.serviceData = res.payload ?? null
        } catch (e) {
            this.actions.showNotification('error', '获取服务详情失败')
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
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-4 md:px-6 py-3">
        <!-- 桌面端 -->
        <div class="hidden md:flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-lg bg-emerald-500 flex items-center justify-center">
              <i class="fas fa-cubes text-white"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">{{ serviceData?.name || '服务详情' }}</h1>
              <p class="text-xs text-slate-500 font-mono truncate max-w-xs">{{ serviceId }}</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <div class="flex gap-1 bg-slate-100 p-1 rounded-lg">
              <button @click="switchTab('swarm-service-info')" :class="['px-3 py-1 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5', activeTab() === 'swarm-service-info' ? 'bg-white text-emerald-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']">
                <i class="fas fa-circle-info"></i><span>详情</span>
              </button>
              <button @click="switchTab('swarm-service-logs')" :class="['px-3 py-1 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5', activeTab() === 'swarm-service-logs' ? 'bg-white text-emerald-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']">
                <i class="fas fa-file-lines"></i><span>日志</span>
              </button>
            </div>
            <button @click="loadDetail()" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-rotate"></i>刷新
            </button>
          </div>
        </div>
        <!-- 移动端 -->
        <div class="block md:hidden">
          <div class="flex items-center justify-between mb-3">
            <div class="flex items-center gap-3 min-w-0 flex-1">
              <div class="w-9 h-9 rounded-lg bg-emerald-500 flex items-center justify-center flex-shrink-0">
                <i class="fas fa-cubes text-white"></i>
              </div>
              <div class="min-w-0">
                <h1 class="text-lg font-semibold text-slate-800 truncate">{{ serviceData?.name || '服务详情' }}</h1>
                <p class="text-xs text-slate-500 font-mono truncate">{{ serviceId.slice(0, 12) }}</p>
              </div>
            </div>
            <button @click="loadDetail()" class="w-9 h-9 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 flex items-center justify-center text-slate-600 transition-colors flex-shrink-0" title="刷新">
              <i class="fas fa-rotate text-sm"></i>
            </button>
          </div>
          <div class="flex justify-center gap-1 bg-slate-100 p-1 rounded-lg">
            <button @click="switchTab('swarm-service-info')" :class="['px-3 py-1 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5', activeTab() === 'swarm-service-info' ? 'bg-white text-emerald-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']">
              <i class="fas fa-circle-info"></i><span class="hidden sm:inline">详情</span>
            </button>
            <button @click="switchTab('swarm-service-logs')" :class="['px-3 py-1 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5', activeTab() === 'swarm-service-logs' ? 'bg-white text-emerald-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']">
              <i class="fas fa-file-lines"></i><span class="hidden sm:inline">日志</span>
            </button>
          </div>
        </div>
      </div>

      <!-- Loading -->
      <div v-if="loading" class="flex flex-col items-center justify-center py-20">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>

      <!-- 未找到 -->
      <div v-else-if="!serviceData" class="flex flex-col items-center justify-center py-20">
        <div class="w-20 h-20 rounded-full bg-slate-100 flex items-center justify-center mb-4">
          <i class="fas fa-cubes text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium">未找到服务详情</p>
      </div>

      <!-- 内容 -->
      <div v-else class="p-4 md:p-6 space-y-6 text-sm">
        <!-- 基本信息 -->
        <div>
          <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">基本信息</h2>
          <div class="grid grid-cols-2 gap-3">
            <div class="col-span-2">
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">服务 ID</label>
              <code class="block px-3 py-2 bg-slate-50 rounded-lg text-xs font-mono text-slate-700 break-all">{{ serviceData.id }}</code>
            </div>
            <div class="col-span-2">
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">镜像</label>
              <code class="block px-3 py-2 bg-slate-50 rounded-lg text-xs font-mono text-slate-700 break-all">{{ serviceData.image }}</code>
            </div>
            <div>
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">模式</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg">
                <span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-slate-100 text-slate-600 capitalize">{{ serviceData.mode }}</span>
              </div>
            </div>
            <div>
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">副本</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700">
                <span class="text-emerald-600 font-medium">{{ serviceData.runningTasks }}</span>
                <span v-if="serviceData.mode === 'replicated'" class="text-slate-400"> / {{ serviceData.replicas ?? '?' }}</span>
                <span v-else class="text-slate-400"> 运行中</span>
              </div>
            </div>
            <div>
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">创建时间</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700">{{ formatTime(serviceData.createdAt) }}</div>
            </div>
            <div>
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">更新时间</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700">{{ formatTime(serviceData.updatedAt) }}</div>
            </div>
          </div>
        </div>

        <!-- 端口 -->
        <div v-if="serviceData.ports && serviceData.ports.length > 0">
          <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">端口映射</h2>
          <div class="border border-slate-200 rounded-lg divide-y divide-slate-100">
            <div v-for="(p, idx) in serviceData.ports" :key="idx" class="px-3 py-2 flex items-center gap-3">
              <code class="text-xs font-mono text-emerald-700 font-medium">{{ p.publishedPort }}</code>
              <i class="fas fa-arrow-right text-slate-300 text-xs"></i>
              <code class="text-xs font-mono text-slate-600">{{ p.targetPort }}/{{ p.protocol }}</code>
              <span class="ml-auto text-xs text-slate-400 capitalize">{{ p.publishMode }}</span>
            </div>
          </div>
        </div>

        <!-- 网络 -->
        <div v-if="serviceData.networks && serviceData.networks.length > 0">
          <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">网络</h2>
          <div class="flex flex-wrap gap-1.5">
            <span v-for="n in serviceData.networks" :key="n" class="inline-flex items-center px-2 py-1 rounded-lg text-xs font-medium bg-blue-50 text-blue-700">{{ n }}</span>
          </div>
        </div>

        <!-- 挂载 -->
        <div v-if="serviceData.mounts && serviceData.mounts.length > 0">
          <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">挂载</h2>
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
          <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">环境变量</h2>
          <div class="border border-slate-200 rounded-lg divide-y divide-slate-100">
            <div v-for="(env, idx) in serviceData.env" :key="idx" class="px-3 py-1.5">
              <code class="text-xs font-mono text-slate-600">{{ env }}</code>
            </div>
          </div>
        </div>

        <!-- 启动参数 -->
        <div v-if="serviceData.args && serviceData.args.length > 0">
          <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">启动参数</h2>
          <code class="block px-3 py-2 bg-slate-50 rounded-lg text-xs font-mono text-slate-700">{{ serviceData.args.join(' ') }}</code>
        </div>

        <!-- 约束 -->
        <div v-if="serviceData.constraints && serviceData.constraints.length > 0">
          <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">调度约束</h2>
          <div class="border border-slate-200 rounded-lg divide-y divide-slate-100">
            <div v-for="(c, idx) in serviceData.constraints" :key="idx" class="px-3 py-1.5">
              <code class="text-xs font-mono text-slate-600">{{ c }}</code>
            </div>
          </div>
        </div>

        <!-- Labels -->
        <div v-if="serviceData.labels && Object.keys(serviceData.labels).length > 0">
          <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">Labels</h2>
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
