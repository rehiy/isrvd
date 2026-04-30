<script lang="ts">
import { Component, Prop, Vue, Watch, toNative } from 'vue-facing-decorator'

import Dropdown from '@/component/dropdown.vue'

@Component({
    components: { Dropdown },
    emits: ['update:modelValue']
})
class Combobox extends Vue {
    @Prop({ type: [String, Array], default: '' }) readonly modelValue!: string | string[]
    @Prop({ type: Boolean, default: false }) readonly multiple!: boolean
    @Prop({ type: String, default: '' }) readonly placeholder!: string
    @Prop({ type: String, default: '' }) readonly searchPlaceholder!: string
    @Prop({ type: Boolean, default: false }) readonly disabled!: boolean
    @Prop({ type: String, default: '320px' }) readonly maxHeight!: string
    @Prop({ type: String, default: 'match' }) readonly align!: 'left' | 'right' | 'match' | 'match-right'
    @Prop({ type: Function, default: () => 'bg-slate-100 text-slate-700 border border-slate-200' })
    readonly tagClass!: (val: string) => string

    // ─── 数据属性 ───
    dropdownOpen = false
    searchQuery = ''
    justSelected = false

    // ─── 计算属性 ───
    get selected(): string[] {
        if (!this.multiple) return []
        return Array.isArray(this.modelValue) ? this.modelValue : []
    }

    get singleValue(): string {
        if (this.multiple) return ''
        return typeof this.modelValue === 'string' ? this.modelValue : ''
    }

    get effectivePlaceholder(): string {
        if (!this.multiple) return this.placeholder
        if (this.selected.length === 0) return this.placeholder
        return this.searchPlaceholder || '搜索...'
    }

    // ─── 生命周期 ───
    mounted() {
        if (!this.multiple) this.searchQuery = this.singleValue
    }

    // ─── 监听器（仅单选需要回填） ───
    @Watch('modelValue')
    onModelValueChange() {
        if (this.multiple) return
        if (this.singleValue !== this.searchQuery) this.searchQuery = this.singleValue
    }

    @Watch('dropdownOpen')
    onDropdownOpenChange(open: boolean) {
        if (this.multiple) {
            if (!open) this.searchQuery = ''
            return
        }
        if (open) {
            this.searchQuery = ''
        } else if (!this.justSelected) {
            this.searchQuery = this.singleValue
        }
        this.justSelected = false
    }

    // ─── 方法 ───
    isSelected(value: string) {
        return this.multiple ? this.selected.includes(value) : this.singleValue === value
    }

    select(value: string) {
        if (this.multiple) {
            const v = value.trim()
            if (!v) return
            const next = this.selected.includes(v)
                ? this.selected.filter(x => x !== v)
                : [...this.selected, v]
            this.$emit('update:modelValue', next)
            this.searchQuery = ''
            return
        }
        this.$emit('update:modelValue', value)
        this.searchQuery = value
        this.justSelected = true
        this.dropdownOpen = false
    }

    removeTag(value: string) {
        if (!this.multiple) return
        this.$emit('update:modelValue', this.selected.filter(x => x !== value))
    }

    clearAll() {
        if (!this.multiple) return
        this.$emit('update:modelValue', [])
    }

    handleInput() {
        if (!this.multiple) this.$emit('update:modelValue', this.searchQuery)
        if (!this.dropdownOpen) this.dropdownOpen = true
    }

    handleEnter() {
        const v = this.searchQuery.trim()
        if (this.multiple) {
            if (!v) return
            if (!this.selected.includes(v)) {
                this.$emit('update:modelValue', [...this.selected, v])
            }
            this.searchQuery = ''
            return
        }
        this.justSelected = true
        this.select(v)
    }

    focusInput() {
        ;(this.$refs.inputRef as HTMLInputElement)?.focus()
        this.dropdownOpen = true
    }
}

export default toNative(Combobox)
</script>

<template>
  <Dropdown v-model:open="dropdownOpen" :max-height="maxHeight" :align="align">
    <template #trigger="{ open }">
      <div
        class="input min-h-[46px] cursor-text flex items-center gap-2"
        :class="[open ? '!border-primary-400' : '', multiple ? 'flex-wrap gap-1.5' : '']"
        @click="focusInput"
      >
        <template v-if="multiple">
          <span
            v-for="tag in selected"
            :key="tag"
            :class="['inline-flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs font-medium transition-all', tagClass(tag)]"
          >
            {{ tag }}
            <button
              type="button"
              class="w-3.5 h-3.5 flex items-center justify-center rounded-full hover:bg-black/10 transition-colors"
              @click.stop="removeTag(tag)"
            >
              <i class="fas fa-times text-[8px]"></i>
            </button>
          </span>
        </template>

        <input
          ref="inputRef"
          v-model="searchQuery"
          type="text"
          class="flex-1 min-w-[80px] border-0 outline-none bg-transparent text-sm text-slate-700 placeholder:text-slate-400 p-0 focus:ring-0 focus:border-0 focus:shadow-none"
          :placeholder="effectivePlaceholder"
          :disabled="disabled"
          @focus="dropdownOpen = true"
          @input="handleInput"
          @keydown.enter.prevent="handleEnter"
        />

        <i
          v-if="!multiple"
          :class="['fas fa-chevron-down text-slate-400 text-xs transition-transform duration-200', open ? 'rotate-180' : '']"
        ></i>
      </div>
    </template>

    <template #search-hint>
      <div v-if="searchQuery.trim()" class="px-3 py-2 bg-slate-50 border-b border-slate-100 flex items-center justify-between">
        <span class="text-xs text-slate-500">
          <template v-if="multiple">按 Enter 添加: </template>
          <template v-else>使用: </template>
          <code class="bg-slate-200 px-1.5 py-0.5 rounded text-slate-700">{{ searchQuery.trim() }}</code>
        </span>
        <slot name="hint-extra" :query="searchQuery.trim()" />
      </div>
    </template>

    <template #default>
      <slot
        :query="searchQuery.trim().toLowerCase()"
        :select="select"
        :selected="selected"
        :is-selected="isSelected"
      />
    </template>

    <template #empty>
      <slot name="empty" :query="searchQuery.trim()" />
    </template>

    <template #footer>
      <slot name="footer" :selected="selected" :clear-all="clearAll" />
    </template>
  </Dropdown>
</template>
