import axios, { AxiosRequestConfig, AxiosResponse, InternalAxiosRequestConfig } from 'axios'

/**
 * 标准 API 响应结构
 */
export interface APIResponse<T = unknown> {
    success: boolean
    message?: string
    payload?: T
}

/**
 * 类型安全的 HTTP 客户端接口
 * 
 * 响应拦截器已将 AxiosResponse 解包为 APIResponse
 */
export interface HttpClient {
    get<T = unknown>(url: string, config?: AxiosRequestConfig): Promise<APIResponse<T>>
    post<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<APIResponse<T>>
    put<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<APIResponse<T>>
    patch<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<APIResponse<T>>
    delete<T = unknown>(url: string, config?: AxiosRequestConfig): Promise<APIResponse<T>>
}

/**
 * Blob 下载专用客户端接口
 * 
 * 当 responseType 为 'blob' 时，拦截器解包后直接返回 Blob 对象
 */
export interface HttpBlobClient {
    get(url: string, config?: AxiosRequestConfig): Promise<Blob>
    post(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<Blob>
}

/**
 * 系统通知类型
 */
export type NotificationType = 'success' | 'error' | 'warning' | 'info'

// API 使用相对 baseURL，业务接口只传模块路径
const axiosInstance = axios.create({ baseURL: 'api/' })
const axiosBlobInstance = axios.create({ baseURL: 'api/' })

/**
 * 将 api/ 相对路径转换为 WebSocket 绝对 URL
 * 兼容部署在 / 或 /xxx/ 子路径下的场景
 */
export const wsUrl = (path: string): string => {
    const base = new URL('api/' + path.replace(/^\/+/, ''), window.location.href)
    base.protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    return base.toString()
}

/**
 * 导出类型安全的 HTTP 客户端
 * 
 * 实际使用 axios 实例，通过类型断言修正拦截器解包后的返回类型
 */
export const http = axiosInstance as unknown as HttpClient

/**
 * 导出 Blob 下载专用客户端
 */
export const httpBlob = axiosBlobInstance as unknown as HttpBlobClient

/**
 * 注册 Axios 拦截器
 * 
 * @param state 全局状态，包含 token
 * @param actions 全局动作，包含显示通知和清除认证信息的方法
 */
export const interceptors = (
    state: { token: string | null },
    actions: { showNotification: (type: NotificationType, message: string) => void; clearAuth: () => void }
) => {
    /**
     * 公共请求拦截器：注入 Authorization header
     */
    const attachAuth = (config: InternalAxiosRequestConfig) => {
        if (state.token) {
            config.headers['Authorization'] = state.token
        }
        return config
    }

    /**
     * 公共错误处理
     * 
     * @param error 错误对象
     * @param isBlob 是否为 Blob 下载请求
     */
    const handleError = (error: unknown, isBlob = false) => {
        if (axios.isCancel(error)) return Promise.reject(error)
        const err = error as { response?: { status?: number; data?: { message?: string } }; request?: unknown }
        if (err.response?.status === 401) {
            actions.showNotification('error', '登录已过期，请重新登录')
            actions.clearAuth()
        } else if (err.response) {
            if (isBlob) {
                actions.showNotification('error', `下载失败: ${err.response.status}`)
            } else {
                const message = err.response.data?.message
                actions.showNotification('error', message || `请求失败: ${err.response.status}`)
            }
        } else if (err.request) {
            actions.showNotification('error', '网络连接失败，请检查网络')
        } else {
            actions.showNotification('error', '发生未知错误')
        }
        return Promise.reject(error)
    }

    axiosInstance.interceptors.request.use(
        (config: InternalAxiosRequestConfig) => attachAuth(config),
        (error: unknown) => Promise.reject(error)
    )

    axiosBlobInstance.interceptors.request.use(
        (config: InternalAxiosRequestConfig) => attachAuth(config),
        (error: unknown) => Promise.reject(error)
    )

    axiosBlobInstance.interceptors.response.use(
        (value: AxiosResponse) => value.data,
        (error: unknown) => handleError(error, true)
    )

    axiosInstance.interceptors.response.use(
        (value: AxiosResponse) => {
            // 过滤逻辑：不显示 GET 请求和 HTTP 200 状态码的消息
            const isGetRequest = value.config?.method?.toLowerCase() === 'get'
            const isSuccessStatus = value.status === 200
            if (!isGetRequest && !isSuccessStatus && value.data?.message) {
                actions.showNotification(value.data.success ? 'success' : 'error', value.data.message)
            }
            return value.data
        },
        (error: unknown) => handleError(error, false)
    )
}
