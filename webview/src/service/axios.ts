import axios, { AxiosRequestConfig, AxiosResponse, InternalAxiosRequestConfig } from 'axios'

export const interceptors = (state: { token: string | null; loading: boolean }, actions: { showNotification: (type: string, message: string) => void; clearAuth: () => void }) => {
    axios.interceptors.request.use(
        (config: InternalAxiosRequestConfig) => {
            if (state.token) {
                config.headers['Authorization'] = state.token
            }
            // 返回结果
            state.loading = true
            return config
        },
        (error: any) => {
            // 返回结果
            return Promise.reject(error)
        }
    )

    axios.interceptors.response.use(
        (value: AxiosResponse) => {
            // 过滤逻辑：不显示 GET 请求和 HTTP 200 状态码的消息
            const isGetRequest = value.config?.method?.toLowerCase() === 'get'
            const isSuccessStatus = value.status === 200

            // 只有当不是 GET 请求且不是 200 状态码时才显示消息
            if (!isGetRequest && !isSuccessStatus && value.data?.message) {
                const message = value.data?.message
                actions.showNotification(value.data.success ? 'success' : 'error', message)
            }
            // 返回结果
            state.loading = false
            return value.data
        },
        (error: any) => {
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
            // 返回结果
            state.loading = false
            return Promise.reject(error)
        }
    )
}
