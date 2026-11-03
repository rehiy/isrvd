<script lang="ts">
import { Component, Prop, Vue, toNative } from 'vue-facing-decorator'

import CodeEditor from '@/component/code-editor.vue'

@Component({
    components: { CodeEditor },
    emits: ['update:modelValue']
})
class ComposeEditor extends Vue {
    @Prop({ default: '' }) modelValue!: string
    @Prop({ default: false }) disabled!: boolean
    @Prop({ default: '50vh' }) height!: string
    @Prop({ default: '' }) warning!: string
    @Prop({ default: 'compose.yml' }) filename!: 'compose.yml' | '.env'
}

export default toNative(ComposeEditor)
</script>

<template>
  <div class="space-y-3">
    <div>
      <label class="form-label">
        <i class="fas fa-file-code mr-1 text-slate-400"></i>{{ filename }}
      </label>
      <CodeEditor :model-value="modelValue" :filename="filename" :height="height" :disabled="disabled" @update:model-value="$emit('update:modelValue', $event)" />
      <p v-if="filename === '.env'" class="text-xs text-slate-400 mt-1">{{ $t('每行 KEY=VALUE，用于 Compose 变量插值') }}</p>
    </div>
    <div v-if="warning" class="bg-amber-50 border border-amber-200 rounded-lg p-3">
      <p class="text-sm text-amber-700">
        <i class="fas fa-exclamation-triangle mr-1"></i>{{ warning }}
      </p>
    </div>
  </div>
</template>
