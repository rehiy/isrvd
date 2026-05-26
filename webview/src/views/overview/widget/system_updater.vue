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

    deploying = false
    updaterModalOpen = false
    updaterContainer = ''

    openUpdaterModal() {
        this.updaterContainer = ''
        this.updaterModalOpen = true
    }

    async handleDeployUpdater() {
        if (this.deploying) {
            return
        }

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
                autoRemove: true,
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
  <div v-if="versionCheck?.update" class="flex items-center gap-2">
    <a
      :href="versionCheck.release"
      target="_blank"
      rel="noopener noreferrer"
      class="update-link !px-2 py-0.5 text-[10px]"
      :title="`发现新版本 ${versionCheck.latest}`"
    >
      <i class="fas fa-arrow-up text-[10px]"></i>
      <span class="hidden sm:inline">v{{ versionCheck.latest }}</span>
    </a>
    <button
      v-if="portal.hasPerm('POST /api/docker/container')"
      class="btn btn-cyan w-8 h-8 !px-0 !py-0 text-[12px]"
      title="部署 docker-updater 升级当前容器"
      :disabled="deploying"
      @click="openUpdaterModal"
    >
      <i class="fas fa-rotate-right"></i>
    </button>

    <BaseModal
      v-model="updaterModalOpen"
      title="升级当前 Docker 容器"
      :loading="deploying"
      :confirm-disabled="!updaterContainer.trim()"
      confirm-class="btn-secondary"
      @confirm="handleDeployUpdater"
    >
      <form class="space-y-5" @submit.prevent="handleDeployUpdater">
        <p class="text-sm text-slate-600">请输入要通过 docker-updater 升级的当前容器名称。</p>
        <div>
          <label for="updaterContainer" class="form-label">Docker 容器名称</label>
          <input
            id="updaterContainer"
            v-model="updaterContainer"
            type="text"
            :disabled="deploying"
            class="input"
            placeholder="例如：isrvd"
            required
          />
        </div>
      </form>

      <template #confirm-text>
        {{ deploying ? '升级中...' : '部署并升级' }}
      </template>
    </BaseModal>
  </div>
</template>
