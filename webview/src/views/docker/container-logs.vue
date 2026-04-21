<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { DockerContainerInfo } from '@/service/types'
import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

import ContainerNav from '@/views/docker/widget/container-nav.vue'

@Component({
    components: { ContainerNav }
})
class ContainerLogs extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

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
            const res = await api.containerLogs(this.containerId, this.logTail)
            this.logContent = (res.payload?.logs || []).join('')
        } catch (e) {
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
        <div class="flex flex-col md:flex-row md:items-center justify-between gap-3 mb-4">
          <div class="flex items-center gap-3">
            <label class="text-sm text-slate-600">显示行数</label>
            <select v-model="logTail" class="w-28 px-3 py-1.5 bg-white border border-slate-200 rounded-lg text-xs text-slate-700 hover:border-slate-300 focus:outline-none focus:border-slate-400">
              <option value="50">50 行</option>
              <option value="100">100 行</option>
              <option value="200">200 行</option>
              <option value="500">500 行</option>
              <option value="1000">1000 行</option>
            </select>
          </div>
          <button @click="loadLogs" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
            <i class="fas fa-rotate"></i>刷新
          </button>
        </div>
        <div v-if="logLoading" class="flex flex-col items-center justify-center py-12 gap-3 text-slate-400 text-sm">
          <div class="w-8 h-8 spinner"></div>
          <span>加载日志中...</span>
        </div>
        <pre v-else-if="logContent" class="bg-slate-900 text-slate-100 rounded-xl p-3 md:p-4 text-xs font-mono overflow-auto max-h-[600px] whitespace-pre-wrap break-all">{{ logContent }}</pre>
        <div v-else class="flex flex-col items-center justify-center py-16">
          <div class="w-16 h-16 rounded-full bg-slate-100 flex items-center justify-center mb-3">
            <i class="fas fa-file-lines text-2xl text-slate-300"></i>
          </div>
          <p class="text-slate-500 text-sm">暂无日志</p>
        </div>
      </div>
    </div>
  </div>
</template>
