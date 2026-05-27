<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { DockerContainerInfo } from '@/service/types'
import { COMPOSE_PROJECT_LABEL, COMPOSE_SERVICE_LABEL } from '@/service/types/docker'

import BaseModal from '@/component/modal.vue'

import ComposeEditor from '@/views/compose/widget/compose-editor.vue'

@Component({
    expose: ['show'],
    components: { BaseModal, ComposeEditor },
    emits: ['success']
})
class ContainerEditModal extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    isOpen = false
    modalLoading = false
    projectName = ''
    displayName = ''
    composeContent = ''

    get modalTitle() {
        return this.displayName ? `编辑配置：${this.displayName}` : '编辑容器配置'
    }

    // ─── 方法 ───
    async show(container: DockerContainerInfo) {
        const composeProject = container.labels?.[COMPOSE_PROJECT_LABEL]
        const composeService = container.labels?.[COMPOSE_SERVICE_LABEL]
        this.projectName = composeProject || container.name || container.id
        this.displayName = composeProject
            ? `${composeProject} / ${composeService || container.name}`
            : (container.name || container.id)
        this.composeContent = ''
        this.modalLoading = true
        this.isOpen = true
        try {
            const res = await api.dockerContainerCompose(this.projectName)
            this.composeContent = res.payload?.content || ''
        } catch {
            this.isOpen = false
        } finally {
            this.modalLoading = false
        }
    }

    async handleConfirm() {
        if (!this.composeContent.trim()) return
        this.modalLoading = true
        try {
            await api.composeDockerRedeploy(this.projectName, { content: this.composeContent })
            this.portal.showNotification('success', 'Compose 配置更新成功，已重建关联容器')
            this.isOpen = false
            this.$emit('success')
        } catch {}
        this.modalLoading = false
    }
}

export default toNative(ContainerEditModal)
</script>

<template>
  <BaseModal v-model="isOpen" :title="modalTitle" :loading="modalLoading" confirm-class="btn-emerald" show-footer @confirm="handleConfirm">
    <ComposeEditor v-model="composeContent" warning="更新配置后将会按 Compose 项目重建关联容器，旧容器将被停止并删除" />
    <template #confirm-text>更新并重建</template>
  </BaseModal>
</template>
