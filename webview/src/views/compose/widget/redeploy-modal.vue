<script lang="ts">
import { Component, Prop, Vue, Watch, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { ComposeDeployTarget, ComposeHistoryInfo } from '@/service/types'

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
    composeContent = ''
    envContent = ''
    envTouched = false
    composeFileModTime = 0
    composeSource: 'file' | 'runtime' | '' = ''
    resolvedName = ''
    history: ComposeHistoryInfo[] = []
    selectedRevision = ''
    loadedRevision = ''
    historyError = false
    historyLoading = false
    loadError = false
    submitting = false
    loadedContent = ''
    loadedEnv = ''
    pendingSwitch: { force: boolean, revision: string } | null = null
    lastLoad = { force: false, revision: '' }
    loadGeneration = 0

    @Watch('isOpen')
    onOpenChange(open: boolean) {
        if (!open) {
            this.loadGeneration++
            this.modalLoading = false
            this.historyLoading = false
            this.submitting = false
        }
    }

    get canReadHistory() {
        return this.portal.hasPerm(`GET /api/compose/${this.target}/:name/history`)
    }

    get refreshing() {
        return this.modalLoading && !this.submitting && this.lastLoad.force
    }

    get dirty() {
        return this.composeContent !== this.loadedContent || this.envContent !== this.loadedEnv
    }

    get loadedVersionMissing() {
        return this.loadedRevision && !this.history.some(record => record.id === this.loadedRevision)
    }

    get confirmText() {
        if (this.submitting) return '重建中...'
        if (this.modalLoading) return '读取中...'
        return this.loadedRevision ? '使用此配置重建' : '更新并重建'
    }

    historyLabel(record: ComposeHistoryInfo) {
        const actions = { snapshot: '变更前', deploy: '部署', redeploy: '重部署' }
        return `${new Date(record.time).toLocaleString('zh-CN', { hour12: false })} · ${actions[record.action]} · ${record.success ? '成功' : '失败'}${record.hasSnapshot ? '' : ' · 无快照'}`
    }

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
        const generation = ++this.loadGeneration
        this.resolvedName = this.resourceName
        this.composeContent = ''
        this.envContent = ''
        this.envTouched = false
        this.composeFileModTime = 0
        this.composeSource = ''
        this.history = []
        this.selectedRevision = ''
        this.loadedRevision = ''
        this.historyError = false
        this.historyLoading = false
        this.loadError = false
        this.submitting = false
        this.loadedContent = ''
        this.loadedEnv = ''
        this.pendingSwitch = null
        this.isOpen = true
        await this.loadCompose()
        if (generation === this.loadGeneration && this.isOpen && this.canReadHistory) {
            await this.loadHistory()
        }
    }

    async loadHistory() {
        if (this.historyLoading) return
        const generation = this.loadGeneration
        this.historyLoading = true
        this.historyError = false
        try {
            const history = (await api.composeHistoryList(this.target, this.resolvedName)).payload || []
            if (generation === this.loadGeneration && this.isOpen) this.history = history
        } catch {
            if (generation === this.loadGeneration && this.isOpen) this.historyError = true
        } finally {
            if (generation === this.loadGeneration) this.historyLoading = false
        }
    }

    async loadCompose(force = false, revision = '') {
        const generation = this.loadGeneration
        this.modalLoading = true
        this.loadError = false
        this.lastLoad = { force, revision }
        try {
            const res = revision
                ? await api.composeHistoryInspect(this.target, this.resolvedName, revision)
                : this.target === 'docker'
                ? await api.composeDockerInspect(this.resolvedName, force)
                : await api.composeSwarmInspect(this.resolvedName, force)
            if (generation !== this.loadGeneration || !this.isOpen) return
            const payload = res.payload
            if (!payload) throw new Error('配置内容为空')
            this.composeContent = payload.content || ''
            this.envContent = payload.envContent || ''
            this.loadedContent = this.composeContent
            this.loadedEnv = this.envContent
            this.envTouched = !!revision
            this.selectedRevision = revision
            this.loadedRevision = revision
            this.composeFileModTime = payload.fileModTime || 0
            this.composeSource = payload.source || ''
            if (this.target === 'docker') this.resolvedName = payload.projectName || this.resolvedName
            if (force) this.portal.showNotification('success', '已从运行态重新反推 Compose 配置')
        } catch {
            if (generation !== this.loadGeneration || !this.isOpen) return
            this.selectedRevision = this.loadedRevision
            this.loadError = true
        } finally {
            if (generation === this.loadGeneration) {
                this.modalLoading = false
            }
        }
    }

    async requestLoad(force = false, revision = '') {
        if (this.modalLoading) return
        this.selectedRevision = this.loadedRevision
        if (this.dirty) {
            this.pendingSwitch = { force, revision }
            return
        }
        this.pendingSwitch = null
        await this.loadCompose(force, revision)
    }

    async discardAndLoad() {
        const next = this.pendingSwitch
        if (!next || this.modalLoading) return
        this.pendingSwitch = null
        await this.loadCompose(next.force, next.revision)
    }

    onEnvContentChange(value: string) {
        this.envContent = value
        this.envTouched = true
    }

    async handleConfirm() {
        if (this.modalLoading || this.pendingSwitch || !this.composeContent.trim()) return
        const generation = this.loadGeneration
        this.modalLoading = true
        this.submitting = true
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
            if (generation === this.loadGeneration) this.isOpen = false
            this.$emit('success')
        } catch {} finally {
            if (generation === this.loadGeneration) {
                this.modalLoading = false
                this.submitting = false
            }
        }
    }
}

