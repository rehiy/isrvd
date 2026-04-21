import { reactive } from 'vue'

import { interceptors } from '@/service/axios'
import type { FilerFileInfo } from '@/service/types'

// Provide/Inject keys
export const APP_STATE_KEY = 'app.state'
export const APP_ACTIONS_KEY = 'app.actions'

// ─── 类型定义 ───

interface Notification {
    id: number
    type: string
    message: string
    timer: ReturnType<typeof setTimeout>
}

interface ConfirmOptions {
    title?: string
    message?: string
    icon?: string
    iconColor?: string
    confirmText?: string
    danger?: boolean
    onConfirm?: (() => void | Promise<void>) | null
}

interface ConfirmState {
    show: boolean
    title: string
    message: string
    icon: string
    iconColor: string
    confirmText: string
    danger: boolean
    loading: boolean
    onConfirm: (() => void | Promise<void>) | null
}

interface ServiceAvailability {
    agent: boolean
    apisix: boolean
    docker: boolean
    swarm: boolean
    compose: boolean
}

export interface AppState {
    token: string | null
    username: string | null
    isPrimary: boolean
    allowTerminal: boolean
    permissions: Record<string, string>
    loading: boolean
    currentPath: string
    files: FilerFileInfo[]
    notifications: Notification[]
    confirm: ConfirmState
    serviceAvailability: ServiceAvailability
}

export interface AppActions {
    setAuth(data: { token: string; username: string }): void
    clearAuth(): void
    setPermissions(data: { isPrimary: boolean; allowTerminal: boolean; permissions: Record<string, string> }): void
    hasPerm(module: string, write?: boolean): boolean
    loadFiles(path?: string): Promise<void>
    showNotification(type: string, message: string): void
    clearNotification(id: number): void
    showConfirm(options: ConfirmOptions): void
    confirmLoading(loading: boolean): void
    closeConfirm(): void
    handleConfirm(): Promise<void>
    updateServiceAvailability(availability: {
        agent?: { available?: boolean };
        apisix?: { available?: boolean };
        docker?: { available?: boolean };
        swarm?: { available?: boolean };
        compose?: { available?: boolean }
    }): void
}

// 全局权限状态（响应式），供 router 守卫使用（避免循环依赖）
export const permState = reactive({
    loaded: false,
    isPrimary: false,
    allowTerminal: false,
    permissions: {} as Record<string, string>
})

export const initProvider = () => {
    const state = reactive<AppState>({
        // 用户认证状态
        token: null,
        username: null,
        isPrimary: false,
        allowTerminal: false,
        permissions: {},

        // 网络请求状态
        loading: false,

        // 文件管理状态
        currentPath: '/',
        files: [],

        // 通知状态
        notifications: [],

        // 确认模态框状态
        confirm: {
            show: false,
            title: '',
            message: '',
            icon: '',
            iconColor: 'blue',
            confirmText: '确认',
            danger: false,
            loading: false,
            onConfirm: null
        },

        // 服务可用性状态
        serviceAvailability: {
            agent: false,
            apisix: false,
            docker: false,
            swarm: false,
            compose: false
        }
    })

    const actions: AppActions = {
        // 认证操作
        setAuth(data: { token: string; username: string }) {
            state.token = data.token
            state.username = data.username
            localStorage.setItem('app-token', data.token)
            localStorage.setItem('app-username', data.username)
        },

        clearAuth() {
            state.token = null
            state.username = null
            state.isPrimary = false
            state.allowTerminal = false
            state.permissions = {}
            permState.loaded = false
            permState.isPrimary = false
            permState.allowTerminal = false
            permState.permissions = {}
            localStorage.removeItem('app-token')
            localStorage.removeItem('app-username')
        },

        setPermissions(data: { isPrimary: boolean; allowTerminal: boolean; permissions: Record<string, string> }) {
            state.isPrimary = data.isPrimary
            state.allowTerminal = data.allowTerminal
            state.permissions = data.permissions || {}
            permState.loaded = true
            permState.isPrimary = data.isPrimary
            permState.allowTerminal = data.allowTerminal
            permState.permissions = data.permissions || {}
        },

        hasPerm(module: string, write = false): boolean {
            if (state.isPrimary) return true
            const perm = state.permissions[module] || ''
            return write ? perm === 'rw' : (perm === 'r' || perm === 'rw')
        },

        // 文件操作
        async loadFiles(path: string = state.currentPath) {
            console.log('wait for loadFiles:', path)
        },

        // 通知操作
        showNotification(type: string, message: string) {
            if (!message) return
            const id = Date.now() + Math.random()
            const timer = setTimeout(() => this.clearNotification(id), 5000)
            state.notifications.push({ id, type, message, timer })
        },

        clearNotification(id: number) {
            const idx = state.notifications.findIndex(n => n.id === id)
            if (idx !== -1) {
                const item = state.notifications[idx]
                state.notifications.splice(idx, 1)
                if (item && item.timer) {
                    clearTimeout(item.timer)
                }
            }
        },

        // 确认模态框操作
        showConfirm(options: ConfirmOptions) {
            state.confirm = {
                show: true,
                title: options.title || '确认操作',
                message: options.message || '',
                icon: options.icon || 'fa-question-circle',
                iconColor: options.iconColor || 'blue',
                confirmText: options.confirmText || '确认',
                danger: options.danger || false,
                loading: false,
                onConfirm: options.onConfirm || null
            }
        },

        confirmLoading(loading: boolean) {
            state.confirm.loading = loading
        },

        closeConfirm() {
            state.confirm.show = false
            state.confirm.loading = false
            state.confirm.onConfirm = null
        },

        async handleConfirm() {
            if (state.confirm.onConfirm) {
                state.confirm.loading = true
                try {
                    await state.confirm.onConfirm()
                } finally {
                    state.confirm.loading = false
                }
            }
            this.closeConfirm()
        },

        // 服务可用性操作
        updateServiceAvailability(availability: { agent?: { available?: boolean }; docker?: { available?: boolean }; swarm?: { available?: boolean }; compose?: { available?: boolean }; apisix?: { available?: boolean } }) {
            state.serviceAvailability = {
                agent: availability.agent?.available || false,
                apisix: availability.apisix?.available || false,
                docker: availability.docker?.available || false,
                swarm: availability.swarm?.available || false,
                compose: availability.compose?.available || false
            }
        }
    }

    interceptors(state, actions)

    return { state, actions }
}
