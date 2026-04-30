// ─── 认证相关 ───

export interface AuthInfoResponse {
    mode: 'jwt' | 'header'
    username?: string
    member?: import('./system').SystemMemberInfo
}

export interface AuthLoginRequest {
    username: string
    password: string
}

export interface AuthLoginResponse {
    token: string
    username: string
}
