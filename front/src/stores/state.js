// ==================== 状态管理 ====================

import { reactive } from 'vue'
import axios from 'axios'

// Provide/Inject keys
export const APP_STATE_KEY = Symbol('app.state')
export const APP_ACTIONS_KEY = Symbol('app.actions')

export const initProvider = () => {
    const state = reactive({
        // 用户认证状态
        user: null,
        token: null,

        // 文件管理状态
        loading: false,
        currentPath: '/',
        files: [],

        // 通知状态
        notification: {
            type: null, // 'success' | 'error' | null
            message: '',
            timer: null
        }
    })

    const actions = {
        // 认证操作
        setAuth(userData) {
            state.user = userData.user
            state.token = userData.token
            localStorage.setItem('app-token', userData.token)
            localStorage.setItem('app-username', userData.user)
            if (state.token) {
                axios.defaults.headers.common['Authorization'] = state.token
            }
        },

        clearAuth() {
            state.user = null
            state.token = null
            localStorage.removeItem('app-token')
            localStorage.removeItem('app-username')
            delete axios.defaults.headers.common['Authorization']
        },

        // 文件操作
        async loadFiles(path = state.currentPath) {
            console.log('wait for loadFiles', path)
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

    axiosRegister(state, actions)

    return { state, actions }
}

export const axiosRegister = (state, actions) => {
    axios.interceptors.response.use(
        value => {
            if (value.data?.message) {
                const message = value.data?.message
                actions.showNotification(value.data.success ? 'success' : 'error', message)
            }
            return value.data
        },
        error => {
            // 处理未授权错误
            if (error.response?.status === 401) {
                actions.showNotification('error', '登录已过期，请重新登录')
                actions.clearAuth()
            }
            // 处理其他 HTTP 错误
            else if (error.response) {
                const message = error.response.data?.message
                actions.showNotification('error', message || `请求失败: ${error.response.status}`)
            }
            // 处理网络错误
            else if (error.request) {
                actions.showNotification('error', '网络连接失败，请检查网络')
            }
            // 处理其他错误
            else {
                actions.showNotification('error', '发生未知错误')
            }
            return Promise.reject(error)
        }
    )
}