export default toNative(RedeployModal)
</script>

<template>
  <BaseModal v-model="isOpen" :title="title" :loading="modalLoading" :close-disabled="submitting" :confirm-disabled="!composeContent.trim() || !!pendingSwitch" max-width-class="max-w-6xl" confirm-class="btn-emerald" show-footer @confirm="handleConfirm">
    <template #header-actions>
      <button type="button" class="btn-icon-sm" :disabled="modalLoading" :title="refreshTitle" @click="requestLoad(true)">
        <i :class="refreshing ? 'fas fa-spinner fa-spin' : 'fas fa-rotate'"></i>
      </button>
    </template>

    <div v-if="canReadHistory" class="mb-3">
      <div class="flex items-center justify-between gap-2 mb-1">
        <label class="form-label mb-0">配置版本</label>
        <div class="action-group-sm">
          <span v-if="dirty" class="text-xs text-amber-600">未提交修改</span>
          <button type="button" class="btn-icon-sm" :disabled="modalLoading || historyLoading" :title="historyError ? '重试加载部署记录' : '刷新部署记录'" @click="loadHistory">
            <i :class="historyLoading ? 'fas fa-spinner fa-spin' : 'fas fa-clock-rotate-left'"></i>
          </button>
        </div>
      </div>
      <select v-model="selectedRevision" class="input" :disabled="modalLoading || historyLoading || !!pendingSwitch" @change="requestLoad(false, selectedRevision)">
        <option value="">当前配置</option>
        <option v-if="loadedVersionMissing" :value="loadedRevision" disabled>已加载的历史配置</option>
        <option v-for="record in history" :key="record.id" :value="record.id" :disabled="!record.hasSnapshot">{{ historyLabel(record) }}</option>
      </select>
      <p v-if="historyLoading" class="text-xs text-slate-500 mt-1" role="status">正在加载部署记录...</p>
      <p v-else-if="historyError" class="text-sm text-red-500 mt-1" role="alert">部署记录加载失败</p>
      <p v-else-if="history.length === 0" class="text-xs text-slate-400 mt-1">暂无部署记录</p>
      <p v-if="loadedRevision" class="text-sm text-amber-600 mt-1">历史配置将替换当前配置并重建实例；数据卷与数据库内容不会恢复。</p>
    </div>
    <div v-if="pendingSwitch" class="bg-amber-50 border border-amber-200 rounded-lg p-3 mb-3" role="alert">
      <p class="text-sm text-amber-700 mb-2">当前修改尚未提交，切换配置将丢弃这些修改。</p>
      <div class="flex flex-wrap gap-2">
        <button type="button" class="btn btn-secondary" @click="pendingSwitch = null">保留编辑</button>
        <button type="button" class="btn btn-amber" @click="discardAndLoad"><i class="fas fa-rotate-left"></i>放弃修改并切换</button>
      </div>
    </div>
    <div v-if="loadError" class="flex flex-wrap items-center gap-2 mb-3" role="alert">
      <span class="text-sm text-red-500">{{ loadedContent ? '配置加载失败，已保留原内容' : '配置加载失败' }}</span>
      <button type="button" class="btn-icon-sm" title="重试加载配置" @click="requestLoad(lastLoad.force, lastLoad.revision)"><i class="fas fa-rotate"></i></button>
    </div>
    <p v-if="modalLoading" class="text-sm text-slate-500 mb-3" role="status">{{ submitting ? '正在重建应用...' : '正在读取配置...' }}</p>
    <div class="grid gap-3 sm:grid-cols-2">
      <div class="min-w-0">
        <ComposeEditor v-model="composeContent" :disabled="modalLoading" height="min(42vh, 360px)" />
      </div>
      <div class="min-w-0">
        <EnvEditor :model-value="envContent" :disabled="modalLoading" height="min(42vh, 360px)" @update:model-value="onEnvContentChange" />
      </div>
      <div v-if="composeWarning" class="bg-amber-50 border border-amber-200 rounded-lg p-3 sm:col-span-2">
        <p class="text-sm text-amber-700">
          <i class="fas fa-exclamation-triangle mr-1"></i>{{ composeWarning }}
        </p>
      </div>
    </div>
    <template #confirm-text>{{ confirmText }}</template>
  </BaseModal>
</template>
