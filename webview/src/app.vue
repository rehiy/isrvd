<script setup>
import { onMounted, provide } from 'vue'

import { initProvider, APP_STATE_KEY, APP_ACTIONS_KEY } from '@/store/state.js'

import NavigationBar from '@/component/navigation.vue'
import NotificationManager from '@/component/notification.vue'

import AuthLogin from '@/views/login.vue'

const { state, actions } = initProvider()

// 提供状态和动作给子组件
provide(APP_STATE_KEY, state)
provide(APP_ACTIONS_KEY, actions)

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
      <NavigationBar />
      <main class="px-6 py-6">
        <router-view />
      </main>
    </template>

    <AuthLogin v-else />

    <NotificationManager />
  </div>
</template>
