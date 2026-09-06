import axios, { AxiosRequestConfig, AxiosResponse, InternalAxiosRequestConfig } from 'axios'

export interface APIResponse<T = unknown> {
    success: boolean
    message?: string
    payload?: T
}

// 响应拦截器已将 AxiosResponse 解包为 APIResponse。
export interface HttpClient {
    get<T = unknown>(url: string, config?: AxiosRequestConfig): Promise<APIResponse<T>>
    post<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<APIResponse<T>>
    put<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<APIResponse<T>>
    patch<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<APIResponse<T>>
    delete<T = unknown>(url: string, config?: AxiosRequestConfig): Promise<APIResponse<T>>
}

// Blob 请求由独立拦截器直接解包为 Blob。
export interface HttpBlobClient {
    get(url: string, config?: AxiosRequestConfig): Promise<Blob>
    post(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<Blob>
}

export type NotificationType = 'success' | 'error' | 'warning' | 'info'

// API 使用相对 baseURL，业务接口只传模块路径
const axiosInstance = axios.create({ baseURL: 'api/' })
const axiosBlobInstance = axios.create({ baseURL: 'api/' })

/**
 * 将 api/ 相对路径转换为绝对 HTTP/HTTPS URL
 * 兼容部署在 / 或 /xxx/ 子路径下的场景
 * 用于下载链接、OIDC 跳转、SSE（EventSource）等需要绝对 URL 的场景
 */
export const absUrl = (path: string): string =>
    new URL('api/' + path.replace(/^\/+/, ''), window.location.href).toString()

/**
 * 将 api/ 相对路径转换为 WebSocket 绝对 URL
 * 兼容部署在 / 或 /xxx/ 子路径下的场景
 */
export const wsUrl = (path: string): string =>
    absUrl(path).replace(/^https?:/, m => m === 'https:' ? 'wss:' : 'ws:')

// 类型断言反映响应拦截器解包后的实际返回值。
export const http = axiosInstance as unknown as HttpClient

export const httpBlob = axiosBlobInstance as unknown as HttpBlobClient

export const interceptors = (
    state: { token: string | null },
    actions: { showNotification: (type: NotificationType, message: string) => void; clearAuth: () => void }
) => {
    const attachAuth = (config: InternalAxiosRequestConfig) => {
        if (state.token) {
            config.headers['Authorization'] = state.token
        }
        return config
    }

    const handleError = async (error: unknown, isBlob = false) => {
        if (axios.isCancel(error)) return Promise.reject(error)
        if (!axios.isAxiosError<APIResponse | Blob>(error)) {
            actions.showNotification('error', '发生未知错误')
            return Promise.reject(error)
        }
        let message = ''
        const data = error.response?.data
        if (data instanceof Blob) {
            try {
                message = (JSON.parse(await data.text()) as APIResponse).message || ''
            } catch { /* 非 JSON 下载错误使用状态码兜底 */ }
        } else {
            message = data?.message || ''
        }

        // 登录接口本身返回的 401（用户名/密码错误等）是登录失败，不是会话过期，走下方通用错误提示
        const isLoginRequest = error.config?.url?.includes('account/login')
        if (error.response?.status === 401 && !isLoginRequest) {
            actions.showNotification('error', message || '登录已过期，请重新登录')
            actions.clearAuth()
        } else if (error.response) {
            const fallback = isBlob ? `下载失败: ${error.response.status}` : `请求失败: ${error.response.status}`
            actions.showNotification('error', message || fallback)
        } else if (error.request) {
            actions.showNotification('error', '网络连接失败，请检查网络')
        } else {
            actions.showNotification('error', '发生未知错误')
        }
        return Promise.reject(error)
    }

    for (const instance of [axiosInstance, axiosBlobInstance]) {
        instance.interceptors.request.use(attachAuth, (error: unknown) => Promise.reject(error))
    }

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
        (error: unknown) => handleError(error)
    )
}
