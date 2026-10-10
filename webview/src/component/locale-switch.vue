<script lang="ts">
import { Component, Prop, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import Dropdown from '@/component/dropdown.vue'

/**
 * 语言切换：下拉列出全部可选语言，当前项高亮。
 * 语言列表来自 locales/messages 目录，新增语言后这里自动出现，无需改动。
 */
@Component({
    components: { Dropdown }
})
class LocaleSwitch extends Vue {
    // 触发按钮是否只显示图标与简称（登录页等空间紧张处）
    @Prop({ type: Boolean, default: false }) readonly compact!: boolean
    // 面板展开方向：页脚处向上，头部处向下
    @Prop({ type: String, default: 'bottom' }) readonly placement!: string

    portal = usePortal()

    // ─── 数据属性 ───
    open = false

    // ─── 方法 ───
    select(code: string) {
        this.portal.setLocale(code)
        this.open = false
    }
}

export default toNative(LocaleSwitch)
</script>

<template>
  <Dropdown v-model:open="open" :placement="placement" align="right" :close-on-click="true" max-height="320px">
    <template #trigger="{ toggle }">
      <button type="button" class="btn btn-ghost text-xs gap-1.5" :class="compact ? 'px-2 py-1' : '!px-2'" :title="$t('切换语言')" @click="toggle">
        <i class="fas fa-globe"></i>
        <span>{{ compact ? portal.localeMeta.short : portal.localeMeta.label }}</span>
      </button>
    </template>

    <button
      v-for="item in portal.locales"
      :key="item.code"
      type="button"
      class="dropdown-item"
      :class="{ 'dropdown-item-active': item.code === portal.locale }"
      @click="select(item.code)"
    >
      <i class="fas fa-check w-4 text-center" :class="item.code === portal.locale ? '' : 'invisible'"></i>
      <span>{{ item.label }}</span>
    </button>
  </Dropdown>
</template>
