import { createApp } from 'vue'
import { createPinia } from 'pinia'

import App from '@/app.vue'
import router, { setRouterGuard } from '@/router'
import { isNotifiedError } from '@/service/client'
import { useLocaleStore, usePortal } from '@/stores'

// 导入全局样式
import '@fortawesome/fontawesome-free/css/all.min.css'

// 导入 Tailwind CSS 样式
import './assets/style.css'

// CopilotKit 侧栏自带样式（自包含，不依赖其他 CSS）
import '@copilotkit/vue/styles.css'

// 创建并挂载应用
const app = createApp(App)
const pinia = createPinia()

app.use(pinia)
app.use(router)

// 请求失败已由拦截器统一提示，这类错误不再向上报告；其他错误（代码缺陷等）保持原有输出
window.addEventListener('unhandledrejection', event => {
    if (isNotifiedError(event.reason)) {
        event.preventDefault()
    }
})
// 设置 errorHandler 后 Vue 不再输出默认的 dev 警告（含组件调用栈）与抛错，这里手动补回组件来源
app.config.errorHandler = (error, instance, info) => {
    if (isNotifiedError(error)) {
        return
    }
    if (import.meta.env.DEV) {
        const source = instance?.$options?.__file || instance?.$options?.name || 'unknown'
        console.error(`Unhandled error during execution of ${info}\n  at <${source}>`, error)
        return
    }
    console.error(`[${info}]`, error)
}

// 设置路由守卫（在 Pinia 初始化之后）
const portal = usePortal()
setRouterGuard(portal.hasPerm, () => portal.permissionsLoaded, portal.isAuthenticated)

// 模板内统一用 $t('中文原文') 取文案，切换语言时自动重渲染
app.config.globalProperties.$t = useLocaleStore().t

app.mount('#app')
