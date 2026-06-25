<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { CronJob, CronTypeInfo } from '@/service/types'

import PageSearch from '@/component/page-search.vue'

import JobEditModal from './widget/job-edit-modal.vue'
import JobLogsModal from './widget/job-logs-modal.vue'

@Component({
    components: { PageSearch, JobEditModal, JobLogsModal }
})
class CronJobs extends Vue {
    portal = usePortal()

    @Ref readonly editModalRef!: InstanceType<typeof JobEditModal>
    @Ref readonly logsModalRef!: InstanceType<typeof JobLogsModal>

    jobs: CronJob[] = []
    loading = false
    searchText = ''
    types: CronTypeInfo[] = []

    get dockerAvailable() {
        return this.portal.serviceAvailability.docker
    }

    get filteredJobs() {
        const keyword = this.searchText.trim().toLowerCase()
        if (!keyword) return this.jobs
        return this.jobs.filter(j =>
            j.name.toLowerCase().includes(keyword) ||
            j.id.toLowerCase().includes(keyword) ||
            j.schedule.toLowerCase().includes(keyword) ||
            j.type.toLowerCase().includes(keyword) ||
            j.description.toLowerCase().includes(keyword) ||
            j.content.toLowerCase().includes(keyword) ||
            (j.workDir || '').toLowerCase().includes(keyword)
        )
    }

    formatTime(t?: string): string {
        if (!t) return '-'
        return new Date(t).toLocaleString('zh-CN')
    }

    runtimeStatusText(job: CronJob): string {
        if (job.runtimeStatus === 'scheduled') return '调度中'
        if (job.runtimeStatus === 'unregistered') return '未注册'
        return '已禁用'
    }

    runtimeStatusClass(job: CronJob): string {
        if (job.runtimeStatus === 'scheduled') return 'text-emerald-600 font-medium hover:text-emerald-700'
        if (job.runtimeStatus === 'unregistered') return 'text-amber-600 font-medium hover:text-amber-700'
        return 'text-slate-400 hover:text-slate-500'
    }

    isDockerJob(job: CronJob): boolean {
        return job.type === 'DOCKER_TMP' || job.type === 'DOCKER_CTR'
    }

    canOperateJob(job: CronJob): boolean {
        return !this.isDockerJob(job) || this.dockerAvailable
    }

    async loadTypes() {
        try {
            const res = await api.cronTypes()
            const types = res.payload?.types || []
            this.types = this.dockerAvailable
                ? types
                : types.filter(type => type.value !== 'DOCKER_TMP' && type.value !== 'DOCKER_CTR')
        } catch {
            this.types = []
            this.portal.showNotification('error', '获取可用脚本类型失败')
        }
    }

    async loadJobs() {
        this.loading = true
        try {
            const res = await api.cronJobList()
            this.jobs = res.payload?.jobs || []
        } catch {
            this.portal.showNotification('error', '获取计划任务失败')
        } finally {
            this.loading = false
        }
    }

    openCreate() {
        if (this.types.length === 0) {
            this.portal.showNotification('error', '暂无可用脚本类型')
            return
        }
        this.editModalRef?.show(null, this.types)
    }

    openEdit(job: CronJob) {
        this.editModalRef?.show(job, this.types)
    }

    openDelete(job: CronJob) {
        this.portal.showConfirm({
            title: '删除计划任务',
            message: `确定要删除任务 <strong class="text-slate-900">${job.name}</strong> 吗？此操作不可恢复。`,
            icon: 'fa-trash',
            iconColor: 'red',
            confirmText: '确认删除',
            danger: true,
            onConfirm: async () => {
                await api.cronJobDelete(job.id)
                this.portal.showNotification('success', '任务已删除')
                this.loadJobs()
            }
        })
    }

    async toggleEnabled(job: CronJob) {
        try {
            await api.cronJobStatusPatch(job.id, !job.enabled)
            this.portal.showNotification('success', job.enabled ? '任务已禁用' : '任务已启用')
            await this.loadJobs()
        } catch {
            this.portal.showNotification('error', '操作失败')
        }
    }

    async runNow(job: CronJob) {
        try {
            await api.cronJobRun(job.id)
            this.portal.showNotification('success', `任务 "${job.name}" 已触发`)
        } catch {
            this.portal.showNotification('error', '触发任务失败')
        }
    }

    openLogs(job: CronJob) {
        this.logsModalRef?.show(job)
    }

    mounted() {
        this.loadTypes()
        this.loadJobs()
    }
}

export default toNative(CronJobs)
</script>

