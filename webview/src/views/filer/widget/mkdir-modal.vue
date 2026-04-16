<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import { APP_STATE_KEY, APP_ACTIONS_KEY } from '@/store/state'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    components: { BaseModal }
})
class MkdirModal extends Vue {
    @Inject({ from: APP_STATE_KEY }) readonly state!: any
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: any

    // ─── 数据属性 ───
    isOpen = false
    formData = { name: '' }

    // ─── 方法 ───
    show() {
        this.formData.name = ''
        this.isOpen = true
    }

    async handleConfirm() {
        if (!this.formData.name.trim()) return
        await api.mkdir(this.state.currentPath + '/' + this.formData.name)
        this.actions.loadFiles()
        this.isOpen = false
    }
}

export default toNative(MkdirModal)
</script>

<template>
  <BaseModal ref="modalRef" v-model="isOpen" title="新建目录" :loading="state.loading" :confirm-disabled="!formData.name.trim()" @confirm="handleConfirm">
    <form @submit.prevent="handleConfirm">
      <div>
        <label for="dirName" class="block text-sm font-medium text-slate-700 mb-2">
          目录名称
        </label>
        <div class="relative">
          <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
            <i class="fas fa-folder text-slate-400"></i>
          </div>
          <input 
            type="text" 
            id="dirName" 
            v-model="formData.name" 
            :disabled="state.loading" 
            required
            class="input pl-11"
            placeholder="请输入目录名称"
          >
        </div>
      </div>
    </form>
    <template #confirm-text>
      <i class="fas fa-folder mr-2" v-if="!state.loading"></i>
      {{ state.loading ? '创建中...' : '创建目录' }}
    </template>
  </BaseModal>
</template>
