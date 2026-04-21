// ─── 认证相关 ───

export interface AuthLoginRequest {
    username: string
    password: string
}

export interface AuthLoginResponse {
    token: string
    username: string
}
