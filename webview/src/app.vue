<script lang="ts">
import { Component, Provide, Ref, Vue, Watch, toNative } from 'vue-facing-decorator'

import { APP_ACTIONS_KEY, APP_STATE_KEY, initProvider } from '@/store/state'

import ConfirmModal from '@/component/confirm.vue'
import NavigationBar from '@/component/navigation.vue'
import NotificationManager from '@/component/notification.vue'
import PageAgent from '@/component/page-agent.vue'

import AuthLogin from '@/views/login.vue'
import AuthLogout from '@/views/logout.vue'

import { fetchServiceProbe } from '@/service/probe'
import api from '@/service/api'
import router from '@/router'

const { state, actions } = initProvider()

@Component({
    components: { ConfirmModal, NavigationBar, NotificationManager, PageAgent, AuthLogin, AuthLogout }
})
class App extends Vue {
    // ─── 数据属性 ───
    @Provide(APP_STATE_KEY) state = state
    @Provide(APP_ACTIONS_KEY) actions = actions
    sidebarCollapsed = false

    // ─── Refs ───
    @Ref readonly navigationRef!: InstanceType<typeof NavigationBar>

    toggleMobileMenu() {
        if (this.navigationRef) {
            this.navigationRef.toggleMobileSidebar()
        }
    }

    async loadServiceAvailability() {
        try {
            const availability = await fetchServiceProbe()
            this.actions.updateServiceAvailability(availability)
        } catch (e) {
            console.warn('Failed to load service probe:', e)
        }
    }

    async loadMe() {
        try {
            const res = await api.getMe()
            const member = res?.payload?.member
            if (member) {
                this.actions.setPermissions({
                    isPrimary: member.isPrimary,
                    permissions: member.permissions || {}
                })
                // 权限加载完成后重新触发路由守卫（处理刷新页面场景）
                router.replace(router.currentRoute.value.fullPath).catch(() => {})
            }
        } catch (e) {
            console.warn('Failed to load user permissions:', e)
        }
    }

    // ─── 侦听器 ───
    @Watch('state.username')
    onUsernameChange(username: string, oldUsername: string) {
        if (username && !oldUsername) {
            this.loadServiceAvailability()
            this.loadMe()
        }
    }

    // ─── 生命周期 ───
    async mounted() {
        try {
            const res = await api.authInfo()
            const mode = res?.payload?.mode
            if (mode === 'header') {
                // header 认证模式：直接使用代理注入的用户名，无需登录
                const username = res.payload?.username || ''
                if (username) {
                    this.actions.setAuth({ authMode: 'header', token: '', username })
                }
            } else {
                // jwt 认证模式：从 localStorage 恢复登录状态
                const token = localStorage.getItem('app-token')
                const username = localStorage.getItem('app-username')
                if (token && username) {
                    this.actions.setAuth({ authMode: 'jwt', token, username })
                }
            }
        } catch (e) {
            console.warn('Failed to load auth info:', e)
        }
    }
}

export default toNative(App)
</script>

<template>
  <div class="min-h-screen bg-slate-50">
    <template v-if="state.username">
      <!-- 移动端顶部菜单栏 -->
      <header 
        class="fixed top-0 left-0 right-0 h-16 bg-white border-b border-slate-200 z-40 flex items-center justify-between px-4 transition-all duration-300"
        :class="sidebarCollapsed ? 'lg:left-16' : 'lg:left-64'"
      >
        <!-- 移动端菜单切换按钮 -->
        <button
          @click="toggleMobileMenu"
          class="lg:hidden p-2 rounded-lg hover:bg-slate-100 transition-colors"
        >
          <i class="fas fa-bars text-slate-600"></i>
        </button>
        
        <!-- 工具栏按钮区域 -->
        <div class="flex items-center gap-2 overflow-x-auto flex-1 mx-2">
        </div>
        
        <!-- 用户信息 -->
        <div class="flex items-center gap-1">
          <PageAgent v-if="state.serviceAvailability.agent" />
          <div class="hidden sm:block w-px h-5 bg-slate-200 mx-1" v-if="state.serviceAvailability.agent"></div>
          <div
            class="px-2 py-2 text-sm font-medium text-slate-500 flex items-center gap-2 cursor-default select-none"
            :title="state.username"
          >
            <i class="fas fa-user-tie"></i>
            <span class="hidden sm:inline">{{ state.username }}</span>
          </div>
          <div class="hidden sm:block w-px h-5 bg-slate-200 mx-1" v-if="state.authMode !== 'header'"></div>
          <AuthLogout v-if="state.authMode !== 'header'" />
        </div>
      </header>

      <NavigationBar ref="navigationRef" v-model:collapsed="sidebarCollapsed" />
      <main class="px-4 py-6 pt-20 transition-all duration-300" :class="sidebarCollapsed ? 'lg:ml-16' : 'lg:ml-64'">
        <router-view />
      </main>
    </template>

    <AuthLogin v-else />

    <NotificationManager />
    <ConfirmModal />
  </div>
</template>
