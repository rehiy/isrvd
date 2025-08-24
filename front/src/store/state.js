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

        // 文件管理状态
        currentPath: '/',
        files: [],

        // 网络请求状态
        loading: false,

        // 通知状态
        notification: {
            type: null, // 'success' | 'error' | null
            message: '',
            timer: null
        }
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
            if (!message) {
                return
            }

            if (state.notification.timer) {
                clearTimeout(state.notification.timer)
            }

            state.notification.type = type
            state.notification.message = message

            const duration = type === 'error' ? 5000 : 3000
            state.notification.timer = setTimeout(() => this.clearNotification(), duration)
        },

        clearNotification() {
            if (state.notification.timer) {
                clearTimeout(state.notification.timer)
                state.notification.timer = null
            }

            state.notification.type = null
            state.notification.message = ''
        },
    }

    interceptors(state, actions)

    return { state, actions }
}
