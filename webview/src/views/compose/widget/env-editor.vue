<script lang="ts">
import { Codemirror } from 'vue-codemirror'
import { Component, Prop, Vue, toNative } from 'vue-facing-decorator'

@Component({
    components: { Codemirror },
    emits: ['update:modelValue']
})
class EnvEditor extends Vue {
    @Prop({ default: '' }) modelValue!: string
    @Prop({ default: false }) disabled!: boolean
    @Prop({ default: '50vh' }) height!: string
    @Prop({ default: '' }) warning!: string

    get content() {
        return this.modelValue
    }

    set content(val: string) {
        this.$emit('update:modelValue', val)
    }
}

export default toNative(EnvEditor)
</script>

<template>
  <div class="space-y-3">
    <div>
      <label class="form-label">
        <i class="fas fa-file-code mr-1 text-slate-400"></i>.env
        <span class="text-xs font-normal text-slate-400">（KEY=VALUE，用于 Compose 变量插值）</span>
      </label>
      <div class="editor-container">
        <Codemirror v-model="content" :style="{ height }" :disabled="disabled" />
      </div>
    </div>
    <div v-if="warning" class="bg-amber-50 border border-amber-200 rounded-lg p-3">
      <p class="text-sm text-amber-700">
        <i class="fas fa-exclamation-triangle mr-1"></i>{{ warning }}
      </p>
    </div>
  </div>
</template>
