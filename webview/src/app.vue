<script setup>
import { onMounted, provide, ref } from 'vue'
import { useRoute } from 'vue-router'

import { APP_ACTIONS_KEY, APP_STATE_KEY, initProvider } from '@/store/state.js'

import ConfirmModal from '@/component/confirm.vue'
import NavigationBar from '@/component/navigation.vue'
import NotificationManager from '@/component/notification.vue'

import AuthLogin from '@/views/login.vue'
import AuthLogout from '@/views/logout.vue'

const { state, actions } = initProvider()

// 提供状态和动作给子组件
provide(APP_STATE_KEY, state)
provide(APP_ACTIONS_KEY, actions)

// 侧边栏折叠状态
const sidebarCollapsed = ref(false)
provide('sidebarCollapsed', sidebarCollapsed)

// 工具栏按钮管理
const toolbarButtons = ref([])
const route = useRoute()

// 清空按钮（路由变化时自动调用）
const clearToolbar = () => {
  toolbarButtons.value = []
}

// 注册按钮
const registerToolbarButton = (button) => {
  const existing = toolbarButtons.value.find(b => b.id === button.id)
  if (existing) {
    Object.assign(existing, button)
  } else {
    toolbarButtons.value.push(button)
  }
}

// 注册多个按钮
const registerToolbarButtons = (buttons) => {
  buttons.forEach(btn => registerToolbarButton(btn))
}

// 提供给子组件使用
provide('toolbar', {
  buttons: toolbarButtons,
  clear: clearToolbar,
  register: registerToolbarButton,
  registerAll: registerToolbarButtons
})

// 路由变化时清空按钮
const clearToolbarOnRouteChange = () => {
  clearToolbar()
}

onMounted(() => {
  const token = localStorage.getItem('app-token')
  const username = localStorage.getItem('app-username')

  if (token && username) {
    actions.setAuth({ token, username })
  }
})
</script>

<template>
  <div class="min-h-screen bg-slate-50">
    <template v-if="state.username">
      <!-- 顶部菜单栏 -->
      <header 
        class="fixed top-0 right-0 h-16 bg-white border-b border-slate-200 z-40 flex items-center justify-between px-6 transition-all duration-300"
        :class="sidebarCollapsed ? 'left-16' : 'left-64'"
      >
        <!-- 工具栏按钮区域 -->
        <div class="flex items-center gap-2">
          <button
            v-for="btn in toolbarButtons"
            :key="btn.id"
            @click="btn.onClick"
            :class="btn.variant === 'primary' ? 'btn-primary' : 'btn-secondary'"
            class="text-sm py-2"
            :disabled="btn.loading"
          >
            <i v-if="btn.loading" class="fas fa-spinner fa-spin mr-1.5"></i>
            <i v-else-if="btn.icon" :class="btn.icon" class="mr-1.5"></i>
            {{ btn.label }}
          </button>
        </div>
        
        <!-- 用户信息 -->
        <div class="flex items-center space-x-3">
          <div class="w-8 h-8 rounded-full bg-primary-400 flex items-center justify-center flex-shrink-0">
            <i class="fas fa-user text-white text-sm"></i>
          </div>
          <div class="text-sm font-medium text-slate-700">{{ state.username }}</div>
          <AuthLogout />
        </div>
      </header>

      <NavigationBar v-model:collapsed="sidebarCollapsed" />
      <main 
        class="px-6 py-6 pt-20 transition-all duration-300"
        :class="sidebarCollapsed ? 'ml-16' : 'ml-64'"
      >
<router-view @vue:mounted="clearToolbarOnRouteChange" />
      </main>
    </template>

    <AuthLogin v-else />

    <NotificationManager />
    <ConfirmModal />
  </div>
</template>
