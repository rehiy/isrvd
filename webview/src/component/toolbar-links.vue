<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import Dropdown from '@/component/dropdown.vue'

import { usePortal } from '@/stores'

@Component({
    components: { Dropdown }
})
class ToolbarLinks extends Vue {
    portal = usePortal()

    menuOpen = false
}

export default toNative(ToolbarLinks)
</script>

<template>
  <!-- 桌面端：横向按钮列表 -->
  <div class="hidden md:flex items-center gap-2 overflow-x-auto ml-auto mr-2">
    <a
      v-for="link in portal.toolbarLinks"
      :key="link.url"
      :href="link.url"
      target="_blank"
      rel="noopener noreferrer"
      class="btn btn-ghost gap-2 whitespace-nowrap"
    >
      <i v-if="link.icon" :class="link.icon.includes(' ') ? link.icon : `fas ${link.icon}`"></i>
      <span class="whitespace-nowrap">{{ link.label }}</span>
    </a>
  </div>

  <!-- 手机端：图标下拉菜单 -->
  <div class="flex md:hidden items-center ml-auto mr-2">
    <Dropdown v-model:open="menuOpen" placement="bottom" align="right" :close-on-click="true" max-height="320px">
      <template #trigger="{ toggle }">
        <button
          class="btn-icon"
          title="快捷链接"
          @click="toggle"
        >
          <i class="fas fa-star"></i>
        </button>
      </template>

      <template v-if="portal.toolbarLinks.length === 0">
        <div class="px-4 py-3 text-sm text-slate-400">无快捷链接</div>
      </template>

      <a
        v-for="link in portal.toolbarLinks"
        :key="link.url"
        :href="link.url"
        target="_blank"
        rel="noopener noreferrer"
        class="dropdown-item"
        @click="menuOpen = false"
      >
        <i v-if="link.icon" :class="link.icon.includes(' ') ? link.icon : `fas ${link.icon}`"></i>
        <span>{{ link.label }}</span>
      </a>
    </Dropdown>
  </div>
</template>
