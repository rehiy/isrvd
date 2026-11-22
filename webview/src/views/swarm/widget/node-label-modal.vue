<script lang="ts">
import axios from 'axios'
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    components: { BaseModal },
    emits: ['success']
})
class NodeLabelModal extends Vue {
    portal = usePortal()
    @Ref readonly formRef!: HTMLFormElement

    // ─── 数据属性 ───
    isOpen = false
    loading = false
    loaded = false
    conflicted = false
    nodeId = ''
    version = 0
    nextRowId = 0
    originalLabels: Record<string, string> = {}
    rows: { id: number; key: string; value: string }[] = []

    // ─── 计算属性 ───
    get canEdit() {
        return this.portal.hasPerm('GET /api/swarm/node/:id') && this.portal.hasPerm('PUT /api/swarm/node/:id/labels')
    }

    get rowErrors() {
        const counts = new Map<string, number>()
        for (const row of this.rows) {
            counts.set(row.key, (counts.get(row.key) || 0) + 1)
        }
        return this.rows.map(row => {
            if (row.key !== row.key.trim()) {
                return this.$t('标签键不能为空或包含首尾空白')
            }
            if (row.key && (counts.get(row.key) || 0) > 1) {
                return this.$t('标签键重复，请使用不同的键')
            }
            return ''
        })
    }

    get confirmDisabled() {
        return !this.loaded || this.conflicted || !this.canEdit || !this.hasChanges
            || this.rows.some(row => !row.key) || this.rowErrors.some(Boolean)
    }

    get hasChanges() {
        return this.rows.length !== Object.keys(this.originalLabels).length || this.rows.some(row =>
            !Object.prototype.hasOwnProperty.call(this.originalLabels, row.key) || this.originalLabels[row.key] !== row.value
        )
    }

    // ─── 方法 ───
    async show(nodeId: string) {
        if (this.loading || this.isOpen || !this.canEdit) {
            return
        }
        this.nodeId = nodeId
        this.version = 0
        this.originalLabels = {}
        this.rows = []
        this.nextRowId = 0
        this.loaded = false
        this.conflicted = false
        this.isOpen = true
        try {
            await this.loadLabels()
        } catch (error) {
            this.isOpen = false
            throw error
        }
    }

    async loadLabels() {
        if (this.loading || !this.isOpen || !this.canEdit) {
            return
        }
        this.loading = true
        try {
            const node = (await api.swarmNodeInspect(this.nodeId)).payload
            if (!node) {
                this.portal.showNotification('error', this.$t('未找到节点详情'))
                this.isOpen = false
                return
            }
            this.version = node.version
            this.originalLabels = { ...node.labels }
            this.rows = Object.entries(node.labels || {}).sort(([a], [b]) => a.localeCompare(b)).map(([key, value]) => ({ id: this.nextRowId++, key, value }))
            this.loaded = true
            this.conflicted = false
        } finally {
            this.loading = false
        }
    }

    async addLabel() {
        if (this.loading || !this.loaded || !this.isOpen || !this.canEdit) {
            return
        }
        this.rows.push({ id: this.nextRowId++, key: '', value: '' })
        await this.$nextTick()
        const inputs = this.formRef.querySelectorAll<HTMLInputElement>('[data-label-key]')
        inputs[inputs.length - 1]?.focus()
    }

    removeLabel(id: number) {
        if (this.loading || !this.loaded || !this.isOpen || !this.canEdit) {
            return
        }
        this.rows = this.rows.filter(row => row.id !== id)
    }

    async handleConfirm() {
        if (this.loading || !this.isOpen || this.confirmDisabled || !this.formRef.reportValidity()) {
            return
        }
        this.loading = true
        try {
            await api.swarmNodeLabelUpdate(this.nodeId, {
                version: this.version,
                labels: Object.fromEntries(this.rows.map(row => [row.key, row.value]))
            })
            this.isOpen = false
            this.portal.showNotification('success', this.$t('节点标签已更新'))
            this.$emit('success', this.nodeId)
        } catch (error) {
            if (axios.isAxiosError(error) && error.response?.status === 409) {
                this.conflicted = true
            }
            throw error
        } finally {
            this.loading = false
        }
    }
}

export default toNative(NodeLabelModal)
</script>

<template>
  <BaseModal v-model="isOpen" :title="$t('编辑节点标签')" :loading="loading" :confirm-disabled="confirmDisabled" confirm-class="btn-blue" @confirm="handleConfirm">
    <template #confirm-text>{{ $t('保存') }}</template>
    <div v-if="!loaded" class="empty-state">
      <div class="spinner-lg"></div>
      <p class="text-slate-500">{{ $t('加载中...') }}</p>
    </div>
    <form v-else ref="formRef" class="space-y-4" @submit.prevent="handleConfirm">
      <button type="submit" class="sr-only" tabindex="-1" :disabled="loading || confirmDisabled">{{ $t('保存') }}</button>
      <div class="space-y-4">
        <p v-if="rows.length === 0" class="empty-note">{{ hasChanges ? $t('保存后将清空节点标签。') : $t('暂无节点标签') }}</p>
        <div v-for="(row, index) in rows" :key="row.id" class="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <div class="min-w-0">
            <label class="block">
              <span class="form-label">{{ $t('标签键') }} <span class="text-red-500">*</span></span>
              <input v-model="row.key" type="text" class="input font-mono" placeholder="region" autocomplete="off" :spellcheck="false" required data-label-key :aria-invalid="!!rowErrors[index]" :aria-describedby="rowErrors[index] ? `node-label-error-${row.id}` : undefined" />
            </label>
            <p v-if="rowErrors[index]" :id="`node-label-error-${row.id}`" class="text-xs text-red-500 mt-1">{{ rowErrors[index] }}</p>
          </div>
          <div class="min-w-0">
            <label :for="`node-label-value-${row.id}`" class="form-label">{{ $t('标签值') }}</label>
            <div class="flex items-center gap-2">
              <input :id="`node-label-value-${row.id}`" v-model="row.value" type="text" class="input min-w-0 font-mono" placeholder="guangzhou" autocomplete="off" :spellcheck="false" />
              <button type="button" class="btn-icon btn-icon-red shrink-0" :title="$t('删除标签')" :aria-label="$t('删除标签')" @click="removeLabel(row.id)"><i class="fas fa-trash"></i></button>
            </div>
          </div>
        </div>
        <p v-if="rows.length > 0" class="text-xs text-slate-400 mt-1">{{ $t('标签值可留空。') }}</p>
        <button type="button" class="btn-add-row" @click="addLabel"><i class="fas fa-plus"></i>{{ $t('添加标签') }}</button>
      </div>
      <div class="modal-warning">
        <div class="flex items-start gap-2">
          <i class="fas fa-triangle-exclamation mt-0.5" aria-hidden="true"></i>
          <p aria-live="polite">{{ conflicted ? $t('节点已更新，当前修改尚未保存。重新加载将丢弃这些修改。') : $t('修改或删除标签可能触发服务重新调度，请确认服务的调度约束。') }}</p>
        </div>
        <button v-if="conflicted" type="button" class="btn btn-secondary mt-2" @click="loadLabels">{{ $t('放弃修改并重新加载') }}</button>
      </div>
    </form>
  </BaseModal>
</template>
