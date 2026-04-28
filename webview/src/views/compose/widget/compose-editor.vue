<script lang="ts">
import { yaml } from '@codemirror/lang-yaml'
import { Codemirror } from 'vue-codemirror'
import { Component, Prop, Vue, toNative } from 'vue-facing-decorator'

@Component({
    components: { Codemirror },
    emits: ['update:modelValue']
})
class ComposeEditor extends Vue {
    @Prop({ default: '' }) modelValue!: string
    @Prop({ default: false }) disabled!: boolean
    @Prop({ default: '50vh' }) height!: string
    @Prop({ default: '' }) warning!: string

    readonly extensions = [yaml()]

    get content() {
        return this.modelValue
    }

    set content(val: string) {
        this.$emit('update:modelValue', val)
    }
}

export default toNative(ComposeEditor)
</script>

<template>
  <div class="space-y-3">
    <div v-if="warning" class="bg-amber-50 border border-amber-200 rounded-lg p-3">
      <p class="text-sm text-amber-700">
        <i class="fas fa-exclamation-triangle mr-1"></i>{{ warning }}
      </p>
    </div>
    <div>
      <label class="block text-sm font-medium text-slate-700 mb-2">
        <i class="fas fa-file-code mr-1 text-slate-400"></i>compose.yml
      </label>
      <div class="rounded-xl overflow-hidden border border-slate-200">
        <Codemirror
          v-model="content"
          :style="{ height }"
          :extensions="extensions"
          :disabled="disabled"
        />
      </div>
    </div>
  </div>
</template>
