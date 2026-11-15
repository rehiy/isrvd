<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { SwarmNodeDetail } from '@/service/types'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    components: { BaseModal },
    emits: ['success']
})
class NodeLabelModal extends Vue {
    portal = usePortal()
    isOpen = false
    loading = false
    nodeId = ''
    hostname = ''
    version = 0
    originalLabels: Record<string, string> = {}
    rows: { key: string; value: string }[] = []

    get validationError() {
        const keys = new Set<string>()
        for (const row of this.rows) {
            if (!row.key || row.key !== row.key.trim()) {
                return this.$t('标签键不能为空或包含首尾空白')
            }
            if (keys.has(row.key)) {
                return this.$t('标签键重复，请使用不同的键')
            }
            keys.add(row.key)
        }
        return ''
    }

    get hasChanges() {
        return this.rows.length !== Object.keys(this.originalLabels).length || this.rows.some(row =>
            !Object.prototype.hasOwnProperty.call(this.originalLabels, row.key) || this.originalLabels[row.key] !== row.value
        )
    }

    show(node: SwarmNodeDetail) {
        this.nodeId = node.id
        this.hostname = node.hostname
        this.version = node.version
        this.originalLabels = { ...node.labels }
        this.rows = Object.entries(node.labels || {}).sort(([a], [b]) => a.localeCompare(b)).map(([key, value]) => ({ key, value }))
        this.isOpen = true
    }

    async handleConfirm() {
        if (this.loading || this.validationError || !this.hasChanges || !this.portal.hasPerm('PUT /api/swarm/node/:id/labels')) {
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
        } finally {
            this.loading = false
        }
    }
}

export default toNative(NodeLabelModal)
</script>

<template>
  <BaseModal v-model="isOpen" :title="$t('编辑节点标签')" :loading="loading" :confirm-disabled="!!validationError || !hasChanges" confirm-class="btn-blue" @confirm="handleConfirm">
    <template #confirm-text>{{ $t('保存') }}</template>
    <div class="space-y-4">
      <p class="text-sm text-slate-600 break-all">{{ hostname }} <span class="text-xs text-slate-400 font-mono">{{ nodeId }}</span></p>
      <p class="text-sm text-amber-600">{{ $t('修改或删除标签可能触发服务重新调度，请确认服务的调度约束。') }}</p>
      <p v-if="rows.length === 0" class="empty-note">{{ $t('暂无节点标签，添加地域或存储标签供多个服务共用。') }}</p>
      <div v-for="(row, index) in rows" :key="index" class="panel-frame p-4">
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <label class="block min-w-0">
            <span class="form-label">{{ $t('标签键') }}</span>
            <input v-model="row.key" type="text" class="input font-mono" placeholder="region" autocomplete="off" :spellcheck="false" />
          </label>
          <label class="block min-w-0">
            <span class="form-label">{{ $t('标签值') }}</span>
            <input v-model="row.value" type="text" class="input font-mono" placeholder="guangzhou" autocomplete="off" :spellcheck="false" />
          </label>
        </div>
        <div class="flex justify-end mt-3">
          <button type="button" class="btn-icon btn-icon-red" :title="$t('删除标签')" :aria-label="$t('删除标签')" @click="rows.splice(index, 1)"><i class="fas fa-trash"></i></button>
        </div>
      </div>
      <p v-if="validationError" class="text-sm text-red-500" role="alert">{{ validationError }}</p>
      <button type="button" class="btn-add-row" @click="rows.push({ key: '', value: '' })"><i class="fas fa-plus"></i>{{ $t('添加标签') }}</button>
      <p class="text-xs text-slate-500">{{ $t('标签值允许为空。删除全部行并保存将清空节点标签。') }}</p>
    </div>
  </BaseModal>
</template>
