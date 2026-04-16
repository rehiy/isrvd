<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import { APP_STATE_KEY, APP_ACTIONS_KEY } from '@/store/state'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    components: { BaseModal }
})
class RenameModal extends Vue {
    @Inject({ from: APP_STATE_KEY }) readonly state!: any
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: any

    // ─── 数据属性 ───
    isOpen = false
    formData = { name: '', file: null as any }

    // ─── 方法 ───
    show(file: any) {
        this.formData.file = file
        this.formData.name = file.name
        this.isOpen = true
    }

    async handleConfirm() {
        if (!this.formData.name.trim() || !this.formData.file) return
        await api.rename(this.formData.file.path, this.formData.name)
        this.actions.loadFiles()
        this.isOpen = false
    }
}

export default toNative(RenameModal)
</script>

<template>
  <BaseModal ref="modalRef" v-model="isOpen" title="重命名" :loading="state.loading" :confirm-disabled="!formData.name.trim()" @confirm="handleConfirm">
    <form @submit.prevent="handleConfirm">
      <div>
        <label for="target" class="block text-sm font-medium text-slate-700 mb-2">
          新名称
        </label>
        <div class="relative">
          <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
            <i class="fas fa-pen text-slate-400"></i>
          </div>
          <input 
            type="text" 
            id="target" 
            v-model="formData.name" 
            :disabled="state.loading" 
            required
            class="input pl-11"
            placeholder="请输入新名称"
          >
        </div>
      </div>
    </form>
    <template #confirm-text>
      <i class="fas fa-check mr-2" v-if="!state.loading"></i>
      {{ state.loading ? '重命名中...' : '确认重命名' }}
    </template>
  </BaseModal>
</template>
