import axios, { AxiosRequestConfig, AxiosResponse, InternalAxiosRequestConfig } from 'axios'
import type { APIResponse } from './types'

// 类型安全的 HTTP 客户端接口（响应拦截器已将 AxiosResponse 解包为 APIResponse）
export interface HttpClient {
    get<T = unknown>(url: string, config?: AxiosRequestConfig): Promise<APIResponse<T>>
    post<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<APIResponse<T>>
    put<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<APIResponse<T>>
    patch<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<APIResponse<T>>
    delete<T = unknown>(url: string, config?: AxiosRequestConfig): Promise<APIResponse<T>>
}

// Blob 下载专用客户端接口（responseType: 'blob' 时拦截器解包后直接是 Blob）
export interface HttpBlobClient {
    post(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<Blob>
}

// 获取 baseURL 配置
const baseURL = window.__BASE_URL__ || ''

// 创建配置了 baseURL 的 axios 实例
const axiosInstance = axios.create({ baseURL })
const axiosBlobInstance = axios.create({ baseURL })

// 导出类型安全的 HTTP 客户端（实际使用 axios 实例，通过类型断言修正拦截器解包后的返回类型）
export const http = axiosInstance as unknown as HttpClient
export const httpBlob = axiosBlobInstance as unknown as HttpBlobClient

export const interceptors = (state: { token: string | null; loading: boolean }, actions: { showNotification: (type: string, message: string) => void; clearAuth: () => void }) => {
    axiosInstance.interceptors.request.use(
        (config: InternalAxiosRequestConfig) => {
            if (state.token) {
                config.headers['Authorization'] = state.token
            }
            // 返回结果
            state.loading = true
            return config
        },
        (error: unknown) => {
            // 返回结果
            return Promise.reject(error)
        }
    )

    axiosInstance.interceptors.response.use(
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
        (error: unknown) => {
            const err = error as { response?: { status?: number; data?: { message?: string } }; request?: unknown }
            // 处理未授权错误
            if (err.response?.status === 401) {
                actions.showNotification('error', '登录已过期，请重新登录')
                actions.clearAuth()
            }
            // 处理其他 HTTP 错误
            else if (err.response) {
                const message = err.response.data?.message
                actions.showNotification('error', message || `请求失败: ${err.response.status}`)
            }
            // 处理网络错误
            else if (err.request) {
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
