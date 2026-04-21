<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { ContainerInfo } from '@/service/types'
import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

import BaseModal from '@/component/modal.vue'
import ComposeEditor from '@/views/compose/widget/compose-editor.vue'

@Component({
    expose: ['show'],
    components: { BaseModal, ComposeEditor },
    emits: ['success']
})
class ContainerEditModal extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    // ─── 数据属性 ───
    isOpen = false
    modalLoading = false
    projectName = ''
    composeContent = ''

    // ─── 方法 ───
    async show(container: ContainerInfo) {
        this.projectName = container.name
        this.composeContent = ''
        this.modalLoading = true
        this.isOpen = true
        try {
            const res = await api.getContainerCompose(container.name)
            this.composeContent = res.payload?.content || ''
        } catch (e) {
            this.isOpen = false
        } finally {
            this.modalLoading = false
        }
    }

    async handleConfirm() {
        if (!this.composeContent.trim()) return
        this.modalLoading = true
        try {
            await api.composeRedeployDocker(this.projectName, { content: this.composeContent })
            this.actions.showNotification('success', '容器配置更新成功，已重建容器')
            this.isOpen = false
            this.$emit('success')
        } catch (e) {}
        this.modalLoading = false
    }
}

export default toNative(ContainerEditModal)
</script>

<template>
  <BaseModal
    v-model="isOpen"
    title="编辑容器配置"
    :loading="modalLoading"
    show-footer
    @confirm="handleConfirm"
  >
    <template #confirm-text>更新并重建</template>
    <ComposeEditor
      v-model="composeContent"
      warning="更新配置后将会重建容器，旧容器将被停止并删除"
    />
  </BaseModal>
</template>
