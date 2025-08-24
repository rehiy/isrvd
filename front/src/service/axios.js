import axios from 'axios'

export const interceptors = (state, actions) => {
    axios.interceptors.response.use(
        value => {
            if (state.token) {
                value.config.headers['Authorization'] = state.token
            }
            return value
        },
        error => {
            return Promise.reject(error)
        }
    )

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
