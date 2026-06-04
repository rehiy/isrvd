<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import { cycleTheme, getThemeMode, THEME_META, type ThemeMode } from '@/helper/theme'

import Dropdown from '@/component/dropdown.vue'

@Component({
  components: { Dropdown }
})
class UserMenu extends Vue {
  portal = usePortal()

  // ─── 数据属性 ───
  menuOpen = false
  themeMode: ThemeMode = getThemeMode()

  get themeIcon() { return THEME_META[this.themeMode].icon }
  get themeLabel() { return THEME_META[this.themeMode].label }

  // ─── 方法 ───
  toggleTheme() {
    this.themeMode = cycleTheme()
  }

  handleLogout() {
    this.portal.clearAuth()
  }
}

export default toNative(UserMenu)
</script>

<template>
  <!-- header 认证模式：仅显示用户名，无注销入口 -->
  <div v-if="portal.authMode === 'header'" class="px-2 py-2 text-sm font-medium text-slate-500 flex items-center gap-2 cursor-default select-none" :title="portal.username || '未登录'">
    <i class="fas fa-user-tie"></i>
    <span class="hidden sm:inline">{{ portal.username }}</span>
  </div>

  <!-- jwt 认证模式：用户名 + 下拉菜单 -->
  <Dropdown v-else v-model:open="menuOpen" placement="bottom" align="right" :close-on-click="true" max-height="320px">
    <template #trigger="{ toggle }">
      <button class="btn btn-ghost !px-2" :title="portal.username || '未登录'" @click="toggle">
        <i class="fas fa-user-tie"></i>
        <span class="hidden sm:inline">{{ portal.username }}</span>
        <i class="fas fa-chevron-down text-xs text-slate-400 hidden sm:inline transition-transform duration-200" :class="{ 'rotate-180': menuOpen }"></i>
      </button>
    </template>

    <!-- 主题切换：浅色 / 深色 / 跟随系统 -->
    <button class="dropdown-item" :title="`当前：${themeLabel}，点击切换`" @click.stop="toggleTheme">
      <i :class="themeIcon" class="w-4 text-center"></i>
      <span>{{ themeLabel }}</span>
    </button>

    <!-- 账户设置 -->
    <router-link to="/account/password" class="dropdown-item" @click="menuOpen = false">
      <i class="fas fa-lock"></i>
      账号安全
    </router-link>
    <router-link to="/account/passkeys" class="dropdown-item" @click="menuOpen = false">
      <i class="fas fa-fingerprint"></i>
      Passkey
    </router-link>
    <router-link to="/account/apikey" class="dropdown-item" @click="menuOpen = false">
      <i class="fas fa-key"></i>
      API Key
    </router-link>

    <!-- 分割线 -->
    <div class="border-t border-slate-100 my-1"></div>

    <!-- 注销选项 -->
    <button class="dropdown-item-danger" @click="handleLogout">
      <i class="fas fa-sign-out-alt"></i>
      退出
    </button>
  </Dropdown>
</template>
