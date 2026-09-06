<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import type { DockerContainerInfo } from '@/service/types'
import { COMPOSE_PROJECT_LABEL, COMPOSE_SERVICE_LABEL } from '@/service/types/docker'

import RedeployModal from '@/views/compose/widget/redeploy-modal.vue'

@Component({
    expose: ['show'],
    components: { RedeployModal },
    emits: ['success']
})
class ContainerEditModal extends Vue {
    @Ref readonly modalRef!: InstanceType<typeof RedeployModal>

    projectName = ''
    displayName = ''

    get modalTitle() {
        return this.displayName ? `编辑配置：${this.displayName}` : '编辑容器配置'
    }

    async show(container: DockerContainerInfo) {
        const composeProject = container.labels?.[COMPOSE_PROJECT_LABEL]
        const composeService = container.labels?.[COMPOSE_SERVICE_LABEL]
        this.projectName = composeProject || container.name || container.id
        this.displayName = composeProject
            ? `${composeProject} / ${composeService || container.name}`
            : (container.name || container.id)
        await this.$nextTick()
        await this.modalRef.show()
    }
}

export default toNative(ContainerEditModal)
</script>

<template>
  <RedeployModal
    ref="modalRef"
    target="docker"
    :resource-name="projectName"
    :title="modalTitle"
    warning="更新配置后将会按 Compose 项目重建关联容器，旧容器将被停止并删除"
    refresh-title="跳过 compose.yml，按当前容器运行态重新反推 Compose"
    success-message="Compose 配置更新成功，已重建关联容器"
    @success="$emit('success')"
  />
</template>
