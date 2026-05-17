<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { DockerContainerInfo } from '@/service/types'

import { usePortal } from '@/stores'

import ContainerNav from './widget/container-nav.vue'

@Component({
    components: { ContainerNav }
})
class ContainerLogs extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    container: DockerContainerInfo | null = null
    logLoading = false
    logContent = ''
    logTail = '100'

    get containerId() {
        return this.$route.params.id as string
    }

    // ─── 方法 ───
    onContainerLoaded(ct: DockerContainerInfo) {
        this.container = ct
        this.loadLogs()
    }

    async loadLogs() {
        if (!this.container) return
        this.logLoading = true
        this.logContent = ''
        try {
            const res = await api.dockerContainerLogs(this.containerId, this.logTail)
            this.logContent = (res.payload?.logs || []).join('')
        } catch {
            this.logContent = '加载日志失败'
        }
        this.logLoading = false
    }
}

export default toNative(ContainerLogs)
</script>

<template>
  <div>
    <div class="card mb-4">
      <ContainerNav :container-id="containerId" @loaded="onContainerLoaded" />
      <!-- 内容区域 -->
      <div class="p-4 md:p-6">
        <div class="flex items-center justify-between gap-3 mb-4">
          <div class="flex items-center gap-3">
            <label class="text-sm text-slate-600">显示行数</label>
            <select v-model="logTail" class="w-28 select-sm">
              <option value="50">50 行</option>
              <option value="100">100 行</option>
              <option value="200">200 行</option>
              <option value="500">500 行</option>
              <option value="1000">1000 行</option>
            </select>
          </div>
          <button class="btn btn-sm btn-secondary" @click="loadLogs">
            <i class="fas fa-rotate"></i>刷新
          </button>
        </div>
        <div v-if="logLoading" class="loading-state">
          <div class="w-12 h-12 spinner mb-3"></div>
          <p class="text-slate-500">加载中...</p>
        </div>
        <pre v-else-if="logContent" class="bg-slate-900 text-slate-100 rounded-xl p-3 md:p-4 text-xs font-mono overflow-auto max-h-[600px] whitespace-pre-wrap break-all">
          {{ logContent }}
        </pre>
        <div v-else class="empty-state">
          <div class="empty-state-icon">
            <i class="fas fa-file-lines text-2xl text-slate-300"></i>
          </div>
          <p class="text-slate-500 text-sm">暂无日志</p>
        </div>
      </div>
    </div>
  </div>
</template>
