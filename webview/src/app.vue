<script setup>
import { onMounted, provide, ref } from 'vue'

import { APP_ACTIONS_KEY, APP_STATE_KEY, initProvider } from '@/store/state.js'

import NavigationBar from '@/component/navigation.vue'
import NotificationManager from '@/component/notification.vue'

import AuthLogin from '@/views/login.vue'

const { state, actions } = initProvider()

// 提供状态和动作给子组件
provide(APP_STATE_KEY, state)
provide(APP_ACTIONS_KEY, actions)

// 侧边栏折叠状态
const sidebarCollapsed = ref(false)
provide('sidebarCollapsed', sidebarCollapsed)

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
      <NavigationBar v-model:collapsed="sidebarCollapsed" />
      <main 
        class="px-6 py-6 transition-all duration-300"
        :class="sidebarCollapsed ? 'ml-16' : 'ml-64'"
      >
        <router-view />
      </main>
    </template>

    <AuthLogin v-else />

    <NotificationManager />
  </div>
</template>
