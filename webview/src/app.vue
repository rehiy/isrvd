<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'

import ConfirmModal from '@/component/confirm.vue'
import NavigationBar from '@/component/navigation.vue'
import NotificationManager from '@/component/notification.vue'
import PageAgent from '@/component/page-agent.vue'
import ToolbarLinks from '@/component/toolbar-links.vue'
import UserMenu from '@/component/user-menu.vue'

import AuthLogin from '@/views/account/login.vue'

@Component({
    components: { ConfirmModal, NavigationBar, NotificationManager, PageAgent, ToolbarLinks, UserMenu, AuthLogin }
})
class App extends Vue {
    portal = usePortal()
    sidebarCollapsed = false

    @Ref readonly navigationRef!: InstanceType<typeof NavigationBar>

    toggleMobileMenu() {
        this.navigationRef?.toggleMobileSidebar()
    }

    // 处理 OIDC 回调（query 参数中的 oidc_code / oidc_error）
    // 放在 App 入口而非 Login 组件，确保已登录状态下也能处理。
    async handleOIDCCallback() {
        const params = new URLSearchParams(window.location.search)
        const error = params.get('oidc_error')
        const code = params.get('oidc_code')
        if (!error && !code) return

        // 立即清除 query，避免刷新时重复处理
        window.history.replaceState({}, document.title, window.location.pathname + window.location.hash)

        if (error) {
            this.portal.showNotification('error', error)
            return
        }

        try {
            const { payload } = await api.accountOIDCExchange({ code: code ?? '' })
            if (!payload) return
            this.portal.setAuth({ authMode: 'jwt', ...payload })
            await this.portal.initialize()
        } catch {
            this.portal.showNotification('error', 'OIDC 登录失败，请重试')
        }
    }

    async mounted() {
        // OIDC 回调需在 initialize 之前处理，确保 token 已就位
        await this.handleOIDCCallback()
        await this.portal.initialize()
    }
}

export default toNative(App)
</script>

<template>
  <div class="min-h-screen bg-slate-50">
    <!-- 初始化加载状态 -->
    <div v-if="!portal.initialized" class="flex items-center justify-center min-h-screen">
      <div class="flex flex-col items-center gap-4">
        <div class="w-12 h-12 border-4 border-slate-200 border-t-blue-500 rounded-full animate-spin"></div>
        <span class="text-slate-500 text-sm">正在初始化...</span>
      </div>
    </div>

    <!-- 主内容 -->
    <template v-else-if="portal.username">
      <!-- 移动端顶部菜单栏 -->
      <header
        class="fixed top-0 left-0 right-0 h-16 bg-white/80 backdrop-blur-xl border-b border-slate-200/50 z-40 flex items-center justify-between px-4 transition-all duration-300"
        :class="sidebarCollapsed ? 'lg:left-16' : 'lg:left-64'"
      >
        <!-- 移动端菜单切换按钮 -->
        <button class="btn-icon lg:hidden" @click="toggleMobileMenu">
          <i class="fas fa-bars text-slate-600"></i>
        </button>

        <!-- 工具栏按钮区域 -->
        <ToolbarLinks />

        <!-- 用户信息 -->
        <div class="flex items-center gap-1">
          <PageAgent v-if="portal.hasPerm('agent')" />
          <div v-if="portal.hasPerm('agent')" class="hidden sm:block w-px h-5 bg-slate-200 mx-1"></div>
          <UserMenu />
        </div>
      </header>

      <NavigationBar ref="navigationRef" v-model:collapsed="sidebarCollapsed" />
      <main class="px-4 py-6 pt-20 transition-all duration-300" :class="sidebarCollapsed ? 'lg:ml-16' : 'lg:ml-64'">
        <router-view />
      </main>
    </template>

    <!-- 登录页面 -->
    <AuthLogin v-else />

    <NotificationManager />
    <ConfirmModal />
  </div>
</template>
