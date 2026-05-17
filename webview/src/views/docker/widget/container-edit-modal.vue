<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { DockerContainerInfo } from '@/service/types'

import BaseModal from '@/component/modal.vue'

import ComposeEditor from '@/views/compose/widget/compose-editor.vue'

import { usePortal } from '@/stores'

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
    composeContent = ''

    // ─── 方法 ───
    async show(container: DockerContainerInfo) {
        this.projectName = container.name
        this.composeContent = ''
        this.modalLoading = true
        this.isOpen = true
        try {
            const res = await api.dockerContainerCompose(container.name)
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
            this.portal.showNotification('success', '容器配置更新成功，已重建容器')
            this.isOpen = false
            this.$emit('success')
        } catch {}
        this.modalLoading = false
    }
}

export default toNative(ContainerEditModal)
</script>

<template>
  <BaseModal v-model="isOpen" title="编辑容器配置" :loading="modalLoading" confirm-class="btn-emerald" show-footer @confirm="handleConfirm">
    <ComposeEditor v-model="composeContent" warning="更新配置后将会重建容器，旧容器将被停止并删除" />
    <template #confirm-text>更新并重建</template>
  </BaseModal>
</template>
