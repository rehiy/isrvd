import { defineStore, storeToRefs } from 'pinia'

import { interceptors } from '@/service/axios'
import api from '@/service/api'
import type { BootstrapData } from '@/service/types'
import { initTheme } from '@/helper/theme'

import { useAuthStore } from './auth'
import { useSystemStore } from './system'
import { useUIStore } from './ui'

/**
 * Portal Store - 统一入口
 *
 * 组合 auth、system、ui 三个子 store，提供统一的访问接口。
 * 主要职责：
 * 1. 调用 /overview/bootstrap，将数据分发到 auth/system store
 * 2. 注册 axios 拦截器
 * 3. 提供统一的访问入口，隔离内部实现
 */
export const usePortalStore = defineStore('portal', () => {
    // ─── 引用子 Store ───

    const authStore = useAuthStore()
    const systemStore = useSystemStore()
    const uiStore = useUIStore()

    // ─── 分发启动数据到子 store ───

    function applyBootstrap(data: BootstrapData) {
        authStore.apply(data)
        if (data.probe || data.config) {
            systemStore.apply(data)
        }
        systemStore.initialized = true
    }

    // ─── 初始化（冷启动）───
    // 恢复本地 token → 发起 bootstrap 请求 → 分发到 auth/system store

    async function initialize() {
        systemStore.initialized = false
        systemStore.initError = null

        initTheme()
        authStore.restoreToken()

        try {
            const res = await api.overviewBootstrap()
            if (!res?.payload) {
                authStore.clearAuth()
                systemStore.initialized = true
                return
            }
            applyBootstrap(res.payload as BootstrapData)
        } catch (e) {
            // 网络异常：保持现有登录状态，标记初始化完成
            console.error('Portal initialize failed:', e)
            systemStore.initError = e instanceof Error ? e.message : '初始化失败'
            systemStore.initialized = true
        }
    }

    // ─── 登录后刷新（token 已由调用方写入）───
    // 重新获取 bootstrap（此时带新 token）→ 重新分发到 auth/system store

    async function refresh() {
        systemStore.initialized = false
        systemStore.initError = null

        try {
            const res = await api.overviewBootstrap()
            if (!res?.payload) {
                systemStore.initialized = true
                return
            }
            applyBootstrap(res.payload as BootstrapData)
        } catch (e) {
            console.error('Portal refresh failed:', e)
            systemStore.initError = e instanceof Error ? e.message : '刷新失败'
            systemStore.initialized = true
        }
    }

    // ─── 权限检查（组合 auth 和 system）───

    function hasPerm(module: string): boolean {
        return systemStore.hasPerm(module, authStore.founder, authStore.permissions)
    }

    // ─── 注册 Axios 拦截器 ───

    const stateProxy = {
        get token() { return authStore.token },
        set token(val: string | null) { authStore.token = val },
    }

    interceptors(stateProxy, {
        showNotification: uiStore.showNotification,
        clearAuth: authStore.clearAuth,
    })

    // ─── 导出统一接口 ───

    const authRefs = storeToRefs(authStore)
    const systemRefs = storeToRefs(systemStore)
    const uiRefs = storeToRefs(uiStore)

    return {
        // Portal 方法
        initialize,
        refresh,

        // Auth Store 状态（响应式）
        authMode: authRefs.authMode,
        token: authRefs.token,
        username: authRefs.username,
        permissionsLoaded: authRefs.permissionsLoaded,
        founder: authRefs.founder,
        permissions: authRefs.permissions,
        oidcEnabled: authRefs.oidcEnabled,
        oidcOnly: authRefs.oidcOnly,
        oidcLoginLabel: authRefs.oidcLoginLabel,
        passkeyEnabled: authRefs.passkeyEnabled,
        // Auth Store 方法
        setAuth: authStore.setAuth,
        clearAuth: authStore.clearAuth,
        isAuthenticated: authStore.isAuthenticated,

        // System Store 状态（响应式）
        initialized: systemRefs.initialized,
        initError: systemRefs.initError,
        serviceAvailability: systemRefs.serviceAvailability,
        toolbarLinks: systemRefs.toolbarLinks,
        maxUploadSize: systemRefs.maxUploadSize,
        marketplaceUrl: systemRefs.marketplaceUrl,
        openapiEnabled: systemRefs.openapiEnabled,
        // System Store 方法
        hasPerm,

        // UI Store 状态（响应式）
        notifications: uiRefs.notifications,
        confirm: uiRefs.confirm,
        // UI Store 方法
        showNotification: uiStore.showNotification,
        clearNotification: uiStore.clearNotification,
        showConfirm: uiStore.showConfirm,
        confirmLoading: uiStore.confirmLoading,
        closeConfirm: uiStore.closeConfirm,
        handleConfirm: uiStore.handleConfirm,
    }
})

// ─── 类型导出 ───
export type PortalStore = ReturnType<typeof usePortalStore>
