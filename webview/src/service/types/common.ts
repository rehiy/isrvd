// ─── 通用 API 响应 ───

export interface APIResponse<T = unknown> {
    success: boolean
    message?: string
    payload?: T
}
