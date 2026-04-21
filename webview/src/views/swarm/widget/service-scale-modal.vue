<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { SwarmServiceInfo } from '@/service/types'
import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    components: { BaseModal },
    emits: ['success']
})
class ServiceScaleModal extends Vue {
    // ─── 数据属性 ───
    isOpen = false
    loading = false
    service: SwarmServiceInfo | null = null
    replicas = 1

    // ─── 方法 ───
    show(svc: SwarmServiceInfo) {
        this.service = svc
        this.replicas = svc.replicas ?? 1
        this.isOpen = true
    }

    async handleConfirm() {
        if (!this.service) return
        this.loading = true
        try {
            await api.swarmServiceAction(this.service.id, 'scale', this.replicas)
            this.isOpen = false
            this.$emit('success')
        } catch (e) {}
        this.loading = false
    }
}

export default toNative(ServiceScaleModal)
</script>

<template>
  <BaseModal v-model="isOpen" title="服务扩缩容" :loading="loading" show-footer @confirm="handleConfirm">
    <template #confirm-text>确认扩缩容</template>
    <div v-if="service" class="space-y-4">
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">服务</label>
        <div class="px-3 py-2 bg-slate-50 rounded-lg text-sm text-slate-600">{{ service.name }}</div>
      </div>
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">目标副本数</label>
        <input type="number" v-model.number="replicas" min="0" max="100" class="input" />
        <p class="mt-1 text-xs text-slate-400">当前运行中副本：{{ service.runningTasks }} / {{ service.replicas }}</p>
      </div>
    </div>
  </BaseModal>
</template>
