import { defineStore } from 'pinia'
import { ref } from 'vue'

import api from '@/service/api'
import type {
    AllConfig,
    NotifyConfig,
    OIDCConfig,
    PasskeyConfig,
    ServerConfig,
} from '@/service/types'

/** 配置分区，对应后端 AllConfig 的顶层字段 */
export type ConfigSection =
    | 'server'
    | 'password'
    | 'passkey'
    | 'oidc'
    | 'tha'
    | 'copilot'
    | 'notify'
    | 'apisix'
    | 'caddy'
    | 'docker'
    | 'monitor'
    | 'marketplace'
    | 'links'

/** 配置分组，一个分组包含若干配置分区，对应一个子路由 */
export type ConfigGroup = 'service' | 'auth' | 'ai' | 'gateway' | 'notify' | 'extensions'

export interface ConfigGroupMeta {
    id: ConfigGroup
    label: string
    description: string
    icon: string
    sections: ConfigSection[]
}

/** 配置分组导航定义：顺序即展示顺序 */
export const configGroups: ConfigGroupMeta[] = [
    { id: 'service', label: '基础服务', description: '端口、目录、上传、跨域与 JWT', icon: 'fa-server', sections: ['server'] },
    { id: 'auth', label: '登录认证', description: '密码、Passkey、OIDC 与代理 Header 登录', icon: 'fa-shield-halved', sections: ['password', 'passkey', 'oidc', 'tha'] },
    { id: 'ai', label: 'AI 助手', description: 'LLM 代理与模型改写', icon: 'fa-robot', sections: ['copilot'] },
    { id: 'gateway', label: '网关容器', description: 'APISIX、Caddy 与 Docker 连接参数', icon: 'fa-route', sections: ['apisix', 'caddy', 'docker'] },
    { id: 'notify', label: '告警通知', description: 'Webhook 通道与资源阈值', icon: 'fa-bell', sections: ['notify'] },
    { id: 'extensions', label: '系统扩展', description: '监控采集、应用市场与导航链接', icon: 'fa-puzzle-piece', sections: ['monitor', 'marketplace', 'links'] },
]

/** 默认配置，保证各分区字段完整，避免表单绑定出现 undefined */
function defaultConfig(): AllConfig {
    return {
        server: { listenAddr: '', rootDirectory: '', maxUploadSize: 104857600, allowedOrigins: [], jwtSecret: '', jwtExpiration: 86400, openapi: false, debug: false },
        password: { disabled: false, minLength: 6 },
        passkey: { enabled: false, rpName: '', rpId: '', rpOrigins: [], timeout: 60000 },
        oidc: { enabled: false, issuerUrl: '', clientId: '', clientSecret: '', redirectUrl: '', usernameClaim: 'sub', scopes: ['openid', 'profile', 'email'], loginLabel: '' },
        tha: { enabled: false, headerName: '', trustedCIDRs: [] },
        copilot: { model: '', baseUrl: '', apiKey: '' },
        notify: { webhooks: [], rules: [] },
        apisix: { adminUrl: '', adminKey: '' },
        caddy: { adminUrl: '' },
        docker: { host: '', containerRoot: '' },
        monitor: { interval: 0 },
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
 * 保存时只提交当前分组涉及的分区（后端对 nil 分区跳过更新）。
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
        const server: ServerConfig = { ...defaultConfig().server, ...payload.server }
        const passkey: PasskeyConfig = { ...defaultConfig().passkey, ...payload.passkey }
        const oidc: OIDCConfig = { ...defaultConfig().oidc, ...payload.oidc }
        const notify: NotifyConfig = {
            webhooks: (payload.notify?.webhooks || []).map(hook => ({ ...hook })),
            rules: (payload.notify?.rules || []).map(rule => ({ ...rule })),
        }

        draft.value = {
            server,
            password: { ...defaultConfig().password, ...payload.password },
            passkey,
            oidc,
            tha: { ...defaultConfig().tha, ...payload.tha },
            copilot: { ...defaultConfig().copilot, ...payload.copilot },
            notify,
            apisix: { ...defaultConfig().apisix, ...payload.apisix },
            caddy: { ...defaultConfig().caddy, ...payload.caddy },
            docker: { ...defaultConfig().docker, ...payload.docker },
            monitor: { ...defaultConfig().monitor, ...payload.monitor },
            marketplace: { ...defaultConfig().marketplace, ...payload.marketplace },
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
                return clone(draft.value[section as keyof AllConfig])
        }
    }

    /** 返回分组的提交载荷：仅包含该分组涉及的分区 */
    function groupPayload(group: ConfigGroup): Partial<AllConfig> {
        const meta = configGroups.find(item => item.id === group)
        if (!meta) return {}
        const payload: Record<string, unknown> = {}
        for (const section of meta.sections) {
            payload[section] = sectionPayload(section)
        }
        return payload as Partial<AllConfig>
    }

    /** 分组是否存在未保存改动 */
    function isDirty(group: ConfigGroup): boolean {
        const meta = configGroups.find(item => item.id === group)
        if (!meta) return false
        return meta.sections.some(
            section => JSON.stringify(sectionPayload(section)) !== JSON.stringify(baseline.value[section as keyof AllConfig]),
        )
    }

    /** 加载全量配置；force 为 true 时请求后端重载后返回 */
    async function load(force = false) {
        if (loading.value) return
        loading.value = true
        error.value = ''
        try {
            const res = await api.systemConfig(force ? { reload: 'true' } : undefined)
            applyPayload(res.payload as AllConfig)
            baseline.value = clone(draft.value)
            loaded.value = true
        } catch (e) {
            error.value = e instanceof Error ? e.message : '加载系统配置失败'
            throw e
        } finally {
            loading.value = false
        }
    }

    /** 保存当前分组：只提交该分组涉及的分区 */
    async function saveGroup(group: ConfigGroup) {
        if (saving.value) return
        saving.value = true
        error.value = ''
        try {
            await api.systemConfigUpdate(groupPayload(group))
            // 后端对密钥类字段返回空值，重新加载以刷新基线
            await load()
        } catch (e) {
            error.value = e instanceof Error ? e.message : '保存系统配置失败'
            throw e
        } finally {
            saving.value = false
        }
    }

    function reset() {
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
        saveGroup,
        isDirty,
        reset,
    }
})
