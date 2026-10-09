<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { NodeInfo } from '@/service/types'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    components: { BaseModal },
    emits: ['success']
})
class NodeEditModal extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    isOpen = false
    modalLoading = false
    editId = ''
    name = ''

    // ─── 方法 ───
    show(node: NodeInfo) {
        this.editId = node.id
        this.name = node.name
        this.isOpen = true
    }

    async handleConfirm() {
        const name = this.name.trim()
        if (!name) return

        this.modalLoading = true
        try {
            await api.nodeUpdate(this.editId, { name })
            this.portal.showNotification('success', '节点已重命名')
            this.isOpen = false
            this.$emit('success')
        } catch {}
        this.modalLoading = false
    }
}

export default toNative(NodeEditModal)
</script>

<template>
  <BaseModal v-model="isOpen" title="重命名节点" :loading="modalLoading" :confirm-disabled="!name.trim()" max-width-class="max-w-lg" confirm-class="btn-blue" show-footer @confirm="handleConfirm">
    <form class="space-y-4" @submit.prevent="handleConfirm">
      <div>
        <label class="form-label">节点名称 <span class="text-red-500">*</span></label>
        <input v-model="name" type="text" maxlength="64" placeholder="请输入节点名称" required class="input" />
        <p class="text-xs text-slate-400 mt-1">1–64 个字符，仅用于在界面中区分节点</p>
      </div>
    </form>

    <template #confirm-text>保存修改</template>
  </BaseModal>
</template>
