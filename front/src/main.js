import { createApp } from 'vue'
import App from './app.vue'

// 导入全局样式
import 'bootstrap/dist/css/bootstrap.min.css'
import '@fortawesome/fontawesome-free/css/all.min.css'
import '@xterm/xterm/css/xterm.css'
import './assets/style.css'

// 导入并初始化 Bootstrap
import * as bootstrap from 'bootstrap'

// 确保 bootstrap 对象可在全局访问
window.bootstrap = bootstrap

// 创建并挂载应用
createApp(App).mount('#app')
