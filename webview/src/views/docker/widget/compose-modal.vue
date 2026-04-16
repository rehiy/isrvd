<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import { APP_ACTIONS_KEY } from '@/store/state'

import { Codemirror } from 'vue-codemirror'
import { yaml } from '@codemirror/lang-yaml'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    components: { BaseModal, Codemirror },
    emits: ['success']
})
class ComposeModal extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: any

    // ─── 数据属性 ───
    isOpen = false
    composeLoading = false
    composeContent = ''
    readonly composeExtensions = [yaml()]

    // ─── 方法 ───
    show() {
        this.composeContent = ''
        this.isOpen = true
    }

    async handleConfirm() {
        if (!this.composeContent.trim()) return
        this.composeLoading = true
        try {
            const res = await api.deployCompose(this.composeContent)
            const created = res.payload || []
            this.actions.showNotification('success', `Compose 部署成功，已创建 ${created.length} 个容器`)
            this.isOpen = false
            this.$emit('success')
        } catch (e) {
            this.actions.showNotification('error', 'Compose 部署失败')
        }
        this.composeLoading = false
    }
}

export default toNative(ComposeModal)
</script>

<template>
  <BaseModal
    ref="modalRef"
    v-model="isOpen"
    title="通过 Compose 创建容器"
    :loading="composeLoading"
    show-footer
    @confirm="handleConfirm"
  >
    <template #confirm-text>部署</template>
    <div class="space-y-4">
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">Compose 内容 <span class="text-red-500">*</span></label>
        <div class="rounded-xl overflow-hidden border border-slate-200">
          <Codemirror v-model="composeContent" :style="{ height: '50vh' }" :extensions="composeExtensions" :disabled="composeLoading" />
        </div>
        <p class="mt-1 text-xs text-slate-400">粘贴 docker-compose.yml 内容，将按照服务定义逐个创建容器</p>
      </div>
    </div>
  </BaseModal>
</template>
