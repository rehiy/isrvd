import { defineStore } from 'pinia'
import { ref } from 'vue'

import type { BootstrapData } from '@/service/types'

export const useAuthStore = defineStore('auth', () => {
    // ─── 状态定义 ───

    const authMode = ref<'jwt' | 'header' | null>(null)
    const token = ref<string | null>(null)
    const username = ref<string | null>(null)
    const permissionsLoaded = ref(false)
    const founder = ref(false)
    const permissions = ref<string[]>([])
    const oidcEnabled = ref(false)
    const oidcOnly = ref(false)
    const oidcLoginLabel = ref('')
    const passkeyEnabled = ref(false)

    // ─── 操作定义 ───

    function setAuth(data: { authMode: 'jwt' | 'header'; token: string; username: string }) {
        authMode.value = data.authMode
        username.value = data.username
        token.value = data.token || ''
        if (data.authMode === 'jwt') {
            localStorage.setItem('app-token', data.token)
            localStorage.setItem('app-username', data.username)
        }
    }

    function clearAuth() {
        authMode.value = null
        token.value = null
        username.value = null
        permissionsLoaded.value = false
        founder.value = false
        permissions.value = []
        localStorage.removeItem('app-token')
        localStorage.removeItem('app-username')
    }

    function isAuthenticated() {
        return !!username.value
    }

    // apply 将 bootstrap 响应中的 auth 字段写入 store
    function apply(data: BootstrapData) {
        const auth = data.auth

        if (!auth?.username || !auth?.member) {
            clearAuth()
        } else {
            if (auth.mode === 'header') {
                setAuth({ authMode: 'header', token: '', username: auth.username })
            }
            permissionsLoaded.value = true
            founder.value = auth.member.founder || false
            permissions.value = auth.member.permissions || []
        }

        // OIDC / Passkey 配置无论登录状态都更新（放在 clearAuth 之后）
        oidcEnabled.value = auth?.oidcEnabled || false
        oidcOnly.value = auth?.oidcOnly || false
        oidcLoginLabel.value = auth?.oidcBtnLabel || ''
        passkeyEnabled.value = auth?.passkeyEnabled || false
    }

    // restoreToken 从 localStorage 恢复 token（冷启动时使用）
    function restoreToken() {
        const savedToken = localStorage.getItem('app-token')
        const savedUsername = localStorage.getItem('app-username')
        if (savedToken && savedUsername) {
            token.value = savedToken
            username.value = savedUsername
            authMode.value = 'jwt'
        }
    }

    return {
        // 状态
        authMode,
        token,
        username,
        permissionsLoaded,
        founder,
        permissions,
        oidcEnabled,
        oidcOnly,
        oidcLoginLabel,
        passkeyEnabled,
        // 操作
        setAuth,
        clearAuth,
        isAuthenticated,
        restoreToken,
        apply,
    }
})

// ─── 类型导出 ───
export type AuthStore = ReturnType<typeof useAuthStore>
