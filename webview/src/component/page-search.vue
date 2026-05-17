<script lang="ts">
import { Component, Prop, Ref, Vue, toNative } from 'vue-facing-decorator'

import { bindTypeToSearchFocus } from '@/helper/utils'

@Component({
    emits: ['update:modelValue']
})
class PageSearch extends Vue {
    @Prop({ type: String, default: '' }) readonly modelValue!: string
    @Prop({ type: String, default: '搜索...' }) readonly placeholder!: string
    @Prop({ type: String, default: '' }) readonly searchKey!: string
    @Prop({ type: String, default: 'w-48' }) readonly widthClass!: string
    @Prop({ type: String, default: 'primary' }) readonly focusColor!: string
    @Prop({ type: Boolean, default: false }) readonly typeToSearch!: boolean

    @Ref readonly inputRef!: HTMLInputElement

    private unbindTypeToSearchFocus: (() => void) | null = null

    get value() {
        return this.modelValue
    }

    set value(value: string) {
        this.$emit('update:modelValue', value)
    }

    get inputClass() {
        const focusClass = {
            primary: 'focus:ring-primary-500',
            blue: 'focus:ring-blue-500',
            indigo: 'focus:ring-indigo-500',
            violet: 'focus:ring-violet-500',
            purple: 'focus:ring-purple-500',
            cyan: 'focus:ring-cyan-500',
            emerald: 'focus:ring-emerald-500',
            amber: 'focus:ring-amber-500',
            rose: 'focus:ring-rose-500'
        }[this.focusColor] || 'focus:ring-primary-500'
        return `${this.widthClass} h-9 pl-8 pr-3 text-xs border border-slate-200 rounded-lg focus:outline-none focus:ring-2 ${focusClass} focus:border-transparent`
    }

    get searchAttr() {
        return this.searchKey || undefined
    }

    mounted() {
        if (!this.typeToSearch || !this.searchKey) return
        this.unbindTypeToSearchFocus = bindTypeToSearchFocus(() =>
            Array.from(document.querySelectorAll(`[data-page-search="${this.searchKey}"]`)) as HTMLInputElement[]
        )
    }

    unmounted() {
        this.unbindTypeToSearchFocus?.()
        this.unbindTypeToSearchFocus = null
    }
}

export default toNative(PageSearch)
</script>

<template>
  <div class="relative">
    <input ref="inputRef" v-model="value" :data-page-search="searchAttr" type="text" :placeholder="placeholder" :class="inputClass" />
    <i class="fas fa-search absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-400 text-xs"></i>
  </div>
</template>
