import { defineStore } from 'pinia'
import { ref } from 'vue'

import { useLocaleStore } from './locale'

import { nodeLocation } from '@/helper/node'
import api from '@/service/api'
import type {
    AllConfig,
    NotifyConfig,
    OIDCConfig,
    PasskeyConfig,
    ServerConfig,
} from '@/service/types'

/** 配置分区，对应后端 AllConfig 的顶层字段 */
export type ConfigSection = keyof AllConfig

/** 配置分组，一个分组包含若干配置分区，对应一个子路由 */
export type ConfigGroup = 'service' | 'auth' | 'gateway' | 'alert' | 'metrics' | 'integrations'

export interface ConfigGroupMeta {
    id: ConfigGroup
    label: string
    description: string
    icon: string
    sections: ConfigSection[]
}

const viewingNode = Boolean(nodeLocation().id)

/** 配置分组导航定义：顺序即展示顺序 */
export const configGroups = ([
    { id: 'service', label: '基础服务', description: '端口、目录、上传、跨域与 JWT', icon: 'fa-server', sections: ['server'] },
    { id: 'auth', label: '登录认证', description: '密码、Passkey、OIDC 与代理 Header 登录', icon: 'fa-shield-halved', sections: ['password', 'passkey', 'oidc', 'tha'] },
    { id: 'gateway', label: '网关容器', description: 'APISIX、Caddy 与 Docker 连接参数', icon: 'fa-route', sections: ['apisix', 'caddy', 'docker'] },
    { id: 'metrics', label: '监控指标', description: '监控采集与指标出口', icon: 'fa-chart-line', sections: ['monitor', 'metrics'] },
    { id: 'alert', label: '告警通知', description: '通知通道、资源阈值与应用故障告警', icon: 'fa-bell', sections: ['notify'] },
    { id: 'integrations', label: '扩展集成', description: 'AI 助手、应用市场与导航链接', icon: 'fa-puzzle-piece', sections: ['copilot', 'marketplace', 'links'] },
] satisfies ConfigGroupMeta[])
    .filter(item => !viewingNode || item.id !== 'auth')
    .map(item => ({ ...item, sections: item.sections.filter(section => !viewingNode || section !== 'copilot') }))

/** 默认配置，保证各分区字段完整，避免表单绑定出现 undefined */
function defaultConfig(): AllConfig {
    return {
        server: { title: '', listenAddr: '', rootDirectory: '', maxUploadSize: 104857600, allowedOrigins: [], jwtSecret: '', jwtExpiration: 86400, openapi: false, debug: false },
        password: { disabled: false, minLength: 6 },
        passkey: { enabled: false, rpName: '', rpId: '', rpOrigins: [], timeout: 60000 },
        oidc: { enabled: false, issuerUrl: '', clientId: '', clientSecret: '', redirectUrl: '', usernameClaim: 'sub', scopes: ['openid', 'profile', 'email'], loginLabel: '' },
        tha: { enabled: false, headerName: '', trustedCIDRs: [] },
        copilot: { model: '', baseUrl: '', apiKey: '' },
        notify: {
            webhooks: [],
            rules: [],
            events: { containerEnabled: false, cronEnabled: false, certificateEnabled: false, nodeEnabled: false, restartThreshold: 3, restartWindow: 300, certificateDays: 14 },
        },
        apisix: { adminUrl: '', adminKey: '' },
        caddy: { adminUrl: '' },
        docker: { host: '', tls: { enabled: false, skipVerify: false, ca: '', cert: '', key: '' }, containerRoot: '', registries: [] },
        monitor: { interval: 0 },
        metrics: { hostEnabled: true, containersEnabled: true },
        marketplace: { url: '' },
        links: [],
    }
}

function clone<T>(value: T): T {
    return JSON.parse(JSON.stringify(value)) as T
}

/** 按空白切分多行/空格文本为数组 */
function splitList(text: string): string[] {
    return text.split(/\s+/).filter(Boolean)
}

/**
 * 系统配置草稿 Store
 *
 * 由父布局统一加载一次，各配置分组子页共享同一份草稿；
 * 保存时一次提交全部分区，因此分组切换不会丢失改动。
 */
