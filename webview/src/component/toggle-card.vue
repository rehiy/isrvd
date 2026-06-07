<script lang="ts">
import { Component, Prop, Vue, toNative } from 'vue-facing-decorator'

@Component({ emits: ['update:modelValue'] })
class ToggleCard extends Vue {
  @Prop({ type: Boolean, default: false }) modelValue!: boolean
  @Prop({ type: String, required: true }) label!: string
  @Prop({ type: String, default: '' }) desc!: string
  @Prop({ type: Boolean, default: false }) violet!: boolean
  @Prop({ type: Boolean, default: false }) disabled!: boolean

  get hasBody(): boolean {
    return !!this.$slots.default
  }

  get hasDesc(): boolean {
    return !!this.$slots.desc || !!this.desc
  }

  toggle() {
    if (this.disabled) return
    this.$emit('update:modelValue', !this.modelValue)
  }
}

export default toNative(ToggleCard)
</script>

<template>
  <!-- 有 body：toggle-expand-card 样式（收缩时普通 toggle-row，展开时变 card） -->
  <div v-if="hasBody" class="toggle-expand-card" :class="{ 'toggle-expand-card-open': modelValue }">
    <div class="toggle-row">
      <div>
        <span class="text-sm text-slate-600">{{ label }}</span>
        <!-- eslint-disable-next-line vue/no-v-html -->
        <p v-if="hasDesc" class="text-xs text-slate-400 mt-0.5">
          <slot name="desc">
            {{ desc }}
          </slot>
        </p>
      </div>
      <button
        type="button"
        class="toggle"
        :class="{ 'toggle-on': modelValue, 'toggle-violet': violet, 'opacity-50 cursor-not-allowed': disabled }"
        :disabled="disabled"
        role="switch"
        :aria-checked="modelValue"
        @click="toggle"
      >
        <span class="toggle-thumb" />
      </button>
    </div>
    <div v-if="modelValue" class="toggle-expand-body">
      <slot />
    </div>
  </div>

  <!-- 无 body：普通 toggle-row -->
  <div v-else class="toggle-row">
    <div>
      <span class="text-sm text-slate-600">{{ label }}</span>
      <!-- eslint-disable-next-line vue/no-v-html -->
      <p v-if="hasDesc" class="text-xs text-slate-400 mt-0.5">
        <slot name="desc">
          {{ desc }}
        </slot>
      </p>
    </div>
    <button
      type="button"
      class="toggle"
      :class="{ 'toggle-on': modelValue, 'toggle-violet': violet, 'opacity-50 cursor-not-allowed': disabled }"
      :disabled="disabled"
      role="switch"
      :aria-checked="modelValue"
      @click="toggle"
    >
      <span class="toggle-thumb" />
    </button>
  </div>
</template>
