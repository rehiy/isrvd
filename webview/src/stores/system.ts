import { defineStore } from 'pinia'
import { ref, reactive } from 'vue'

import api from '@/service/api'
import type { LinkConfig } from '@/service/types'
import { initTheme } from '@/helper/theme'

interface ServiceAvailability {
    agent: boolean
    apisix: boolean
    caddy: boolean
    docker: boolean
    swarm: boolean
    compose: boolean
}

/**
 * 系统 Store
 * 
 * 管理系统级配置和服务状态
 * 主要职责：
 * 1. 服务可用性检测 (agent, apisix, docker, swarm, compose)
 * 2. 工具栏链接配置
 * 3. 主题设置
 * 4. 应用初始化状态
 */
export const useSystemStore = defineStore('system', () => {
    // ─── 状态定义 ───

    const initialized = ref(false)
    const initError = ref<string | null>(null)
    const serviceAvailability = reactive<ServiceAvailability>({
        agent: false,
        apisix: false,
        caddy: false,
        docker: false,
        swarm: false,
        compose: false
    })
    const toolbarLinks = ref<LinkConfig[]>([])

    // ─── 操作定义 ───

    async function initialize() {
        initialized.value = false
        initError.value = null

        initTheme() // 初始化主题

        try {
            await loadSystemData()
            initialized.value = true
        } catch (e) {
            console.error('Initialize failed:', e)
            initError.value = e instanceof Error ? e.message : '初始化失败'
            initialized.value = true
        }
    }

    async function loadSystemData() {
        const [probeRes, configRes] = await Promise.all([
            api.overviewProbe(),
            api.systemConfig(),
        ])

        const probe = probeRes?.payload
        if (probe) {
            Object.assign(serviceAvailability, {
                agent: probe.agent?.available || false,
                apisix: probe.apisix?.available || false,
                caddy: probe.caddy?.available || false,
                docker: probe.docker?.available || false,
                swarm: probe.swarm?.available || false,
                compose: probe.compose?.available || false
            })
        }

        toolbarLinks.value = configRes?.payload?.links || []
    }

    function hasPerm(module: string, founder: boolean, permissions: string[]): boolean {
        const checkAvailability = (seg: string): boolean => {
            const key = seg as keyof ServiceAvailability
            return !(key in serviceAvailability && !serviceAvailability[key])
        }

        // 精确路由匹配
        if (module.includes(' ')) {
            const path = module.split(' ')[1]
            const seg = path?.match(/^\/api\/([^/]+)/)?.[1]
            if (seg && !checkAvailability(seg)) return false
            return founder || permissions.includes(module)
        }

        // 模块匹配
        if (!checkAvailability(module)) return false
        if (founder) return true
        return permissions.some(key => {
            const path = key.split(' ')[1]
            return path && (path.startsWith(`/api/${module}/`) || path === `/api/${module}`)
        })
    }

    return {
        // 状态
        initialized,
        initError,
        serviceAvailability,
        toolbarLinks,
        // 操作
        initialize,
        loadSystemData,
        hasPerm
    }
})

// ─── 类型导出 ───
export type SystemStore = ReturnType<typeof useSystemStore>
