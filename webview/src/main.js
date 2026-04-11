import { createApp } from 'vue'
import App from '@/app.vue'
import router from '@/router'

// 导入全局样式
import '@fortawesome/fontawesome-free/css/all.min.css'
import '@xterm/xterm/css/xterm.css'

// 导入 Tailwind CSS 样式
import './assets/style.css'

// 创建并挂载应用
createApp(App).use(router).mount('#app')
