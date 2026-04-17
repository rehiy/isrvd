<script lang="ts">
import { Component, Prop, Vue, Watch, toNative } from 'vue-facing-decorator'

import Dropdown from '@/component/dropdown.vue'

@Component({
    components: { Dropdown },
    emits: ['update:modelValue']
})
class Combobox extends Vue {
    @Prop({ type: String, default: '' }) readonly modelValue!: string
    @Prop({ type: String, default: '' }) readonly placeholder!: string
    @Prop({ type: Boolean, default: false }) readonly disabled!: boolean
    @Prop({ type: String, default: '320px' }) readonly maxHeight!: string

    // ─── 数据属性 ───
    dropdownOpen = false
    searchQuery = ''
    justSelected = false

    // ─── 生命周期 ───
    mounted() {
        this.searchQuery = this.modelValue
    }

    // ─── 监听器 ───
    @Watch('modelValue')
    onModelValueChange(val: string) {
        if (val !== this.searchQuery) this.searchQuery = val
    }

    @Watch('dropdownOpen')
    onDropdownOpenChange(open: boolean) {
        if (open) {
            this.searchQuery = ''
        } else if (!this.justSelected) {
            this.searchQuery = this.modelValue
        }
        this.justSelected = false
    }

    // ─── 方法 ───
    select(value: string) {
        this.$emit('update:modelValue', value)
        this.searchQuery = value
        this.justSelected = true
        this.dropdownOpen = false
    }

    handleInput() {
        this.$emit('update:modelValue', this.searchQuery)
        if (!this.dropdownOpen) this.dropdownOpen = true
    }

    handleEnter() {
        this.justSelected = true
        this.select(this.searchQuery.trim())
    }
}

export default toNative(Combobox)
</script>

<template>
  <Dropdown v-model:open="dropdownOpen" :max-height="maxHeight">
    <template #trigger="{ open }">
      <div
        class="input min-h-[42px] !px-3 !py-2 cursor-text flex items-center gap-2"
        :class="open ? '!border-primary-400' : ''"
        @click="($refs.inputRef as HTMLInputElement)?.focus(); dropdownOpen = true"
      >
        <input
          ref="inputRef"
          v-model="searchQuery"
          type="text"
          class="flex-1 min-w-[80px] border-0 outline-none bg-transparent text-sm text-slate-700 placeholder:text-slate-400 p-0 focus:ring-0 focus:border-0 focus:shadow-none"
          :placeholder="placeholder"
          :disabled="disabled"
          @focus="dropdownOpen = true"
          @input="handleInput"
          @keydown.enter.prevent="handleEnter"
        />
        <i :class="['fas fa-chevron-down text-slate-400 text-xs transition-transform duration-200', open ? 'rotate-180' : '']"></i>
      </div>
    </template>

    <template #search-hint>
      <div v-if="searchQuery.trim()" class="px-3 py-2 bg-slate-50 border-b border-slate-100 flex items-center justify-between">
        <span class="text-xs text-slate-500">使用: <code class="bg-slate-200 px-1.5 py-0.5 rounded text-slate-700">{{ searchQuery.trim() }}</code></span>
        <slot name="hint-extra" :query="searchQuery.trim()" />
      </div>
    </template>

    <template #default>
      <slot :query="searchQuery.trim().toLowerCase()" :select="select" />
    </template>

    <template #empty>
      <slot name="empty" :query="searchQuery.trim()" />
    </template>

    <template #footer>
      <slot name="footer" />
    </template>
  </Dropdown>
</template>
