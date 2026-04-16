<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import { APP_STATE_KEY, APP_ACTIONS_KEY } from '@/store/state'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    components: { BaseModal }
})
class CreateModal extends Vue {
    @Inject({ from: APP_STATE_KEY }) readonly state!: any
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: any

    // ─── 数据属性 ───
    isOpen = false
    formData = { name: '', content: '' }

    // ─── 方法 ───
    show() {
        this.formData = { name: '', content: '' }
        this.isOpen = true
    }

    async handleConfirm() {
        if (!this.formData.name.trim()) return
        await api.create(this.state.currentPath + '/' + this.formData.name, this.formData.content)
        this.actions.loadFiles()
        this.isOpen = false
    }
}

export default toNative(CreateModal)
</script>

<template>
  <BaseModal ref="modalRef" v-model="isOpen" title="新建文件" :loading="state.loading" :confirm-disabled="!formData.name.trim()" @confirm="handleConfirm">
    <form @submit.prevent="handleConfirm" class="space-y-5">
      <div>
        <label for="fileName" class="block text-sm font-medium text-slate-700 mb-2">
          文件名称
        </label>
        <div class="relative">
          <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
            <i class="fas fa-file text-slate-400"></i>
          </div>
          <input 
            type="text" 
            id="fileName" 
            v-model="formData.name" 
            :disabled="state.loading" 
            required
            class="input pl-11"
            placeholder="请输入文件名称"
          >
        </div>
      </div>
      <div>
        <label for="fileContent" class="block text-sm font-medium text-slate-700 mb-2">
          文件内容
        </label>
        <textarea 
          id="fileContent" 
          rows="10" 
          v-model="formData.content" 
          :disabled="state.loading"
          class="input font-mono text-sm"
          placeholder="请输入文件内容..."
        ></textarea>
      </div>
    </form>
    <template #confirm-text>
      <i class="fas fa-file-circle-plus mr-2" v-if="!state.loading"></i>
      {{ state.loading ? '创建中...' : '创建文件' }}
    </template>
  </BaseModal>
</template>
