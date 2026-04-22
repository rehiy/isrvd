<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import { APP_ACTIONS_KEY, APP_STATE_KEY } from '@/store/state'
import type { AppActions, AppState } from '@/store/state'

import Dropdown from '@/component/dropdown.vue'

@Component({
    components: { Dropdown }
})
class UserMenu extends Vue {
    @Inject({ from: APP_STATE_KEY }) readonly state!: AppState
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    // ─── 数据属性 ───
    menuOpen = false

    // ─── 方法 ───
    async handleLogout() {
        this.menuOpen = false
        await api.logout()
        this.actions.clearAuth()
    }
}

export default toNative(UserMenu)
</script>

<template>
  <!-- header 认证模式：仅显示用户名，无注销入口 -->
  <div
    v-if="state.authMode === 'header'"
    class="px-2 py-2 text-sm font-medium text-slate-500 flex items-center gap-2 cursor-default select-none"
    :title="state.username || '未登录'"
  >
    <i class="fas fa-user-tie"></i>
    <span class="hidden sm:inline">{{ state.username }}</span>
  </div>

  <!-- jwt 认证模式：用户名 + 下拉注销 -->
  <Dropdown v-else v-model:open="menuOpen" placement="bottom" :close-on-click="true" max-height="320px">
    <template #trigger="{ toggle }">
      <button
        class="px-2 py-2 text-sm font-medium text-slate-500 flex items-center gap-2 rounded-lg hover:bg-slate-100 transition-colors"
        :title="state.username || '未登录'"
        @click="toggle"
      >
        <i class="fas fa-user-tie"></i>
        <span class="hidden sm:inline">{{ state.username }}</span>
        <i class="fas fa-chevron-down text-xs text-slate-400 hidden sm:inline transition-transform duration-200" :class="{ 'rotate-180': menuOpen }"></i>
      </button>
    </template>

    <!-- 注销选项 -->
    <button
      class="w-full flex items-center gap-3 px-4 py-3 text-sm font-medium text-slate-600 hover:text-red-600 hover:bg-red-50 transition-colors"
      :disabled="state.loading"
      @click="handleLogout"
    >
      <i class="fas fa-spinner fa-spin" v-if="state.loading"></i>
      <i class="fas fa-sign-out-alt" v-else></i>
      {{ state.loading ? '处理' : '退出' }}
    </button>
  </Dropdown>
</template>
