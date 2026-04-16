<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import { Codemirror } from 'vue-codemirror'
import { yaml } from '@codemirror/lang-yaml'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    components: { BaseModal, Codemirror },
    emits: ['success']
})
class SwarmComposeModal extends Vue {
    // ─── 数据属性 ───
    isOpen = false
    loading = false
    content = ''
    readonly extensions = [yaml()]

    // ─── 方法 ───
    show() {
        this.content = ''
        this.isOpen = true
    }

    async handleConfirm() {
        if (!this.content.trim()) return
        this.loading = true
        try {
            const res = await api.swarmDeployComposeService(this.content)
            const created = res.payload || []
            this.isOpen = false
            this.$emit('success', created.length)
        } catch (e) {}
        this.loading = false
    }
}

export default toNative(SwarmComposeModal)
</script>

<template>
  <BaseModal
    v-model="isOpen"
    title="通过 Compose 创建服务"
    :loading="loading"
    show-footer
    @confirm="handleConfirm"
  >
    <template #confirm-text>部署</template>
    <div class="space-y-4">
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">Compose 内容 <span class="text-red-500">*</span></label>
        <div class="rounded-xl overflow-hidden border border-slate-200">
          <Codemirror v-model="content" :style="{ height: '50vh' }" :extensions="extensions" :disabled="loading" />
        </div>
        <p class="mt-1 text-xs text-slate-400">粘贴 docker-compose.yml 内容，将按照服务定义逐个创建 Swarm 服务</p>
      </div>
    </div>
  </BaseModal>
</template>
