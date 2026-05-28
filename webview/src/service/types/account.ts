// ─── 认证相关 ───

export interface AuthInfo {
    mode: 'jwt' | 'header'
    username?: string
    member?: MemberInfo
    oidcEnabled: boolean
    // OIDC 登录按钮自定义名称
    oidcBtnLabel?: string
}

export interface AuthLogin {
    username: string
    password: string
}

export interface AuthLoginResult {
    token: string
    username: string
}

export interface OIDCExchange {
    code: string
}

// ─── 路由权限 ───

// RouteAccess 路由访问级别（与后端枚举对应）
export const RouteAccessAnon = -1 // 匿名，无需认证
export const RouteAccessPerm = 0 // 需要权限控制（默认）
export const RouteAccessAuth = 1 // 登录即可访问

export interface Route {
    key: string
    module: string
    label: string
    access: number
}

// ─── 成员管理 ───

export interface MemberInfo {
    username: string
    homeDirectory: string
    founder: boolean
    description: string
    permissions: string[]
}

export interface MemberUpsert {
    username: string
    // 写入时为空表示保留原值（仅更新场景）
    password: string
    homeDirectory: string
    description: string
    permissions: string[]
}

// ─── API Token ───

export interface ApiTokenCreate {
    name: string
    expiresIn: number // 过期时间（秒），0 表示永不过期
}

export interface ApiTokenResult {
    token: string
    name: string
}

// ─── 修改密码 ───

export interface ChangePassword {
    oldPassword: string
    newPassword: string
}
