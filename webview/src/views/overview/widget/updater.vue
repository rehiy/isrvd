<script lang="ts">
import axios from 'axios'
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { DockerContainerCreate, SystemVersionInfo } from '@/service/types'

import BaseModal from '@/component/modal.vue'
import ToggleCard from '@/component/toggle-card.vue'

@Component({
    components: { BaseModal, ToggleCard }
})
class SystemUpdater extends Vue {
    portal = usePortal()

    version: SystemVersionInfo | null = null

    deploying = false
    upgrading = false
    updaterModalOpen = false
    updaterContainer = ''
    updaterAutoRemove = true
    selfContainerName = ''
    inDocker = false
    pollController: AbortController | null = null
    refreshTimer: ReturnType<typeof setTimeout> | null = null

    unmounted() {
        this.pollController?.abort()
        if (this.refreshTimer) clearTimeout(this.refreshTimer)
    }

    async mounted() {
        await Promise.all([this.loadSelfContainer(), this.loadVersionInfo()])
    }

    async loadVersionInfo() {
        if (!this.portal.hasPerm('GET /api/overview/version')) return
        try {
            const res = await api.overviewVersion()
            this.version = res.payload ?? null
        } catch {
            // 静默处理，不影响主流程
        }
    }

    async loadSelfContainer() {
        if (!this.portal.hasPerm('GET /api/docker/containers')) return
        try {
            const res = await api.dockerContainerList(true)
            const self = res.payload?.find(c => c.isSelf)
            if (self) {
                this.inDocker = true
                this.selfContainerName = self.name ?? ''
            }
        } catch {
            // 获取失败静默处理，不影响主流程
        }
    }

    openUpdaterModal() {
        if (this.deploying || this.upgrading) return
        this.updaterContainer = this.selfContainerName
        this.updaterAutoRemove = true
        this.updaterModalOpen = true
    }

    // Docker 容器升级：部署临时 docker-updater 容器
    async handleDeployUpdater() {
        if (this.deploying || this.upgrading) return

        const containerName = this.updaterContainer.trim()
        if (!containerName) {
            this.portal.showNotification('error', this.$t('请填写当前 Docker 容器名称'))
            return
        }

        const spec: DockerContainerCreate = {
            image: this.version?.updaterImage || 'rehiy/docker-updater:latest',
            name: `auto-update-${Date.now()}`,
            autoRemove: this.updaterAutoRemove,
            volumes: [
                { type: 'bind', hostPath: '/var/run/docker.sock', containerPath: '/var/run/docker.sock', readOnly: false },
            ],
            cmd: [containerName],
        }
        await this.startUpgrade(signal => api.dockerContainerCreate(spec, { timeout: 120_000, signal }))
    }

    async handleBinaryUpgrade() {
        await this.startUpgrade(signal => api.overviewUpgrade({ timeout: 10_000, signal }))
    }

    async startUpgrade(submit: (signal: AbortSignal) => Promise<unknown>) {
        if (this.deploying || this.upgrading || !this.version?.current) return
        const oldVersion = this.version.current
        this.deploying = true
        const controller = new AbortController()
        this.pollController = controller
        try {
            await submit(controller.signal)
        } catch (error) {
            if (controller.signal.aborted) return
            if (!this.isUpgradeDisconnect(error, true)) {
                this.finishUpgrade()
                return
            }
        }
        if (controller.signal.aborted) return
        this.updaterModalOpen = false
        this.deploying = false
        await this.waitForNewVersion(controller, oldVersion)
    }

    isUpgradeDisconnect(error: unknown, submitting = false) {
        return axios.isAxiosError<{ success?: boolean }>(error) && !axios.isCancel(error)
            && !(submitting && error.response?.status === 503 && error.response.data?.success === false)
            && (!error.response || [502, 503, 504, 524].includes(error.response.status))
    }

    finishUpgrade(message?: string, type: 'error' | 'success' = 'error') {
        this.deploying = false
        this.pollController = null
        if (message) this.portal.showNotification(type, message)
        if (type === 'success') {
            this.refreshTimer = setTimeout(() => window.location.reload(), 2000)
        } else {
            this.upgrading = false
        }
    }

