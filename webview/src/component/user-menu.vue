<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import { cycleTheme, getThemeMode, THEME_META, type ThemeMode } from '@/helper/theme'

import Dropdown from '@/component/dropdown.vue'
import LocaleSwitch from '@/component/locale-switch.vue'

@Component({
  components: { Dropdown, LocaleSwitch }
})
class UserMenu extends Vue {
  portal = usePortal()

  // ─── 数据属性 ───
  menuOpen = false
  themeMode: ThemeMode = getThemeMode()

  get themeIcon() { return THEME_META[this.themeMode].icon }
  get themeLabel() { return this.$t(THEME_META[this.themeMode].label) }

  // 两种入口的鼠标提示：展示当前取值并说明点击可切换
  get themeTitle() { return this.$t('当前：') + this.themeLabel + this.$t('，点击切换') }

  // Passkey 入口：功能已启用且具备查看权限时显示
  get showPasskeyEntry() { return this.portal.passkeyEnabled && this.portal.hasPerm('GET /api/account/passkey/credentials') }

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
  <div v-if="portal.authMode === 'header'" class="flex items-center gap-1">
    <div class="px-2 py-2 text-sm font-medium text-slate-500 flex items-center gap-2 cursor-default select-none" :title="portal.username || $t('未登录')">
      <i class="fas fa-user-tie"></i>
      <span class="hidden sm:inline">{{ portal.username }}</span>
    </div>
    <!-- 语言切换：该模式没有下拉菜单，直接提供入口 -->
    <LocaleSwitch />
  </div>

  <!-- jwt 认证模式：用户名 + 下拉菜单 -->
  <Dropdown v-else v-model:open="menuOpen" placement="bottom" align="right" :close-on-click="true" max-height="320px">
    <template #trigger="{ toggle }">
      <button class="btn btn-ghost !px-2" :title="portal.username || $t('未登录')" @click="toggle">
        <i class="fas fa-user-tie"></i>
        <span class="hidden sm:inline">{{ portal.username }}</span>
        <i class="fas fa-chevron-down text-xs text-slate-400 hidden sm:inline transition-transform duration-200" :class="{ 'rotate-180': menuOpen }"></i>
      </button>
    </template>

    <!-- 主题切换：浅色 / 深色 / 跟随系统 -->
    <button class="dropdown-item" :title="themeTitle" @click.stop="toggleTheme">
      <i :class="themeIcon" class="w-4 text-center"></i>
      <span>{{ themeLabel }}</span>
    </button>

    <!-- 语言切换：语言平铺在菜单里，避免下拉套下拉；列表来自 locales/messages，新增语言自动出现 -->
    <div class="border-t border-slate-100 my-1"></div>
    <button
      v-for="item in portal.locales"
      :key="item.code"
      class="dropdown-item"
      :class="{ 'dropdown-item-active': item.code === portal.locale }"
      @click="portal.setLocale(item.code)"
    >
      <i class="fas w-4 text-center" :class="item.code === portal.locale ? 'fa-check' : 'fa-globe text-slate-300'"></i>
      <span>{{ item.label }}</span>
    </button>
    <div class="border-t border-slate-100 my-1"></div>

    <!-- 账户设置 -->
    <router-link to="/account/password" class="dropdown-item" @click="menuOpen = false">
      <i class="fas fa-lock"></i>
      {{ $t('账号安全') }}
    </router-link>
    <router-link v-if="showPasskeyEntry" to="/account/passkeys" class="dropdown-item" @click="menuOpen = false">
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
      {{ $t('退出') }}
    </button>
  </Dropdown>
</template>
