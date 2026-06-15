import { createApp } from 'vue'
import { createPinia } from 'pinia'

import App from '@/app.vue'
import router, { setRouterGuard } from '@/router'
import { usePortal } from '@/stores'

// 导入全局样式
import '@fortawesome/fontawesome-free/css/all.min.css'

// 导入 Tailwind CSS 样式
import './assets/style.css'

// 创建并挂载应用
const app = createApp(App)
const pinia = createPinia()

app.use(pinia)
app.use(router)

// 设置路由守卫（在 Pinia 初始化之后）
const portal = usePortal()
setRouterGuard(portal.hasPerm, () => portal.permissionsLoaded, portal.isAuthenticated)

app.mount('#app')
