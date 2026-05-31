<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { SwarmServiceInfo } from '@/service/types'

import BaseModal from '@/component/modal.vue'

import ComposeEditor from '@/views/compose/widget/compose-editor.vue'

@Component({
    expose: ['show'],
    components: { BaseModal, ComposeEditor },
    emits: ['success']
})
class ServiceEditModal extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    isOpen = false
    modalLoading = false
    composeContent = ''
    serviceName = ''

    // ─── 方法 ───
    async show(svc: SwarmServiceInfo) {
        this.serviceName = svc.name
        this.composeContent = ''
        this.modalLoading = true
        this.isOpen = true
        try {
            const res = await api.swarmCompose(svc.name)
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
            await api.composeSwarmRedeploy(this.serviceName, { content: this.composeContent })
            this.portal.showNotification('success', 'Swarm 服务配置更新成功，已重建服务')
            this.isOpen = false
            this.$emit('success')
        } catch {}
        this.modalLoading = false
    }
}

export default toNative(ServiceEditModal)
</script>

<template>
  <BaseModal v-model="isOpen" :title="`编辑服务：${serviceName}`" :loading="modalLoading" confirm-class="btn-emerald" show-footer @confirm="handleConfirm">
    <ComposeEditor v-model="composeContent" warning="更新配置后将会删除旧服务并重新创建，期间服务短暂不可用" />
    <template #confirm-text>更新并重建</template>
  </BaseModal>
</template>