export const useConfigStore = defineStore('config', () => {
    const draft = ref<AllConfig>(defaultConfig())
    const baseline = ref<AllConfig>(defaultConfig())

    const loading = ref(false)
    const saving = ref(false)
    const loaded = ref(false)
    const error = ref('')

    // 多行文本类配置：与数组字段互转
    const allowedOriginsText = ref('')
    const passkeyOriginsText = ref('')
    const oidcScopesText = ref('openid profile email')
    const thaTrustedCIDRsText = ref('')

    /** 应用接口返回的全量配置到草稿 */
    function applyPayload(payload: AllConfig) {
        const defaults = defaultConfig()
        // 后端 nil 切片会序列化为 null，需归一为数组，否则 isDirty 会误判
        const server: ServerConfig = { ...defaults.server, ...payload.server, allowedOrigins: payload.server?.allowedOrigins || [] }
        const passkey: PasskeyConfig = { ...defaults.passkey, ...payload.passkey, rpOrigins: payload.passkey?.rpOrigins || [] }
        const oidc: OIDCConfig = { ...defaults.oidc, ...payload.oidc, scopes: payload.oidc?.scopes || [] }
        const notify: NotifyConfig = {
            webhooks: (payload.notify?.webhooks || []).map(hook => ({ ...hook })),
            rules: (payload.notify?.rules || []).map(rule => ({ ...rule })),
            events: { ...defaults.notify.events, ...payload.notify?.events },
        }

        draft.value = {
            server,
            password: { ...defaults.password, ...payload.password },
            passkey,
            oidc,
            tha: { ...defaults.tha, ...payload.tha, trustedCIDRs: payload.tha?.trustedCIDRs || [] },
            copilot: { ...defaults.copilot, ...payload.copilot },
            notify,
            apisix: { ...defaults.apisix, ...payload.apisix },
            caddy: { ...defaults.caddy, ...payload.caddy },
            docker: { ...defaults.docker, ...payload.docker, tls: { ...defaults.docker.tls, ...payload.docker?.tls }, registries: (payload.docker?.registries || []).map(item => ({ ...item })) },
            monitor: { ...defaults.monitor, ...payload.monitor },
            metrics: { ...defaults.metrics, ...payload.metrics },
            marketplace: { ...defaults.marketplace, ...payload.marketplace },
            links: (payload.links || []).map(link => ({ ...link })),
        }

        allowedOriginsText.value = (server.allowedOrigins || []).join('\n')
        passkeyOriginsText.value = (passkey.rpOrigins || []).join('\n')
        oidcScopesText.value = (oidc.scopes || []).join(' ')
        thaTrustedCIDRsText.value = (draft.value.tha.trustedCIDRs || []).join('\n')
    }

    /** 返回单个分区的提交对象（应用多行文本转换）；links 分区为数组 */
    function sectionPayload(section: ConfigSection): unknown {
        switch (section) {
            case 'server':
                return { ...draft.value.server, allowedOrigins: splitList(allowedOriginsText.value) }
            case 'passkey':
                return { ...draft.value.passkey, rpOrigins: splitList(passkeyOriginsText.value) }
            case 'oidc':
                return { ...draft.value.oidc, scopes: splitList(oidcScopesText.value) }
            case 'tha':
                return { ...draft.value.tha, trustedCIDRs: splitList(thaTrustedCIDRsText.value) }
            default:
                return clone(draft.value[section])
        }
    }

    function changedPayload(): Partial<AllConfig> {
        const payload: Record<string, unknown> = {}
        for (const section of configGroups.flatMap(item => item.sections)) {
            const value = sectionPayload(section)
            if (JSON.stringify(value) !== JSON.stringify(baseline.value[section])) {
                payload[section] = value
            }
        }
        return payload as Partial<AllConfig>
    }

    /** 是否存在任一分组的未保存改动 */
    function hasUnsaved(): boolean {
        return loaded.value && Object.keys(changedPayload()).length > 0
    }

    let generation = 0

    /** 读取已保存配置并刷新草稿基线。 */
    async function load(afterSave = false) {
        if (loading.value) {
            return
        }
        const requestGeneration = generation
        loading.value = true
        error.value = ''
        try {
            let res = await api.systemConfig(afterSave)
            if (afterSave && res.success === false && requestGeneration === generation) {
                // 刚保存时服务可能正在重载，给这次回读一次重试机会。
                await new Promise(resolve => setTimeout(resolve, 300))
                if (requestGeneration === generation) {
                    res = await api.systemConfig()
                }
            }
            if (requestGeneration !== generation) {
                return
            }
            const payload = res.payload
            if (!payload || (afterSave && !('config' in payload))) {
                throw new Error(useLocaleStore().t('加载系统配置失败'))
            }
            applyPayload('config' in payload ? payload.config : payload)
            baseline.value = clone(draft.value)
            loaded.value = true
        } catch (e) {
            if (requestGeneration !== generation) {
                return
            }
            error.value = e instanceof Error ? e.message : useLocaleStore().t('加载系统配置失败')
            throw e
        } finally {
            if (requestGeneration === generation) {
                loading.value = false
            }
        }
    }

    /** 一次提交有改动的配置分区，返回持久化结果。 */
    async function saveAll() {
        if (!loaded.value || loading.value || saving.value) {
            return false
        }
        const payload = changedPayload()
        if (Object.keys(payload).length === 0) {
            return false
        }
        const requestGeneration = generation
        saving.value = true
        error.value = ''
        let persisted = false
        try {
            await api.systemConfigUpdate(payload)
            if (requestGeneration !== generation) {
                return false
            }
            persisted = true
            // 已保存快照包含归一化结果，并已脱敏。
            await load(true)
            return requestGeneration === generation
        } catch (e) {
            if (requestGeneration !== generation) {
                return false
            }
            if (persisted) {
                // 保留草稿，等重新读取成功后再允许下一次保存。
                loaded.value = false
                return true
            }
            error.value = e instanceof Error ? e.message : useLocaleStore().t('保存系统配置失败')
            throw e
        } finally {
            if (requestGeneration === generation) {
                saving.value = false
            }
        }
    }

    function reset() {
        generation++
        loading.value = false
        saving.value = false
        draft.value = defaultConfig()
        baseline.value = defaultConfig()
        allowedOriginsText.value = ''
        passkeyOriginsText.value = ''
        oidcScopesText.value = 'openid profile email'
        thaTrustedCIDRsText.value = ''
        loaded.value = false
        error.value = ''
    }

    return {
        draft,
        loading,
        saving,
        loaded,
        error,
        allowedOriginsText,
        passkeyOriginsText,
        oidcScopesText,
        thaTrustedCIDRsText,
        load,
        saveAll,
        hasUnsaved,
        reset,
    }
})
