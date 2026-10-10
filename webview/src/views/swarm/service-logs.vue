<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'

import { LogStream } from '@/helper/log'

@Component
class ServiceLogs extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    serviceName = ''
    logs = new LogStream()

    get serviceId() {
        return this.$route.params.id as string
    }

    // ─── 方法 ───
    activeTab() {
        return this.$route.name
    }

    switchTab(name: string) {
        this.logs.stop()
        this.$router.push({ name, params: { id: this.serviceId } })
    }

    async loadLogs() {
        await this.logs.load(async () => (await api.swarmServiceLogs(this.serviceId, this.logs.tail)).payload?.logs || [])
    }

    startStream() {
        this.logs.start(
            `swarm/service/${encodeURIComponent(this.serviceId)}/logs/stream`,
            this.portal.token ?? '',
            message => this.portal.showNotification('error', message)
        )
    }

    handleTailChange() {
        if (this.logs.active) {
            this.logs.stop()
            this.startStream()
            return
        }
        this.loadLogs()
    }

    async loadServiceName() {
        try {
            const res = await api.swarmServiceInspect(this.serviceId)
            this.serviceName = res.payload?.name || ''
        } catch { /* 忽略，名称仅用于展示 */ }
    }

    // ─── 生命周期 ───
    mounted() {
        this.loadServiceName()
        this.loadLogs()
    }

    unmounted() {
        this.logs.stop()
    }
}

export default toNative(ServiceLogs)
</script>

<template>
  <div class="page">
    <!-- Toolbar -->
    <div class="page-toolbar">
      <!-- 桌面端 -->
      <div class="toolbar-desktop">
        <div class="flex items-center gap-3">
          <div class="page-icon bg-emerald-500">
            <i class="fas fa-cubes text-white"></i>
          </div>
          <div>
            <h1 class="text-lg font-semibold text-slate-800">{{ serviceName || $t('服务日志') }}</h1>
            <p class="text-xs text-slate-600 font-mono truncate max-w-xs">{{ serviceId }}</p>
          </div>
        </div>
        <div class="flex items-center gap-2">
          <div class="tab-group">
            <button v-if="portal.hasPerm('GET /api/swarm/service/:id')" :class="['tab-btn', activeTab() === 'swarm-service' ? 'tab-btn-active text-emerald-600' : 'tab-btn-inactive']" @click="switchTab('swarm-service')">
              <i class="fas fa-circle-info"></i><span>{{ $t('详情') }}</span>
            </button>
            <button v-if="portal.hasPerm('GET /api/swarm/service/:id/logs')" :class="['tab-btn', activeTab() === 'swarm-service-logs' ? 'tab-btn-active text-emerald-600' : 'tab-btn-inactive']" @click="switchTab('swarm-service-logs')">
              <i class="fas fa-file-lines"></i><span>{{ $t('日志') }}</span>
            </button>
          </div>
          <select v-model="logs.tail" class="w-28 select-sm" @change="handleTailChange">
            <option value="50">{{ $t('显示 50 行') }}</option>
            <option value="100">{{ $t('显示 100 行') }}</option>
            <option value="200">{{ $t('显示 200 行') }}</option>
            <option value="500">{{ $t('显示 500 行') }}</option>
            <option value="1000">{{ $t('显示 1000 行') }}</option>
          </select>
          <button class="btn btn-secondary" :disabled="logs.active" @click="loadLogs()">
            <i class="fas fa-rotate"></i>{{ $t('刷新') }}
          </button>
          <button v-if="!logs.active && portal.hasPerm('GET /api/swarm/service/:id/logs/stream')" class="btn btn-emerald" @click="startStream">
            <i class="fas fa-play"></i>{{ $t('实时') }}
          </button>
          <button v-else-if="logs.active" class="btn btn-secondary" @click="logs.stop()">
            <i :class="logs.state === 'connecting' ? 'fas fa-spinner fa-spin' : 'fas fa-stop'"></i>{{ $t('停止') }}
          </button>
        </div>
      </div>
      <!-- 移动端 -->
      <div class="block md:hidden">
        <div class="flex items-center justify-between mb-3">
          <div class="title-group">
            <div class="page-icon bg-emerald-500">
              <i class="fas fa-cubes text-white"></i>
            </div>
            <div class="min-w-0">
              <h1 class="title-text">{{ serviceName || $t('服务日志') }}</h1>
              <p class="text-xs text-slate-600 font-mono truncate">{{ serviceId.slice(0, 12) }}</p>
            </div>
          </div>
          <div class="action-group-sm">
            <select v-model="logs.tail" class="select-sm" @change="handleTailChange">
              <option value="50">{{ $t('50 行') }}</option>
              <option value="100">{{ $t('100 行') }}</option>
              <option value="200">{{ $t('200 行') }}</option>
              <option value="500">{{ $t('500 行') }}</option>
              <option value="1000">{{ $t('1000 行') }}</option>
            </select>
            <button class="btn btn-secondary btn-square" :title="$t('刷新')" :disabled="logs.active" @click="loadLogs()">
              <i class="fas fa-rotate text-sm"></i>
            </button>
            <button v-if="!logs.active && portal.hasPerm('GET /api/swarm/service/:id/logs/stream')" class="btn btn-emerald btn-square" :title="$t('实时')" @click="startStream">
              <i class="fas fa-play text-sm"></i>
            </button>
            <button v-else-if="logs.active" class="btn btn-secondary btn-square" :title="$t('停止')" @click="logs.stop()">
              <i :class="logs.state === 'connecting' ? 'fas fa-spinner fa-spin text-sm' : 'fas fa-stop text-sm'"></i>
            </button>
          </div>
        </div>
        <div class="tab-group">
          <button v-if="portal.hasPerm('GET /api/swarm/service/:id')" :class="['tab-btn', activeTab() === 'swarm-service' ? 'tab-btn-active text-emerald-600' : 'tab-btn-inactive']" @click="switchTab('swarm-service')">
            <i class="fas fa-circle-info"></i><span>{{ $t('详情') }}</span>
          </button>
          <button v-if="portal.hasPerm('GET /api/swarm/service/:id/logs')" :class="['tab-btn', activeTab() === 'swarm-service-logs' ? 'tab-btn-active text-emerald-600' : 'tab-btn-inactive']" @click="switchTab('swarm-service-logs')">
            <i class="fas fa-file-lines"></i><span>{{ $t('日志') }}</span>
          </button>
        </div>
      </div>
    </div>

    <!-- 内容区域 -->
    <div class="p-4">
      <div v-if="logs.loading" class="empty-state">
        <div class="spinner-lg"></div>
        <p class="text-slate-500">{{ $t('加载中...') }}</p>
      </div>
      <pre v-else-if="logs.content || logs.active" class="min-h-[18rem] bg-white text-xs font-mono leading-relaxed text-slate-700 whitespace-pre-wrap break-all">{{ logs.content || $t('等待日志输出...') }}</pre>
      <div v-else class="empty-state">
        <div class="empty-state-icon">
          <i class="fas fa-file-lines text-2xl text-slate-300"></i>
        </div>
        <p class="text-slate-500 text-sm">{{ $t('暂无日志') }}</p>
      </div>
    </div>
  </div>
</template>
