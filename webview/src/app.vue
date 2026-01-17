<script setup>
import { onMounted, provide } from 'vue'

import { initProvider, APP_STATE_KEY, APP_ACTIONS_KEY } from '@/store/state.js'

import NavigationBar from '@/layout/navigation.vue'
import NotificationManager from '@/layout/notification.vue'

import AuthLogin from '@/layout/login.vue'
import FileManager from '@/layout/file-manager/index.vue'

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
  <template v-if="state.username">
    <NavigationBar />
    <div class="container-fluid">
      <FileManager />
    </div>
  </template>

  <AuthLogin v-else />

  <NotificationManager />
</template>
