<script lang="ts">
import { Component, Prop, Vue, toNative } from 'vue-facing-decorator'

import Combobox from '@/component/combobox.vue'

interface OptionMultiSelectItem {
    value: string
    label: string
    disabled?: boolean
}

@Component({
    components: { Combobox },
    emits: ['update:modelValue']
})
class OptionMultiSelect extends Vue {
    @Prop({ type: Array, default: () => [] }) readonly modelValue!: string[]
    @Prop({ type: Array, default: () => [] }) readonly options!: OptionMultiSelectItem[]
    @Prop({ type: String, default: '请选择...' }) readonly placeholder!: string
    @Prop({ type: String, default: '搜索选项...' }) readonly searchPlaceholder!: string
    @Prop({ type: String, default: '暂无可选项' }) readonly emptyText!: string
    @Prop({ type: String, default: '多选' }) readonly ariaLabel!: string
    @Prop({ type: Boolean, default: false }) readonly disabled!: boolean

    get tagClass() {
        return () => 'bg-primary-50 text-primary-700 border border-primary-200'
    }

    labelFor(value: string) {
        return this.options.find(option => option.value === value)?.label || value
    }

    isOptionDisabled(value: string) {
        return Boolean(this.options.find(option => option.value === value)?.disabled)
    }

    hasClearableSelection(selected: string[]) {
        return selected.some(value => !this.isOptionDisabled(value))
    }

    filteredOptions(query: string) {
        if (!query) return this.options
        return this.options.filter(option =>
            option.label.toLowerCase().includes(query) || option.value.toLowerCase().includes(query)
        )
    }

    updateValue(value: string | string[]) {
        if (this.disabled) return
        this.$emit('update:modelValue', Array.isArray(value) ? value : [])
    }

    toggleOption(value: string) {
        if (this.disabled || this.isOptionDisabled(value)) return
        const next = this.modelValue.includes(value)
            ? this.modelValue.filter(item => item !== value)
            : [...this.modelValue, value]
        this.$emit('update:modelValue', next)
    }
}

export default toNative(OptionMultiSelect)
</script>

<template>
  <Combobox
    :model-value="modelValue"
    multiple
    :allow-custom="false"
    :disabled="disabled"
    :aria-label="ariaLabel"
    :placeholder="placeholder"
    :search-placeholder="searchPlaceholder"
    :tag-class="tagClass"
    :tag-disabled="isOptionDisabled"
    @update:model-value="updateValue"
  >
    <template #tag="{ value }">{{ labelFor(value) }}</template>

    <template #default="{ query, isSelected }">
      <div class="select-list" role="group" :aria-label="ariaLabel">
        <label
          v-for="option in filteredOptions(query)"
          :key="option.value"
          :class="[
            'check-label w-full px-2.5 py-2 rounded-lg border transition-colors',
            isSelected(option.value)
              ? 'bg-primary-50 border-primary-200'
              : (disabled || option.disabled ? 'bg-white border-transparent' : 'bg-white border-transparent hover:bg-slate-50'),
            disabled || option.disabled ? 'opacity-50 cursor-not-allowed' : ''
          ]"
          :aria-disabled="disabled || option.disabled"
        >
          <input
            type="checkbox"
            class="rounded border-slate-300 text-primary-500 focus:ring-primary-500"
            :checked="isSelected(option.value)"
            :disabled="disabled || option.disabled"
            @change="toggleOption(option.value)"
          />
          <span class="text-sm font-medium text-slate-700">{{ option.label }}</span>
        </label>
      </div>
    </template>

    <template #empty="{ query }">
      <div v-if="filteredOptions(query.toLowerCase()).length === 0" class="empty-note">{{ emptyText }}</div>
    </template>

    <template #footer="{ selected, clearAll }">
      <div class="select-footer">
        <span class="text-xs text-slate-400" aria-live="polite">已选 {{ selected.length }} 项</span>
        <button v-if="!disabled && hasClearableSelection(selected)" type="button" class="btn-icon btn-icon-slate" title="清空" aria-label="清空已选项" @click="clearAll">
          <i class="fas fa-xmark text-xs"></i>
        </button>
      </div>
    </template>
  </Combobox>
</template>
