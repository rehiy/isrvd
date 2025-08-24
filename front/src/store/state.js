import { reactive } from 'vue'

import { interceptors } from '@/service/axios.js'

// Provide/Inject keys
export const APP_STATE_KEY = Symbol('app.state')
export const APP_ACTIONS_KEY = Symbol('app.actions')

export const initProvider = () => {
    const state = reactive({
        // 用户认证状态
        token: null,
        username: null,

        // 网络请求状态
        loading: false,

        // 文件管理状态
        currentPath: '/',
        files: [],

        // 通知状态
        notifications: []
    })

    const actions = {
        // 认证操作
        setAuth(data) {
            state.token = data.token
            state.username = data.username
            localStorage.setItem('app-token', data.token)
            localStorage.setItem('app-username', data.username)
        },

        clearAuth() {
            state.token = null
            state.username = null
            localStorage.removeItem('app-token')
            localStorage.removeItem('app-username')
        },

        // 文件操作
        async loadFiles(path = state.currentPath) {
            console.log('wait for loadFiles:', path)
        },

        // 通知操作
        showNotification(type, message) {
            if (!message) return;
            const id = Date.now() + Math.random();
            const timer = setTimeout(() => this.clearNotification(id), 5000);
            state.notifications.push({ id, type, message, timer });
        },

        clearNotification(id) {
            const idx = state.notifications.findIndex(n => n.id === id);
            if (idx !== -1) {
                const item = state.notifications[idx];
                state.notifications.splice(idx, 1);
                if (item && item.timer) {
                    clearTimeout(item.timer);
                }
            }
        },
    }

    interceptors(state, actions)

    return { state, actions }
}
