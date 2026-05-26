<script lang="ts">
import { Component, Prop, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { DockerContainerCreate, SystemVersionCheck } from '@/service/types'
import BaseModal from '@/component/modal.vue'

@Component({
    components: { BaseModal }
})
class SystemUpdater extends Vue {
    portal = usePortal()

    @Prop({ type: Object, default: null }) readonly versionCheck!: SystemVersionCheck | null
    @Prop({ type: String, default: '' }) readonly currentVersion!: string

    deploying = false
    updaterModalOpen = false
    updaterContainer = ''
    updaterAutoRemove = true
    selfContainerName = ''

    async mounted() {
        await this.loadSelfContainer()
    }

    async loadSelfContainer() {
        if (!this.portal.hasPerm('GET /api/docker/containers')) return
        try {
            const res = await api.dockerContainerList(true)
            const self = res.payload?.find(c => c.isSelf)
            this.selfContainerName = self?.name ?? ''
        } catch {
            // 获取失败静默处理，不影响主流程
        }
    }

    openUpdaterModal() {
        this.updaterContainer = this.selfContainerName
        this.updaterAutoRemove = true
        this.updaterModalOpen = true
    }

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
                name: `docker-updater-${Date.now()}`,
                autoRemove: this.updaterAutoRemove,
                volumes: [
                    { type: 'bind', hostPath: '/var/run/docker.sock', containerPath: '/var/run/docker.sock', readOnly: false },
                ],
                cmd: [containerName],
            }
            const res = await api.dockerContainerCreate(spec)
            this.portal.showNotification('success', `已部署临时 updater 容器 ${res.payload?.name || spec.name}，正在升级容器 ${containerName}`)
            this.updaterModalOpen = false
        } catch {
            // Axios 拦截器会显示错误通知
        } finally {
            this.deploying = false
        }
    }
}

export default toNative(SystemUpdater)
</script>

<template>
  <template v-if="versionCheck?.update">
    <!-- 版本更新横幅 -->
    <div class="flex items-center justify-between gap-3 px-4 py-3 rounded-xl bg-gradient-to-r from-emerald-50 to-teal-50 border border-emerald-200/70">
      <!-- 左侧：图标 + 文案 -->
      <div class="flex items-center gap-3 min-w-0">
        <div class="w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0 bg-emerald-500 text-white">
          <i class="fas fa-arrow-up text-xs"></i>
        </div>
        <div class="min-w-0">
          <p class="text-xs font-semibold text-emerald-700 leading-tight">发现新版本</p>
          <p class="flex items-center gap-1 text-xs text-slate-500 mt-0.5">
            <span class="text-slate-400 line-through">{{ currentVersion }}</span>
            <i class="fas fa-arrow-right text-[9px] text-emerald-400"></i>
            <span class="text-emerald-600 font-semibold">v{{ versionCheck.latest }}</span>
          </p>
        </div>
      </div>

      <!-- 右侧：操作按钮 -->
      <div class="flex items-center gap-2 flex-shrink-0">
        <a
          :href="versionCheck.release"
          target="_blank"
          rel="noopener noreferrer"
          class="inline-flex items-center gap-1.5 h-7 px-2.5 rounded-lg text-xs font-medium text-emerald-700 bg-white border border-emerald-200 hover:bg-emerald-50 hover:border-emerald-300 transition-colors"
          title="查看更新日志"
        >
          <i class="fas fa-file-alt text-[10px]"></i>
          <span class="hidden xs:inline">更新日志</span>
        </a>
        <button
          v-if="portal.hasPerm('POST /api/docker/container')"
          class="inline-flex items-center gap-1.5 h-7 px-2.5 rounded-lg text-xs font-medium text-white bg-emerald-500 hover:bg-emerald-600 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          title="一键升级当前容器"
          :disabled="deploying"
          @click="openUpdaterModal"
        >
          <i class="fas fa-rotate-right text-[10px]" :class="{ 'fa-spin': deploying }"></i>
          <span class="hidden xs:inline">{{ deploying ? '升级中...' : '一键升级' }}</span>
        </button>
      </div>
    </div>

    <!-- 升级确认 Modal -->
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
