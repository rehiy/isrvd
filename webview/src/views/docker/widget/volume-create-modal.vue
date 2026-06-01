<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    components: { BaseModal },
    emits: ['success']
})
class VolumeCreateModal extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    isOpen = false
    modalLoading = false
    formData = { name: '', driver: 'local' }

    // ─── 方法 ───
    show() {
        this.formData = { name: '', driver: 'local' }
        this.isOpen = true
    }

    async handleConfirm() {
        if (!this.formData.name.trim()) return
        this.modalLoading = true
        try {
            await api.dockerVolumeCreate(this.formData)
            this.portal.showNotification('success', '数据卷创建成功')
            this.isOpen = false
            this.$emit('success')
        } catch {}
        this.modalLoading = false
    }
}

export default toNative(VolumeCreateModal)
</script>

<template>
  <BaseModal v-model="isOpen" title="新建数据卷" :loading="modalLoading" show-footer confirm-class="btn-amber" @confirm="handleConfirm">
    <form class="space-y-4" @submit.prevent="handleConfirm">
      <div>
        <label class="form-label">卷名称</label>
        <input v-model="formData.name" type="text" placeholder="请输入数据卷名称" required class="input" />
      </div>
      <div>
        <label class="form-label">驱动类型</label>
        <select v-model="formData.driver" class="input">
          <option value="local">local (本地)</option>
        </select>
      </div>
    </form>

    <template #confirm-text>确认新建</template>
  </BaseModal>
</template>