    async waitForNewVersion(controller: AbortController, oldVersion: string) {
        const deadline = Date.now() + 300_000
        this.upgrading = true

        while (!controller.signal.aborted && Date.now() < deadline) {
            await new Promise(resolve => setTimeout(resolve, Math.min(3_000, deadline - Date.now())))
            if (controller.signal.aborted || Date.now() >= deadline) break
            try {
                const options = { params: { current: 1 }, timeout: Math.min(10_000, deadline - Date.now()), signal: controller.signal }
                const current = (await api.overviewVersion(options)).payload?.current
                if (controller.signal.aborted) return
                if (current && oldVersion && current.replace(/^v/, '') !== oldVersion.replace(/^v/, '')) {
                    this.finishUpgrade(this.$t('升级成功，即将刷新页面...'), 'success')
                    return
                }
            } catch (error) {
                if (controller.signal.aborted) return
                if (!this.isUpgradeDisconnect(error)) {
                    this.finishUpgrade()
                    return
                }
            }
        }
        if (!controller.signal.aborted) this.finishUpgrade(this.$t('升级超时，请手动检查服务状态'))
    }
}

export default toNative(SystemUpdater)
</script>

<template>
  <template v-if="version?.hasUpdate">
    <!-- 版本更新横幅 -->
    <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 mb-3 px-4 py-3 rounded-xl bg-emerald-50 border border-emerald-100">
      <!-- 左侧：图标 + 文案 -->
      <div class="title-group">
        <div class="row-icon bg-emerald-500 text-white">
          <i class="fas fa-arrow-up text-xs"></i>
        </div>
        <div class="min-w-0">
          <p class="text-xs font-semibold text-emerald-700">{{ $t('发现新版本') }}</p>
          <p class="flex items-center gap-1 text-xs text-slate-500 mt-0.5">
            <span class="text-slate-400 line-through">{{ version?.current }}</span>
            <i class="fas fa-arrow-right text-[9px] text-slate-400"></i>
            <span class="text-emerald-600 font-semibold">{{ version.latest }}</span>
          </p>
        </div>
      </div>

      <!-- 右侧：操作按钮（小屏垂直排列，大屏水平排列） -->
      <div class="flex flex-col xs:flex-row items-stretch xs:items-center gap-2 w-full sm:w-auto sm:flex-shrink-0">
        <a
          :href="version.release"
          target="_blank"
          rel="noopener noreferrer"
          class="btn btn-secondary w-full xs:w-auto"
          :title="$t('查看更新日志')"
        >
          <i class="fas fa-file-alt"></i>
          <span>{{ $t('更新日志') }}</span>
        </a>
        <!-- Docker 容器升级（仅 Docker 环境） -->
        <button
          v-if="inDocker && portal.hasPerm('POST /api/docker/container')"
          class="btn btn-emerald w-full xs:w-auto"
          :title="$t('升级当前容器')"
          :disabled="deploying || upgrading"
          @click="openUpdaterModal"
        >
          <i class="fas fa-rotate-right" :class="{ 'fa-spin': deploying || upgrading }"></i>
          <span>{{ $t('升级容器') }}</span>
        </button>
        <!-- 二进制原地升级 -->
        <button
          v-if="portal.hasPerm('POST /api/overview/upgrade')"
          class="btn btn-primary w-full xs:w-auto"
          :title="$t('下载最新版本并重启')"
          :disabled="deploying || upgrading"
          @click="handleBinaryUpgrade"
        >
          <i class="fas fa-rotate-right" :class="{ 'fa-spin': deploying || upgrading }"></i>
          <span>{{ $t('升级二进制') }}</span>
        </button>
      </div>
    </div>

    <!-- Docker 升级确认 Modal -->
    <BaseModal
      v-model="updaterModalOpen"
      :title="$t('升级当前 Docker 容器')"
      :loading="deploying || upgrading"
      :confirm-disabled="!updaterContainer.trim()"
      confirm-class="btn-primary"
      @confirm="handleDeployUpdater"
    >
      <form class="space-y-4" @submit.prevent="handleDeployUpdater">
        <div>
          <label for="updaterContainer" class="form-label">{{ $t('当前容器名称') }}</label>
          <input
            id="updaterContainer"
            v-model="updaterContainer"
            type="text"
            :disabled="deploying"
            :readonly="!!selfContainerName"
            class="input"
            :placeholder="$t('请输入容器名称')"
            autocomplete="off"
            required
          />
        </div>
        <ToggleCard v-model="updaterAutoRemove" :label="$t('升级完成后自动销毁容器')" :desc="$t('关闭后可通过容器日志查看升级过程')" />
        <p class="text-sm text-slate-600">
          {{ $t('将通过') }} <code class="px-1 py-0.5 rounded bg-slate-100 font-mono text-xs">{{ version?.updaterImage || 'rehiy/docker-updater:latest' }}</code>
          {{ $t('临时容器拉取最新镜像并重启，升级期间服务会短暂中断。') }}
        </p>
      </form>

      <template #confirm-text>
        {{ $t('部署并升级') }}
      </template>
    </BaseModal>
  </template>
</template>
