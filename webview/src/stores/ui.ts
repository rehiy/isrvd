import { defineStore } from 'pinia'
import { ref, reactive } from 'vue'

// ─── 类型定义 ───

interface Notification {
    id: number
    type: string
    message: string
    timer: ReturnType<typeof setTimeout>
}

export interface ConfirmOptions {
    title?: string
    message?: string
    icon?: string
    iconColor?: string
    confirmText?: string
    danger?: boolean
    onConfirm?: (() => void | Promise<void>) | null
}

export const useUIStore = defineStore('ui', () => {
    // ─── 状态定义 ───

    const notifications = ref<Notification[]>([])
    const confirm = reactive({
        show: false,
        title: '',
        message: '',
        icon: '',
        iconColor: 'blue',
        confirmText: '确认',
        danger: false,
        loading: false,
        onConfirm: null as (() => void | Promise<void>) | null
    })

    // ─── Notification 操作 ───

    function showNotification(type: string, message: string) {
        if (!message) return
        const id = Date.now() + Math.random()
        const timer = setTimeout(() => clearNotification(id), 5000)
        notifications.value.push({ id, type, message, timer })
    }

    function clearNotification(id: number) {
        const idx = notifications.value.findIndex(n => n.id === id)
        if (idx !== -1) {
            clearTimeout(notifications.value[idx].timer)
            notifications.value.splice(idx, 1)
        }
    }

    // ─── Confirm 操作 ───

    function showConfirm(options: ConfirmOptions) {
        Object.assign(confirm, {
            show: true,
            title: options.title || '确认操作',
            message: options.message || '',
            icon: options.icon || 'fa-question-circle',
            iconColor: options.iconColor || 'blue',
            confirmText: options.confirmText || '确认',
            danger: options.danger || false,
            loading: false,
            onConfirm: options.onConfirm || null
        })
    }

    function confirmLoading(loading: boolean) {
        confirm.loading = loading
    }

    function closeConfirm() {
        confirm.show = false
        confirm.loading = false
        confirm.onConfirm = null
    }

    async function handleConfirm() {
        if (!confirm.onConfirm) return closeConfirm()
        confirm.loading = true
        try {
            await confirm.onConfirm()
        } finally {
            confirm.loading = false
        }
        closeConfirm()
    }

    return {
        // 状态
        notifications,
        confirm,
        // 操作
        showNotification,
        clearNotification,
        showConfirm,
        confirmLoading,
        closeConfirm,
        handleConfirm
    }
})
