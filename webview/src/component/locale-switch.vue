<script lang="ts">
import { Component, Prop, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import Dropdown from '@/component/dropdown.vue'

/**
 * 语言切换：头部与登录页共用的独立下拉，触发按钮只显示地球图标。
 * 语言列表来自 locales/messages 目录，新增语言后自动出现，无需改动。
 */
@Component({
    components: { Dropdown }
})
class LocaleSwitch extends Vue {
    // 面板展开方向：页脚处向上，头部处向下
    @Prop({ type: String, default: 'bottom' }) readonly placement!: string

    portal = usePortal()

    // ─── 数据属性 ───
    open = false

    // 触发按钮的提示：当前语言，以该语言自己的名称呈现
    get currentLabel() { return this.portal.localeMeta.label }

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
      <button type="button" class="btn btn-ghost !px-2" :title="$t('切换语言') + ' · ' + currentLabel" @click="toggle">
        <i class="fas fa-globe"></i>
        <!-- 与其它头部下拉（节点切换、用户菜单）保持一致的下拉指示 -->
        <i class="fas fa-chevron-down text-xs text-slate-400 transition-transform duration-200" :class="{ 'rotate-180': open }"></i>
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
