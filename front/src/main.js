import { createApp } from 'vue'
import App from '@/app.vue'

// 导入全局样式
import * as bootstrap from 'bootstrap'
import 'bootstrap/dist/css/bootstrap.min.css'
import '@fortawesome/fontawesome-free/css/all.min.css'
import '@xterm/xterm/css/xterm.css'

// 导入自定义样式
import './assets/style.css'

// 暴露 bootstrap
window.bootstrap = bootstrap

// 创建并挂载应用
createApp(App).mount('#app')
