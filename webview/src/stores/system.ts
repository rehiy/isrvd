import { defineStore } from 'pinia'
import { ref, reactive } from 'vue'

import type { BootstrapData, LinkConfig } from '@/service/types'

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
 * 持有服务可用性、版本信息、系统配置等运行时状态。
 * 数据通过 apply(data) 由 portal 统一注入，不直接发起网络请求。
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
        compose: false,
    })
    const toolbarLinks = ref<LinkConfig[]>([])
    const maxUploadSize = ref<number>(104857600) // 默认 100MB
    const marketplaceUrl = ref<string>('')
    const openapiEnabled = ref<boolean>(false)

    // ─── 操作定义 ───

    // apply 将 bootstrap 响应中的 probe 和 config 写入 store
    function apply(data: BootstrapData) {
        const { probe, config } = data

        if (probe) {
            Object.assign(serviceAvailability, {
                agent: probe.agent || false,
                apisix: probe.apisix || false,
                caddy: probe.caddy || false,
                docker: probe.docker || false,
                swarm: probe.swarm || false,
                compose: probe.compose || false,
            })
        }

        if (config) {
            if (typeof config.maxUploadSize === 'number') {
                maxUploadSize.value = config.maxUploadSize
            }
            marketplaceUrl.value = config.marketplaceUrl || ''
            openapiEnabled.value = config.openapiEnabled || false
            toolbarLinks.value = config.links || []
        }
    }

    function hasPerm(module: string, founder: boolean, permissions: string[]): boolean {
        const checkAvailability = (seg: string): boolean => {
            const key = seg as keyof ServiceAvailability
            return !(key in serviceAvailability && !serviceAvailability[key])
        }

        if (module.includes(' ')) {
            const path = module.split(' ')[1]
            const seg = path?.match(/^\/api\/([^/]+)/)?.[1]
            if (seg && !checkAvailability(seg)) return false
            return founder || permissions.includes(module)
        }

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
        maxUploadSize,
        marketplaceUrl,
        openapiEnabled,
        // 操作
        apply,
        hasPerm,
    }
})

// ─── 类型导出 ───
export type SystemStore = ReturnType<typeof useSystemStore>
