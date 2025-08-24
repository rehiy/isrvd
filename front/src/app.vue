<script setup>
import { onMounted, provide } from 'vue'

import { initProvider, APP_STATE_KEY, APP_ACTIONS_KEY } from '@/stores/state.js'

import NavigationBar from '@/layouts/navigation.vue'

import AuthLogin from '@/components/auth/login.vue'
import FileManager from '@/components/file-manager/index.vue'
import NotificationManager from '@/components/notification.vue'

const { state, actions } = initProvider()

// 提供状态和动作给子组件
provide(APP_STATE_KEY, state)
provide(APP_ACTIONS_KEY, actions)

onMounted(() => {
  const savedToken = localStorage.getItem('app-token')
  const savedUsername = localStorage.getItem('app-username')

  if (savedToken && savedUsername) {
    actions.setAuth({ token: savedToken, user: savedUsername })
    actions.loadFiles()
  }
})
</script>

<template>
  <template v-if="state.user">
    <NavigationBar />
    <div class="container-fluid">
      <FileManager />
    </div>
  </template>

  <AuthLogin v-else />

  <NotificationManager />
</template>
