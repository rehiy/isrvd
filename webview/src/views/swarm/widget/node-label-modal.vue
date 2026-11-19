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
    hostname = ''
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
            if (!row.key || row.key !== row.key.trim()) {
                return this.$t('标签键不能为空或包含首尾空白')
            }
            if ((counts.get(row.key) || 0) > 1) {
                return this.$t('标签键重复，请使用不同的键')
            }
            return ''
        })
    }

    get confirmDisabled() {
        return !this.loaded || this.conflicted || !this.canEdit || !this.hasChanges || this.rowErrors.some(Boolean)
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
        this.hostname = ''
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
            this.hostname = node.hostname
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
      <div>
        <p class="text-sm font-medium text-slate-700 break-all">{{ hostname }}</p>
        <p class="text-mono-muted break-all">{{ nodeId }}</p>
      </div>
      <p class="text-sm text-amber-600">{{ $t('修改或删除标签可能触发服务重新调度，请确认服务的调度约束。') }}</p>
      <div v-if="conflicted" class="panel-frame p-4 space-y-3" role="status">
        <p class="text-sm text-amber-600">{{ $t('节点已更新，当前修改尚未保存。重新加载将丢弃这些修改。') }}</p>
        <button type="button" class="btn btn-secondary" @click="loadLabels">{{ $t('放弃修改并重新加载') }}</button>
      </div>
      <div class="space-y-4">
        <p v-if="rows.length === 0" class="empty-note">{{ $t('暂无节点标签') }}</p>
        <div v-for="(row, index) in rows" :key="row.id" class="grid grid-cols-[minmax(0,1fr)_auto] sm:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto] items-start gap-3">
          <div class="min-w-0">
            <label class="block">
              <span class="form-label">{{ $t('标签键') }} <span class="text-red-500">*</span></span>
              <input v-model="row.key" type="text" class="input font-mono" placeholder="region" autocomplete="off" :spellcheck="false" required data-label-key :aria-invalid="!!rowErrors[index]" :aria-describedby="rowErrors[index] ? `node-label-error-${row.id}` : undefined" />
            </label>
            <p v-if="rowErrors[index]" :id="`node-label-error-${row.id}`" class="text-xs text-red-500 mt-1">{{ rowErrors[index] }}</p>
          </div>
          <label class="block min-w-0 col-span-2 row-start-2 sm:col-span-1 sm:row-start-1 sm:col-start-2">
            <span class="form-label">{{ $t('标签值') }}</span>
            <input v-model="row.value" type="text" class="input font-mono" placeholder="guangzhou" autocomplete="off" :spellcheck="false" />
          </label>
          <button type="button" class="btn-icon btn-icon-red mt-6 col-start-2 row-start-1 sm:col-start-3" :title="$t('删除标签')" :aria-label="$t('删除标签')" @click="removeLabel(row.id)"><i class="fas fa-trash"></i></button>
        </div>
        <div>
          <button type="button" class="btn-add-row" @click="addLabel"><i class="fas fa-plus"></i>{{ $t('添加标签') }}</button>
          <p class="text-xs text-slate-400 mt-1">{{ $t('标签值允许为空。删除全部行并保存将清空节点标签。') }}</p>
        </div>
      </div>
      <button type="submit" class="sr-only" tabindex="-1" :disabled="loading || confirmDisabled">{{ $t('保存') }}</button>
    </form>
  </BaseModal>
</template>
