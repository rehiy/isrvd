<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import BaseModal from '@/component/modal.vue'

@Component({
    components: { BaseModal }
})
class ConfirmModal extends Vue {
    portal = usePortal()

    // ─── 计算属性 ───
    get messageParts() {
        return this.portal.confirm.message
            .split(/<strong(?: class="text-slate-900")?>([\s\S]*?)<\/strong>/)
            .map((text, index) => ({ text, strong: index % 2 === 1 }))
    }

    get iconColorClass() {
        const colors: Record<string, string> = {
            blue: 'bg-blue-100 text-blue-500',
            emerald: 'bg-emerald-100 text-emerald-500',
            amber: 'bg-amber-100 text-amber-500',
            red: 'bg-red-100 text-red-500',
            slate: 'bg-slate-100 text-slate-500'
        }
        return colors[this.portal.confirm.iconColor] || colors.blue
    }
}

export default toNative(ConfirmModal)
</script>

<template>
  <BaseModal
    :model-value="portal.confirm.show"
    :title="$t(portal.confirm.title)"
    :loading="portal.confirm.loading"
    :confirm-class="portal.confirm.danger ? 'btn-danger' : 'btn-primary'"
    @cancel="portal.closeConfirm"
    @confirm="portal.handleConfirm"
    @update:model-value="!$event && portal.closeConfirm()"
  >
    <div class="text-center">
      <div class="empty-state-icon mx-auto" :class="iconColorClass.split(' ')[0]">
        <i class="fas text-3xl" :class="[portal.confirm.icon, iconColorClass.split(' ')[1]]"></i>
      </div>
      <p class="text-lg text-slate-700 break-words">
        <template v-for="(part, index) in messageParts" :key="index"><strong v-if="part.strong" class="text-slate-900">{{ part.text }}</strong><template v-else>{{ part.text }}</template></template>
      </p>
      <p v-if="portal.confirm.danger" class="text-sm text-red-600 flex items-center justify-center mt-3">
        <i class="fas fa-exclamation-triangle mr-2"></i>
        {{ $t('此操作不可恢复！') }}
      </p>
    </div>

    <template #confirm-text>
      {{ $t(portal.confirm.confirmText) }}
    </template>
  </BaseModal>
</template>
