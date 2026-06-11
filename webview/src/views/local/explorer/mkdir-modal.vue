<script lang="ts">
import { Component, Prop, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    emits: ['success'],
    components: { BaseModal }
})
class MkdirModal extends Vue {
    @Prop({ required: true }) readonly currentPath!: string

    // ─── 数据属性 ───
    isOpen = false
    loading = false
    formData = { name: '' }

    // ─── 方法 ───
    show() {
        this.formData.name = ''
        this.isOpen = true
    }

    async handleConfirm() {
        if (!this.formData.name.trim()) return
        this.loading = true
        try {
            await api.filerMkdir(this.currentPath + '/' + this.formData.name)
            this.$emit('success')
            this.isOpen = false
        } finally {
            this.loading = false
        }
    }
}

export default toNative(MkdirModal)
</script>

<template>
  <BaseModal ref="modalRef" v-model="isOpen" title="新建目录" :loading="loading" :confirm-disabled="!formData.name.trim()" @confirm="handleConfirm">
    <form @submit.prevent="handleConfirm">
      <div>
        <label for="dirName" class="form-label">
          目录名称
        </label>
        <div class="relative">
          <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
            <i class="fas fa-folder text-slate-400"></i>
          </div>
          <input id="dirName" v-model="formData.name" type="text" :disabled="loading" required class="input pl-11" placeholder="请输入目录名称">
        </div>
      </div>
    </form>

    <template #confirm-text>
      {{ loading ? '新建中...' : '新建目录' }}
    </template>
  </BaseModal>
</template>
