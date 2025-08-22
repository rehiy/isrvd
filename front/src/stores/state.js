// ==================== 状态管理 ====================

import { reactive } from 'vue'
import axios from 'axios'

// Provide/Inject keys
export const APP_STATE_KEY = Symbol('appState')
export const APP_ACTIONS_KEY = Symbol('appActions')

export const initProvider = () => {
    const state = createAppState()
    const actions = createAppActions(state)

    // 注册 axios 全局错误处理器
    axios.interceptors.response.use(
        response => {
            const message = response.data?.message || ''
            if (message) actions.showSuccess(message)
            return response.data
        },
        error => {
            // 处理未授权错误
            if (error.response?.status === 401) {
                actions.clearAuth()
                actions.showError('登录已过期，请重新登录')
            }
            // 处理其他 HTTP 错误
            else if (error.response) {
                const message = error.response.data?.message || `请求失败: ${error.response.status}`
                actions.showError(message)
            }
            // 处理网络错误
            else if (error.request) {
                actions.showError('网络连接失败，请检查网络')
            }
            // 处理其他错误
            else {
                actions.showError('发生未知错误')
            }
            return Promise.reject(error)
        }
    )

    return { state, actions }
}

// 全局状态管理
function createAppState() {
    return reactive({
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
}

// 全局操作方法
function createAppActions(state) {
    return {
        // 认证操作
        setAuth(userData) {
            state.user = userData.user
            state.token = userData.token
            localStorage.setItem('file-manager-token', userData.token)
            localStorage.setItem('file-manager-user', userData.user)
            if (state.token) {
                axios.defaults.headers.common['Authorization'] = state.token
            }
        },

        clearAuth() {
            state.user = null
            state.token = null
            localStorage.removeItem('file-manager-token')
            localStorage.removeItem('file-manager-user')
            delete axios.defaults.headers.common['Authorization']
        },

        // 文件操作
        async loadFiles(path = state.currentPath) {
            console.log('wait for loadFiles', path)
        },

        // 通知操作
        showNotification(type, message) {
            if (state.notification.timer) {
                clearTimeout(state.notification.timer)
            }

            state.notification.type = type
            state.notification.message = message

            const duration = type === 'error' ? 5000 : 3000
            state.notification.timer = setTimeout(() => {
                this.clearNotification()
            }, duration)
        },

        clearNotification() {
            if (state.notification.timer) {
                clearTimeout(state.notification.timer)
                state.notification.timer = null
            }
            state.notification.type = null
            state.notification.message = ''
        },

        showSuccess(message) {
            this.showNotification('success', message)
        },

        showError(message) {
            this.showNotification('error', message)
        },
    }
}
