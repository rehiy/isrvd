<script lang="ts">
import { Component, Prop, Vue, toNative } from 'vue-facing-decorator'
import { Codemirror } from 'vue-codemirror'
import { yaml } from '@codemirror/lang-yaml'

@Component({
    components: { Codemirror },
    emits: ['update:modelValue']
})
class ComposeEditor extends Vue {
    @Prop({ default: '' }) modelValue!: string
    @Prop({ default: false }) disabled!: boolean
    @Prop({ default: '50vh' }) height!: string

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
  <div class="rounded-xl overflow-hidden border border-slate-200">
    <Codemirror
      v-model="content"
      :style="{ height }"
      :extensions="extensions"
      :disabled="disabled"
    />
  </div>
</template>
