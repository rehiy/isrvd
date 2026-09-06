<script lang="ts">
import { Component, Prop, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { ComposeDeployTarget } from '@/service/types'

import BaseModal from '@/component/modal.vue'

import ComposeEditor from '@/views/compose/widget/compose-editor.vue'
import EnvEditor from '@/views/compose/widget/env-editor.vue'

@Component({
    expose: ['show'],
    components: { BaseModal, ComposeEditor, EnvEditor },
    emits: ['success']
})
class RedeployModal extends Vue {
    @Prop({ type: String, required: true }) readonly target!: ComposeDeployTarget
    @Prop({ type: String, required: true }) readonly resourceName!: string
    @Prop({ type: String, required: true }) readonly title!: string
    @Prop({ type: String, required: true }) readonly warning!: string
    @Prop({ type: String, required: true }) readonly refreshTitle!: string
    @Prop({ type: String, required: true }) readonly successMessage!: string

    portal = usePortal()
    isOpen = false
    modalLoading = false
    refreshing = false
    composeContent = ''
    envContent = ''
    envTouched = false
    composeFileModTime = 0
    composeSource: 'file' | 'runtime' | '' = ''
    resolvedName = ''

    get composeWarning() {
        const parts = [this.warning]
        if (this.composeSource === 'runtime') {
            parts.push('当前为运行态反推结果，建议核对后再提交')
        }
        if (this.composeFileModTime) {
            parts.push(`文件更新时间：${new Date(this.composeFileModTime * 1000).toLocaleString('zh-CN', { hour12: false })}`)
        }
        return parts.join('；')
    }

    async show() {
        this.resolvedName = this.resourceName
        this.composeContent = ''
        this.envContent = ''
        this.envTouched = false
        this.composeFileModTime = 0
        this.composeSource = ''
        this.isOpen = true
        await this.loadCompose(false, true)
    }

    async loadCompose(force = false, closeOnError = false) {
        this.modalLoading = true
        if (force) this.refreshing = true
        try {
            const res = this.target === 'docker'
                ? await api.composeDockerInspect(this.resolvedName, force)
                : await api.composeSwarmInspect(this.resolvedName, force)
            const payload = res.payload
            this.composeContent = payload?.content || ''
            this.envContent = payload?.envContent || ''
            this.envTouched = false
            this.composeFileModTime = payload?.fileModTime || 0
            this.composeSource = payload?.source || ''
            if (this.target === 'docker') this.resolvedName = payload?.projectName || this.resolvedName
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

    onEnvContentChange(value: string) {
        this.envContent = value
        this.envTouched = true
    }

    async handleConfirm() {
        if (!this.composeContent.trim()) return
        this.modalLoading = true
        try {
            const data = {
                content: this.composeContent,
                envContent: this.envTouched ? this.envContent : undefined
            }
            if (this.target === 'docker') {
                await api.composeDockerRedeploy(this.resolvedName, data)
            } else {
                await api.composeSwarmRedeploy(this.resolvedName, data)
            }
            this.portal.showNotification('success', this.successMessage)
            this.isOpen = false
            this.$emit('success')
        } catch {} finally {
            this.modalLoading = false
        }
    }
}

export default toNative(RedeployModal)
</script>

<template>
  <BaseModal v-model="isOpen" :title="title" :loading="modalLoading" max-width-class="max-w-[80vw]" confirm-class="btn-emerald" show-footer @confirm="handleConfirm">
    <template #header-actions>
      <button type="button" class="btn-icon-sm" :disabled="modalLoading" :title="refreshTitle" @click="handleForceRefresh()">
        <i :class="refreshing ? 'fas fa-spinner fa-spin' : 'fas fa-rotate'"></i>
      </button>
    </template>

    <div class="grid gap-3 sm:grid-cols-2">
      <div class="min-w-0">
        <ComposeEditor v-model="composeContent" :warning="composeWarning" />
      </div>
      <div class="min-w-0">
        <EnvEditor :model-value="envContent" @update:model-value="onEnvContentChange" />
      </div>
    </div>
    <template #confirm-text>更新并重建</template>
  </BaseModal>
</template>
