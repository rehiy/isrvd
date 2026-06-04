// ─── 认证相关 ───

export interface AuthInfo {
    mode: 'jwt' | 'header'
    username?: string
    member?: MemberInfo
    oidcEnabled: boolean
    // OIDC 登录按钮自定义名称
    oidcBtnLabel?: string
    passkeyEnabled: boolean
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

// ─── Passkey 认证 ───

// 开始登录请求数据
export interface PasskeyBeginLoginData {
    username?: string  // 可选，为空则使用可发现凭证
}

// 开始注册/登录的统一数据
export interface PasskeyBeginData {
    sessionId: string
    options: CredentialCreationOptions | CredentialRequestOptions
}

// Passkey 登录结果
export interface PasskeyLoginResult {
    token: string
    username: string
}

// Passkey 凭证信息（与后端 config.PasskeyCredential 对应）
export interface PasskeyCredential {
    idBase64: string        // 凭证 ID (Base64 编码)
    aaguidBase64: string    // 认证器 AAGUID
    signCount: number       // 初始签名计数
    displayName: string     // 显示名称
    addedAt: string         // 添加时间 (ISO 8601)
}
