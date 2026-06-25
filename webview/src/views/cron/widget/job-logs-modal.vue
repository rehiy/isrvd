<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { CronJob, CronJobLog } from '@/service/types'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    components: { BaseModal }
})
class JobLogsModal extends Vue {
    portal = usePortal()

    isOpen = false
    loading = false
    job: CronJob | null = null
    logs: CronJobLog[] = []

    get successCount() {
        return this.logs.filter(item => item.success).length
    }

    get failedCount() {
        return this.logs.length - this.successCount
    }

    async show(job: CronJob) {
        this.job = job
        this.logs = []
        this.isOpen = true
        await this.loadLogs()
    }

    async loadLogs() {
        if (!this.job) return
        this.loading = true
        try {
            const res = await api.cronJobLogs(this.job.id, 50)
            this.logs = res.payload?.logs || []
        } finally {
            this.loading = false
        }
    }

    formatTime(t?: string): string {
        if (!t) return '-'
        return new Date(t).toLocaleString('zh-CN')
    }

    formatDuration(ms: number): string {
        return ms < 1000 ? `${ms}ms` : `${(ms / 1000).toFixed(2)}s`
    }
}

export default toNative(JobLogsModal)
</script>

<template>
  <BaseModal v-model="isOpen" :show-footer="false" body-class="p-0 overflow-y-auto bg-slate-50">
    <template #title>
      <div class="flex items-center gap-3 min-w-0">
        <div class="page-icon bg-blue-50 text-blue-600">
          <i class="fas fa-list-ul text-sm"></i>
        </div>
        <div class="min-w-0">
          <h1 class="text-lg font-semibold text-slate-800 truncate">执行历史</h1>
          <p class="text-xs text-slate-400 truncate">{{ job?.name }}</p>
        </div>
      </div>
    </template>

    <template #header-actions>
      <button class="btn-icon-sm" :disabled="loading" title="刷新" @click="loadLogs">
        <i class="fas fa-rotate text-sm" :class="{ 'fa-spin': loading }"></i>
      </button>
    </template>

    <div class="p-4 md:p-5">
      <div v-if="loading" class="empty-state">
        <div class="w-10 h-10 spinner mb-3"></div>
        <p class="text-sm text-slate-500">加载中...</p>
      </div>

      <div v-else>
        <div class="rounded-xl px-4 py-3 mb-4">
          <div class="flex flex-wrap items-center gap-x-4 gap-y-2 text-xs text-slate-500">
            <span class="font-medium text-slate-700">{{ job?.name }}</span>
            <span>执行计划 <code class="font-mono">{{ job?.schedule }}</code></span>
            <span>{{ job?.type }}</span>
            <span>最近 {{ logs.length }} 次</span>
            <span class="text-emerald-600">成功 {{ successCount }}</span>
            <span class="text-red-600">失败 {{ failedCount }}</span>
          </div>
        </div>

        <div v-if="logs.length === 0" class="empty-state">
          <div class="empty-state-icon bg-white!">
            <i class="fas fa-file-lines text-2xl text-slate-300"></i>
          </div>
          <p class="text-slate-500 text-sm">暂无执行记录</p>
        </div>

        <div v-else class="divide-y divide-slate-200">
          <div v-for="(log, idx) in logs" :key="log.runId || idx" class="py-3 first:pt-0 last:pb-0">
            <div class="flex flex-col md:flex-row md:items-center md:justify-between gap-2 mb-2">
              <span class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-lg text-xs font-medium w-fit" :class="log.success ? 'bg-emerald-50 text-emerald-600' : 'bg-red-50 text-red-600'">
                <i :class="log.success ? 'fas fa-circle-check' : 'fas fa-circle-xmark'"></i>
                {{ log.success ? '成功' : '失败' }}
              </span>
              <div class="flex items-center gap-3 text-xs text-slate-400">
                <span><i class="far fa-clock mr-1"></i>{{ formatDuration(log.duration) }}</span>
                <span>{{ formatTime(log.startTime) }}</span>
              </div>
            </div>

            <pre v-if="log.output" class="bg-slate-900 text-slate-100 rounded-lg p-3 text-xs font-mono overflow-auto whitespace-pre-wrap break-all max-h-48">{{ log.output }}</pre>
            <pre v-if="log.error" class="mt-2 bg-red-50 text-red-700 rounded-lg p-3 text-xs font-mono overflow-auto whitespace-pre-wrap break-all max-h-28">{{ log.error }}</pre>
            <div v-if="!log.output && !log.error" class="text-xs text-slate-400 rounded-lg bg-slate-100 p-3">
              无输出内容
            </div>
          </div>
        </div>
      </div>
    </div>
  </BaseModal>
</template>
