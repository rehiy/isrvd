<template>
  <template v-if="state.user">
    <NavigationBar />
    <div class="container-fluid">
      <BreadcrumbNav />
      <FileActions />
      <FileExplorer />
    </div>
  </template>

  <LoginForm v-else />

  <NotificationManager />
</template>

<script setup>
import { onMounted, provide } from 'vue'
import { initProvider, APP_STATE_KEY, APP_ACTIONS_KEY } from '@/stores/state.js'
import LoginForm from '@/components/auth/auth.vue'
import NotificationManager from '@/components/base/notification.vue'
import NavigationBar from '@/layouts/navigation.vue'
import BreadcrumbNav from '@/layouts/breadcrumb.vue'
import FileActions from '@/components/file-manager/file-actions.vue'
import FileExplorer from '@/components/file-manager/file-explorer.vue'

const { state, actions } = initProvider()

// 提供状态和动作给子组件
provide(APP_STATE_KEY, state)
provide(APP_ACTIONS_KEY, actions)

onMounted(() => {
  // 检查本地存储的认证信息
  const savedToken = localStorage.getItem('file-manager-token')
  const savedUser = localStorage.getItem('file-manager-user')

  if (savedToken && savedUser) {
    actions.setAuth({ token: savedToken, user: savedUser })
    // 认证状态恢复后立即加载文件
    actions.loadFiles()
  }
})
</script>
