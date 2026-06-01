import { defineStore } from 'pinia'
import { ref } from 'vue'

import api from '@/service/api'
import type { AuthInfo } from '@/service/types'

export const useAuthStore = defineStore('auth', () => {
    // ─── 状态定义 ───

    const authMode = ref<'jwt' | 'header' | null>(null)
    const token = ref<string | null>(null)
    const username = ref<string | null>(null)
    const permissionsLoaded = ref(false)
    const founder = ref(false)
    const permissions = ref<string[]>([])
    const oidcEnabled = ref(false)
    const oidcLoginLabel = ref('')

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

    async function loadAuth() {
        // 恢复 token
        const savedToken = localStorage.getItem('app-token')
        const savedUsername = localStorage.getItem('app-username')
        if (savedToken && savedUsername) {
            token.value = savedToken
            username.value = savedUsername
            authMode.value = 'jwt'
        }

        // 验证认证（网络错误时不清除已有 token，避免抖动导致误登出）
        let authRes
        try {
            authRes = await api.accountInfo()
        } catch {
            // 网络层异常：保持现有登录状态，不清空 token
            return
        }

        if (!authRes?.payload) {
            // 后端明确返回无 payload（如 401），说明 token 确实无效
            clearAuth()
            return
        }

        const payload = authRes.payload as AuthInfo

        // 核心原则：无 username 或无 member = 无权限，直接清理
        if (!payload?.username || !payload?.member) {
            clearAuth()
        } else {
            // 认证模式处理
            if (payload.mode === 'header') {
                setAuth({ authMode: 'header', token: '', username: payload.username })
            }
            // 权限赋值内聚在一处，确保有 member 才写入
            permissionsLoaded.value = true
            founder.value = payload.member.founder || false
            permissions.value = payload.member.permissions || []
        }

        // OIDC 配置是服务端配置，无论登录状态如何都需要更新（放在 clearAuth 之后）
        oidcEnabled.value = payload.oidcEnabled || false
        oidcLoginLabel.value = payload.oidcBtnLabel || ''
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
        oidcLoginLabel,
        // 操作
        setAuth,
        clearAuth,
        isAuthenticated,
        loadAuth
    }
})

// ─── 类型导出 ───
export type AuthStore = ReturnType<typeof useAuthStore>
