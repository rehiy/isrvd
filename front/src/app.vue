<template>
  <template v-if="state.user">
    <NavigationBar />
    <div class="container-fluid">
      <BreadcrumbNav />
      <ActionButtons />
      <FileIndex />
    </div>
  </template>

  <LoginForm v-else />

  <NotificationManager />
</template>

<script>
import { defineComponent, onMounted, provide } from 'vue'
import { initProvider, APP_STATE_KEY, APP_ACTIONS_KEY } from './helpers/state.js'
import LoginForm from './components/auth.vue'
import NotificationManager from './components/notification.vue'
import NavigationBar from './components/navigation.vue'
import BreadcrumbNav from './components/breadcrumb.vue'
import FileIndex from './components/file_index.vue'
import ActionButtons from './components/action_buttons.vue'

export default defineComponent({
  name: 'App',
  components: {
    LoginForm,
    NotificationManager,
    NavigationBar,
    BreadcrumbNav,
    FileIndex,
    ActionButtons
  },
  setup() {
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

    return { state, actions }
  }
})
</script>
