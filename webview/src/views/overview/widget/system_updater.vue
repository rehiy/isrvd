<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { DockerContainerCreate } from '@/service/types'

import BaseModal from '@/component/modal.vue'

@Component({
    components: { BaseModal }
})
class SystemUpdater extends Vue {
    portal = usePortal()

    get versionCheck() { return this.portal.versionCheck }
    get currentVersion() { return this.portal.currentVersion }

    deploying = false
    upgrading = false
    updaterModalOpen = false
    updaterContainer = ''
    updaterAutoRemove = true
    selfContainerName = ''
    inDocker = false

    async mounted() {
        await this.loadSelfContainer()
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
        this.updaterContainer = this.selfContainerName
        this.updaterAutoRemove = true
        this.updaterModalOpen = true
    }

    // Docker 容器升级：部署临时 docker-updater 容器
    async handleDeployUpdater() {
        if (this.deploying) return

        const containerName = this.updaterContainer.trim()
        if (!containerName) {
            this.portal.showNotification('error', '请填写当前 Docker 容器名称')
            return
        }

        this.deploying = true
        try {
            const spec: DockerContainerCreate = {
                image: 'rehiy/docker-updater:latest',
                name: `auto-update-${Date.now()}`,
                autoRemove: this.updaterAutoRemove,
                volumes: [
                    { type: 'bind', hostPath: '/var/run/docker.sock', containerPath: '/var/run/docker.sock', readOnly: false },
                ],
                cmd: [containerName],
            }
            await api.dockerContainerCreate(spec)
            this.updaterModalOpen = false
            this.deploying = false
            await this.waitForNewVersion()
        } catch {
            // Axios 拦截器会显示错误通知
            this.deploying = false
        }
    }

    // 二进制原地升级：下载最新版本替换当前二进制并重启
    async handleBinaryUpgrade() {
        if (this.deploying) return
        this.deploying = true
        try {
            await api.overviewUpgrade()
            this.deploying = false
            await this.waitForNewVersion()
        } catch {
            // Axios 拦截器会显示错误通知
            this.deploying = false
        }
    }

    /** 升级触发后轮询版本号，直到版本变化或超时 */
    async waitForNewVersion() {
        const oldVersion = this.portal.currentVersion
        const maxWait = 300_000  // 最长等待 300 秒（5 分钟）
        const interval = 3_000   // 每 3 秒轮询一次
        const start = Date.now()

        this.upgrading = true
        let succeeded = false
        while (Date.now() - start < maxWait) {
            await new Promise(r => setTimeout(r, interval))
            try {
                const res = await api.overviewMonitor({ type: 'host', since: 0 })
                const newVersion = res.payload?.data?.version
                if (newVersion && newVersion !== oldVersion) {
                    succeeded = true
                    break
                }
            } catch {
                // 服务重启中，请求失败属正常，继续等待
            }
        }

        if (succeeded) {
            // 保持 upgrading = true，按钮继续显示"等待重启..."直到页面刷新
            this.portal.showNotification('success', `升级成功，即将刷新页面...`)
            setTimeout(() => window.location.reload(), 2000)
        } else {
            // 超时
            this.upgrading = false
            this.portal.showNotification('error', '升级超时，请手动检查服务状态')
        }
    }
}

export default toNative(SystemUpdater)
</script>

<template>
  <template v-if="versionCheck?.update">
    <!-- 版本更新横幅 -->
    <div class="flex items-center justify-between gap-3 px-4 py-3 rounded-xl bg-emerald-50 border border-emerald-100">
      <!-- 左侧：图标 + 文案 -->
      <div class="flex items-center gap-3 min-w-0">
        <div class="row-icon bg-emerald-500 text-white">
          <i class="fas fa-arrow-up text-xs"></i>
        </div>
        <div class="min-w-0">
          <p class="text-xs font-semibold text-emerald-700">发现新版本</p>
          <p class="flex items-center gap-1 text-xs text-slate-500 mt-0.5">
            <span class="text-slate-400 line-through">{{ currentVersion }}</span>
            <i class="fas fa-arrow-right text-[9px] text-slate-400"></i>
            <span class="text-emerald-600 font-semibold">{{ versionCheck.latest }}</span>
          </p>
        </div>
      </div>

      <!-- 右侧：操作按钮 -->
      <div class="flex items-center gap-2 flex-shrink-0">
        <a
          :href="versionCheck.release"
          target="_blank"
          rel="noopener noreferrer"
          class="btn btn-secondary"
          title="查看更新日志"
        >
          <i class="fas fa-file-alt"></i>
          <span class="hidden xs:inline">更新日志</span>
        </a>
        <!-- 二进制原地升级 -->
        <button
          v-if="portal.hasPerm('POST /api/overview/upgrade')"
          class="btn btn-primary"
          title="下载最新版本并重启"
          :disabled="deploying || upgrading"
          @click="handleBinaryUpgrade"
        >
          <i class="fas fa-rotate-right" :class="{ 'fa-spin': deploying || upgrading }"></i>
          <span class="hidden xs:inline">{{ deploying ? '升级中...' : upgrading ? '等待重启...' : '二进制升级' }}</span>
        </button>
        <!-- Docker 容器升级（仅 Docker 环境） -->
        <button
          v-if="inDocker && portal.hasPerm('POST /api/docker/container')"
          class="btn btn-emerald"
          title="升级当前容器"
          :disabled="deploying || upgrading"
          @click="openUpdaterModal"
        >
          <i class="fas fa-rotate-right" :class="{ 'fa-spin': deploying || upgrading }"></i>
          <span class="hidden xs:inline">{{ deploying ? '升级中...' : upgrading ? '等待重启...' : '容器升级' }}</span>
        </button>
      </div>
    </div>

    <!-- Docker 升级确认 Modal -->
    <BaseModal
      v-model="updaterModalOpen"
      title="升级当前 Docker 容器"
      :loading="deploying"
      :confirm-disabled="!updaterContainer.trim()"
      confirm-class="btn-primary"
      @confirm="handleDeployUpdater"
    >
      <form class="space-y-4" @submit.prevent="handleDeployUpdater">
        <div>
          <label for="updaterContainer" class="form-label">当前容器名称</label>
          <input
            id="updaterContainer"
            v-model="updaterContainer"
            type="text"
            :disabled="deploying"
            :readonly="!!selfContainerName"
            class="input"
            placeholder="例如：isrvd"
            autocomplete="off"
            required
          />
        </div>
        <div class="toggle-row">
          <div>
            <span class="text-sm text-slate-700">升级完成后自动销毁容器</span>
            <p class="text-xs text-slate-400 mt-0.5">关闭后可通过容器日志查看升级过程</p>
          </div>
          <button
            type="button"
            class="toggle"
            :class="{ 'toggle-on': updaterAutoRemove }"
            role="switch"
            :aria-checked="updaterAutoRemove"
            :disabled="deploying"
            @click="updaterAutoRemove = !updaterAutoRemove"
          >
            <span class="toggle-thumb" />
          </button>
        </div>
        <p class="text-sm text-slate-600">
          将通过 <code class="px-1 py-0.5 rounded bg-slate-100 font-mono text-xs">rehiy/docker-updater</code>
          临时容器拉取最新镜像并重启，升级期间服务会短暂中断。
        </p>
      </form>

      <template #confirm-text>
        {{ deploying ? '升级中...' : '部署并升级' }}
      </template>
    </BaseModal>
  </template>
</template>