<template>
  <div class="page">
    <div class="page-toolbar">
      <div class="hidden md:flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="page-icon bg-amber-500">
            <i class="fas fa-clock text-white"></i>
          </div>
          <div>
            <h1 class="text-lg font-semibold text-slate-800 truncate">计划任务</h1>
            <p class="text-xs text-slate-500">按设定时间或周期自动执行脚本命令</p>
          </div>
        </div>
        <div class="flex items-center gap-2 flex-shrink-0">
          <PageSearch v-model="searchText" search-key="cron-jobs" placeholder="搜索任务名称、执行计划..." focus-color="amber" type-to-search />
          <button class="btn btn-secondary" @click="loadJobs()">
            <i class="fas fa-rotate"></i>刷新
          </button>
          <button v-if="portal.hasPerm('POST /api/cron/jobs')" class="btn btn-amber" @click="openCreate()">
            <i class="fas fa-plus"></i>新建任务
          </button>
        </div>
      </div>

      <div class="block md:hidden">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3 min-w-0 flex-1">
            <div class="page-icon bg-amber-500">
              <i class="fas fa-clock text-white"></i>
            </div>
            <div class="min-w-0">
              <h1 class="text-lg font-semibold text-slate-800 truncate">计划任务</h1>
              <p class="text-xs text-slate-500 truncate">定时自动执行脚本</p>
            </div>
          </div>
          <div class="flex items-center gap-1 flex-shrink-0">
            <button class="btn btn-secondary w-9 h-9 !px-0" title="刷新" @click="loadJobs()">
              <i class="fas fa-rotate text-sm"></i>
            </button>
            <button v-if="portal.hasPerm('POST /api/cron/jobs')" class="btn btn-amber w-9 h-9 !px-0" title="新建任务" @click="openCreate()">
              <i class="fas fa-plus text-sm"></i>
            </button>
          </div>
        </div>
      </div>
    </div>
    <div class="mobile-search">
      <PageSearch v-model="searchText" search-key="cron-jobs" placeholder="搜索任务..." width-class="w-full" focus-color="amber" />
    </div>

    <div v-if="loading" class="card-body">
      <div class="empty-state">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>
    </div>

    <div v-else-if="filteredJobs.length === 0" class="card-body">
      <div class="empty-state">
        <div class="empty-state-icon">
          <i class="fas fa-clock text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">{{ jobs.length === 0 ? '暂无计划任务' : '未找到匹配任务' }}</p>
        <p class="text-sm text-slate-400">{{ jobs.length === 0 ? '点击「新建任务」创建第一个定时任务' : '尝试更换关键词或清空搜索条件' }}</p>
      </div>
    </div>

    <template v-else>
      <div class="card-table hidden md:block">
        <table class="w-full border-collapse">
          <thead>
            <tr class="bg-slate-50 border-b border-slate-200">
              <th class="th">任务名称</th>
              <th class="w-36 th">执行计划</th>
              <th class="w-20 th">类型</th>
              <th class="w-20 th">状态</th>
              <th class="w-36 th">下次执行</th>
              <th class="w-36 th">上次执行</th>
              <th class="w-36 th-right">操作</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <tr v-for="job in filteredJobs" :key="job.id" class="hover:bg-slate-50 transition-colors">
              <td class="px-4 py-3 max-w-[280px]">
                <div class="flex items-center gap-2 min-w-0">
                  <div class="row-icon bg-violet-400">
                    <i class="fas fa-clock text-white text-sm"></i>
                  </div>
                  <div class="min-w-0">
                    <span class="font-medium text-slate-800 truncate block">{{ job.name }}</span>
                    <span v-if="job.description" class="text-xs text-slate-400 truncate block mt-0.5">{{ job.description }}</span>
                  </div>
                </div>
              </td>
              <td class="px-4 py-3">
                <code class="text-xs text-slate-700 font-mono bg-slate-100 px-1.5 py-0.5 rounded">{{ job.schedule }}</code>
              </td>
              <td class="px-4 py-3">
                <span class="inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-medium font-mono bg-slate-100 text-slate-700">{{ job.type }}</span>
              </td>
              <td class="px-4 py-3">
                <button v-if="portal.hasPerm('PATCH /api/cron/jobs/:id') && (canOperateJob(job) || job.enabled)" :title="job.enabled ? '点击禁用' : '点击启用'" class="text-xs font-medium transition-colors" :class="runtimeStatusClass(job)" @click="toggleEnabled(job)">
                  {{ runtimeStatusText(job) }}
                </button>
                <span v-else class="text-xs" :class="runtimeStatusClass(job)">{{ runtimeStatusText(job) }}</span>
              </td>
              <td class="px-4 py-3 text-xs text-slate-600 whitespace-nowrap">{{ formatTime(job.nextRun) }}</td>
              <td class="px-4 py-3 text-xs text-slate-600 whitespace-nowrap">{{ formatTime(job.lastRun) }}</td>
              <td class="px-4 py-3">
                <div class="flex items-center justify-end gap-1.5">
                  <button v-if="portal.hasPerm('POST /api/cron/jobs/:id/run') && canOperateJob(job)" class="btn-icon btn-icon-emerald" title="立即执行" @click="runNow(job)">
                    <i class="fas fa-play text-xs"></i>
                  </button>
                  <button v-if="portal.hasPerm('GET /api/cron/jobs/:id/logs')" class="btn-icon btn-icon-slate" title="执行日志" @click="openLogs(job)">
                    <i class="fas fa-list-ul text-xs"></i>
                  </button>
                  <button v-if="portal.hasPerm('PUT /api/cron/jobs/:id') && canOperateJob(job)" class="btn-icon btn-icon-blue" title="编辑" @click="openEdit(job)">
                    <i class="fas fa-pen text-xs"></i>
                  </button>
                  <button v-if="portal.hasPerm('DELETE /api/cron/jobs/:id')" class="btn-icon btn-icon-red" title="删除" @click="openDelete(job)">
                    <i class="fas fa-trash text-xs"></i>
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <!-- 移动端卡片列表 -->
      <div class="card-body md:hidden space-y-3">
        <div v-for="job in filteredJobs" :key="job.id" class="card-interactive">
          <div class="flex items-start justify-between gap-3 mb-3">
            <div class="flex items-center gap-2 min-w-0 flex-1">
              <div class="list-icon bg-violet-400">
                <i class="fas fa-clock text-white text-base"></i>
              </div>
              <div class="min-w-0">
                <span class="font-medium text-slate-800 text-sm truncate block">{{ job.name }}</span>
                <span class="text-xs text-slate-400 font-mono truncate block mt-0.5">{{ job.id }}</span>
              </div>
            </div>
            <button v-if="portal.hasPerm('PATCH /api/cron/jobs/:id') && (canOperateJob(job) || job.enabled)" class="text-xs font-medium flex-shrink-0 transition-colors" :class="runtimeStatusClass(job)" @click="toggleEnabled(job)">
              {{ runtimeStatusText(job) }}
            </button>
            <span v-else class="text-xs font-medium flex-shrink-0" :class="runtimeStatusClass(job)">{{ runtimeStatusText(job) }}</span>
          </div>

          <div class="text-xs">
            <div class="card-prop-row-start">
              <span class="w-20 flex-shrink-0 text-slate-400">执行计划</span>
              <code class="min-w-0 text-slate-700 font-mono truncate">{{ job.schedule }}</code>
            </div>
            <div class="card-prop-row-start">
              <span class="w-20 flex-shrink-0 text-slate-400">类型</span>
              <span class="min-w-0 text-slate-600 font-mono truncate">{{ job.type }}</span>
            </div>
            <div class="card-prop-row-start">
              <span class="w-20 flex-shrink-0 text-slate-400">下次执行</span>
              <span class="min-w-0 text-slate-600 truncate">{{ formatTime(job.nextRun) }}</span>
            </div>
            <div class="card-prop-row-start">
              <span class="w-20 flex-shrink-0 text-slate-400">上次执行</span>
              <span class="min-w-0 text-slate-600 truncate">{{ formatTime(job.lastRun) }}</span>
            </div>
            <div v-if="job.description" class="card-prop-row-start">
              <span class="w-20 flex-shrink-0 text-slate-400">描述</span>
              <span class="min-w-0 text-slate-600 break-words">{{ job.description }}</span>
            </div>
          </div>

          <div class="flex items-center justify-end gap-1.5 pt-3 mt-3 border-t border-slate-100">
            <button v-if="portal.hasPerm('POST /api/cron/jobs/:id/run') && canOperateJob(job)" class="btn-icon btn-icon-emerald" title="立即执行" @click="runNow(job)"><i class="fas fa-play text-xs"></i></button>
            <button v-if="portal.hasPerm('GET /api/cron/jobs/:id/logs')" class="btn-icon btn-icon-slate" title="执行日志" @click="openLogs(job)"><i class="fas fa-list-ul text-xs"></i></button>
            <button v-if="portal.hasPerm('PUT /api/cron/jobs/:id') && canOperateJob(job)" class="btn-icon btn-icon-blue" title="编辑" @click="openEdit(job)"><i class="fas fa-pen text-xs"></i></button>
            <button v-if="portal.hasPerm('DELETE /api/cron/jobs/:id')" class="btn-icon btn-icon-red" title="删除" @click="openDelete(job)"><i class="fas fa-trash text-xs"></i></button>
          </div>
        </div>
      </div>
    </template>
  </div>

  <JobEditModal ref="editModalRef" @success="loadJobs()" />
  <JobLogsModal ref="logsModalRef" />
</template>
