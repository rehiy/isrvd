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

    // ─── 侦听器 ───
    @Watch('state.username')
    onUsernameChange(username: string, oldUsername: string) {
        if (username && !oldUsername) {
            this.loadServiceAvailability()
        }
    }

    // ─── 生命周期 ───
    mounted() {
        const token = localStorage.getItem('app-token')
        const username = localStorage.getItem('app-username')

        if (token && username) {
            this.actions.setAuth({ token, username })
            this.loadServiceAvailability()
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
        <div class="flex items-center space-x-3">
          <div class="w-8 h-8 rounded-full bg-primary-400 flex items-center justify-center flex-shrink-0">
            <i class="fas fa-user text-white text-sm"></i>
          </div>
          <div class="hidden sm:block text-sm font-medium text-slate-700">{{ state.username }}</div>
          <AuthLogout />
        </div>
      </header>

      <NavigationBar ref="navigationRef" v-model:collapsed="sidebarCollapsed" />
      <main class="px-4 py-6 pt-20 transition-all duration-300" :class="sidebarCollapsed ? 'lg:ml-16' : 'lg:ml-64'">
        <router-view />
      </main>

      <PageAgent v-if="state.serviceAvailability.agent" />
    </template>

    <AuthLogin v-else />

    <NotificationManager />
    <ConfirmModal />
  </div>
</template>
