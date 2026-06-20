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
    refreshing = false
    composeContent = ''
    serviceName = ''
    composeFileModTime = 0
    composeSource: 'file' | 'runtime' | '' = ''

    get composeWarning() {
        const parts: string[] = ['更新配置后将会删除旧服务并重新创建，期间服务短暂不可用']
        if (this.composeSource === 'runtime') {
            parts.push('当前为运行态反推结果，建议核对后再提交')
        }
        if (this.composeFileModTime) {
            parts.push(`文件更新时间：${new Date(this.composeFileModTime * 1000).toLocaleString('zh-CN', { hour12: false })}`)
        }
        return parts.join('；')
    }

    // ─── 方法 ───
    async show(svc: SwarmServiceInfo) {
        this.serviceName = svc.name
        this.composeContent = ''
        this.composeFileModTime = 0
        this.composeSource = ''
        this.isOpen = true
        await this.loadCompose(false, true)
    }

    async loadCompose(force = false, closeOnError = false) {
        this.modalLoading = true
        if (force) this.refreshing = true
        try {
            const res = await api.composeSwarmInspect(this.serviceName, force)
            const payload = res.payload
            this.composeContent = payload?.content || ''
            this.composeFileModTime = payload?.fileModTime || 0
            this.composeSource = payload?.source || ''
            if (force) this.portal.showNotification('success', '已从运行态重新反推 Compose 配置')
        } catch {
            if (closeOnError) this.isOpen = false
        } finally {
            this.modalLoading = false
            this.refreshing = false
        }
    }

    async handleForceRefresh() {
        if (this.modalLoading || this.refreshing) return
        await this.loadCompose(true)
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
    <template #header-actions>
      <button type="button" class="btn-icon-sm" :disabled="modalLoading" title="跳过 compose.yml，按当前服务运行态重新反推 Compose" @click="handleForceRefresh()">
        <i :class="refreshing ? 'fas fa-spinner fa-spin' : 'fas fa-rotate'"></i>
      </button>
    </template>

    <ComposeEditor v-model="composeContent" :warning="composeWarning" />
    <template #confirm-text>更新并重建</template>
  </BaseModal>
</template>
