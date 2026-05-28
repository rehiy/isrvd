import type { AxiosRequestConfig } from 'axios'
import { http, httpBlob } from './axios'
import type {
    // Overview
    SystemProbe,
    SystemStat,
    MonitorHostRecord,
    MonitorContainerRecord,
    // System
    AllConfig,
    AuditLog,
    // Account
    AuthLogin,
    AuthLoginResult,
    OIDCExchange,
    AuthInfo,
    MemberInfo,
    MemberUpsert,
    Route,
    ApiTokenCreate,
    ApiTokenResult,
    ChangePassword,
    // Filer
    FilerList,
    FilerRead,
    // APISIX
    ApisixRoute,
    ApisixConsumer,
    ApisixConsumerCreate,
    ApisixConsumerUpdate,
    ApisixPluginConfigPayload,
    ApisixSSLCreate,
    ApisixSSLUpdate,
    ApisixSSL,
    ApisixUpstreamCreate,
    ApisixUpstreamUpdate,
    ApisixUpstream,
    ApisixPluginConfig,
    ApisixRevokeWhitelist,
    // Caddy
    CaddyInfo,
    CaddyGlobal,
    CaddyRoute,
    CaddyRouteUpsert,
    CaddyCert,
    // Docker
    DockerInfo,
    DockerContainerInfo,
    DockerContainerStats,
    DockerContainerCreate,
    DockerContainerCreateResult,
    DockerContainerDetail,
    DockerContainerCompose,
    DockerImageInfo,
	DockerImageDetail,
    DockerImageSearchResult,
    DockerImagePrune,
    DockerImagePruneResult,
    DockerNetworkInfo,
	DockerNetworkDetail,
    DockerNetworkCreate,
    DockerVolumeInfo,
	DockerVolumeDetail,
    DockerVolumeCreate,
    DockerRegistryInfo,
    DockerRegistryUpsert,
    // Swarm
    SwarmInfo,
    SwarmNodeInfo,
	SwarmNodeDetail,
    SwarmServiceInfo,
    SwarmServiceDetail,
    SwarmTask,
    SwarmCreateService,
    SwarmServiceCompose,
    // Compose
    ComposeDeployResult,
    ComposeDeploy,
    ComposeRedeploy,
    // Cron
    CronJob,
    CronJobCreate,
    CronJobList,
    CronJobLogList,
    CronTypeInfo,
    // SSH
    SSHHostInfo,
    SSHHostUpsert,
} from './types'

// API 服务类，统一管理所有 API 请求
class ApiService {
    // ==================== Overview 系统概览 ====================

    overviewProbe() {
        return http.get<SystemProbe>('overview/probe')
    }

    overviewMonitor(params: { type?: 'host'; id?: string; since: 0 }): Promise<{ payload?: MonitorHostRecord | null }>
    overviewMonitor(params: { type: 'container'; id: string; since: 0 }): Promise<{ payload?: MonitorContainerRecord | null }>
    overviewMonitor(params?: { type?: 'host'; id?: string; since?: number }): Promise<{ payload?: MonitorHostRecord[] }>
    overviewMonitor(params?: { type: 'container'; id: string; since?: number }): Promise<{ payload?: MonitorContainerRecord[] }>
    overviewMonitor(params?: { type?: 'host' | 'container'; id?: string; since?: number }) {
        return http.get<MonitorHostRecord | MonitorContainerRecord | MonitorHostRecord[] | MonitorContainerRecord[]>('overview/monitor', { params })
    }

    overviewUpgrade() {
        return http.post('overview/upgrade', {})
    }

    // ==================== System 系统相关 ====================

    systemConfig(params?: Record<string, string>) {
        return http.get<AllConfig>('system/config', { params })
    }

    systemConfigUpdate(data: Partial<AllConfig>) {
        return http.put<void>('system/config', data)
    }

    systemAuditLogs(params?: { username?: string; limit?: number }) {
        return http.get<AuditLog[]>('system/audit/logs', { params })
    }

    // ==================== Account 账户相关 ====================

    accountLogin(data: AuthLogin) {
        return http.post<AuthLoginResult>('account/login', data)
    }

    accountOIDCExchange(data: OIDCExchange) {
        return http.post<AuthLoginResult>('account/oidc/exchange', data)
    }

    accountInfo() {
        return http.get<AuthInfo>('account/info')
    }

    accountRouteList() {
        return http.get<Route[]>('account/routes')
    }

    accountMemberList() {
        return http.get<MemberInfo[]>('account/members')
    }

    accountMemberCreate(data: MemberUpsert) {
        return http.post<void>('account/member', data)
    }

    accountMemberUpdate(username: string, data: MemberUpsert) {
        return http.put<void>(`account/member/${encodeURIComponent(username)}`, data)
    }

    accountMemberDelete(username: string) {
        return http.delete<void>(`account/member/${encodeURIComponent(username)}`)
    }

    accountTokenCreate(data: ApiTokenCreate) {
        return http.post<ApiTokenResult>('account/token', data)
    }

    accountPasswordChange(data: ChangePassword) {
        return http.put<void>('account/password', data)
    }

    // ==================== Filer 文件管理相关 ====================

    filerList(path: string) {
        return http.get<FilerList>('filer/list', { params: { path } })
    }

    filerDelete(path: string) {
        return http.post<void>('filer/delete', { path })
    }

    filerRename(path: string, target: string) {
        return http.post<void>('filer/rename', { path, target })
    }

    filerMkdir(path: string) {
        return http.post<void>('filer/mkdir', { path })
    }

    filerCreate(path: string, content = '') {
        return http.post<void>('filer/create', { path, content })
    }

    filerRead(path: string) {
        return http.get<FilerRead>('filer/read', { params: { path } })
    }

    filerModify(path: string, content: string) {
        return http.post<void>('filer/modify', { path, content })
    }

    filerChmod(path: string, mode: string) {
        return http.post<void>('filer/chmod', { path, mode })
    }

    filerZip(path: string) {
        return http.post<void>('filer/zip', { path })
    }

    filerUnzip(path: string) {
        return http.post<void>('filer/unzip', { path })
    }

    filerUpload(formData: FormData, config: AxiosRequestConfig = {}) {
        return http.post<void>('filer/upload', formData, {
            headers: {
                'Content-Type': 'multipart/form-data'
            },
            ...config
        })
    }

    filerDownload(path: string) {
        return httpBlob.get('filer/download', { params: { path }, responseType: 'blob' })
    }

    filerDownloadURL(path: string, token = '', inline = false) {
        const params = new URLSearchParams({ path })
        if (inline) params.set('inline', '1')
        if (token) params.set('token', token)
        return `api/filer/download?${params.toString()}`
    }

    // ==================== APISIX 管理相关 ====================

    // 路由管理
    apisixRouteList() {
        return http.get<ApisixRoute[]>('apisix/routes')
    }

    apisixRoute(id: string) {
        return http.get<ApisixRoute>(`apisix/route/${id}`)
    }

    apisixRouteCreate(data: ApisixRoute) {
        return http.post('apisix/route', data)
    }

    apisixRouteUpdate(id: string, data: ApisixRoute) {
        return http.put(`apisix/route/${id}`, data)
    }

    apisixRouteStatus(id: string, status: number) {
        return http.patch<void>(`apisix/route/${id}/status`, { status })
    }

    apisixRouteDelete(id: string) {
        return http.delete<void>(`apisix/route/${id}`)
    }

    // Consumer 管理
    apisixConsumerList() {
        return http.get<ApisixConsumer[]>('apisix/consumers')
    }

    apisixConsumerCreate(data: ApisixConsumerCreate) {
        return http.post('apisix/consumer', data)
    }

    apisixConsumerUpdate(username: string, data: ApisixConsumerUpdate) {
        return http.put(`apisix/consumer/${username}`, data)
    }

    apisixConsumerDelete(username: string) {
        return http.delete<void>(`apisix/consumer/${username}`)
    }

    // 白名单管理
    apisixWhitelist() {
        return http.get<ApisixRoute[]>('apisix/whitelist')
    }

    apisixWhitelistRevoke(payload: ApisixRevokeWhitelist) {
        return http.post<void>('apisix/whitelist/revoke', payload)
    }

    // PluginConfig 管理
    apisixPluginConfigList() {
        return http.get<ApisixPluginConfig[]>('apisix/plugin-configs')
    }

    apisixPluginConfig(id: string) {
        return http.get<ApisixPluginConfig>(`apisix/plugin-config/${id}`)
    }

    apisixPluginConfigCreate(data: ApisixPluginConfigPayload) {
        return http.post('apisix/plugin-config', data)
    }

    apisixPluginConfigUpdate(id: string, data: ApisixPluginConfigPayload) {
        return http.put(`apisix/plugin-config/${id}`, data)
    }

    apisixPluginConfigDelete(id: string) {
        return http.delete<void>(`apisix/plugin-config/${id}`)
    }

    // Upstream 管理
    apisixUpstreamList() {
        return http.get<ApisixUpstream[]>('apisix/upstreams')
    }

    apisixUpstream(id: string) {
        return http.get<ApisixUpstream>(`apisix/upstream/${id}`)
    }

    apisixUpstreamCreate(data: ApisixUpstreamCreate) {
        return http.post('apisix/upstream', data)
    }

    apisixUpstreamUpdate(id: string, data: ApisixUpstreamUpdate) {
        return http.put(`apisix/upstream/${id}`, data)
    }

    apisixUpstreamDelete(id: string) {
        return http.delete<void>(`apisix/upstream/${id}`)
    }

    // SSL 管理
    apisixSSLList() {
        return http.get<ApisixSSL[]>('apisix/ssls')
    }

    apisixSSL(id: string) {
        return http.get<ApisixSSL>(`apisix/ssl/${id}`)
    }

    apisixSSLCreate(data: ApisixSSLCreate) {
        return http.post('apisix/ssl', data)
    }

    apisixSSLUpdate(id: string, data: ApisixSSLUpdate) {
        return http.put(`apisix/ssl/${id}`, data)
    }

    apisixSSLDelete(id: string) {
        return http.delete<void>(`apisix/ssl/${id}`)
    }

    apisixPluginList() {
        return http.get<Record<string, { schema: Record<string, unknown> }>>('apisix/plugins')
    }

    // ==================== Caddy 网关相关 ====================

    caddyInfo() {
        return http.get<CaddyInfo>('caddy/info')
    }

    caddyGlobal() {
        return http.get<CaddyGlobal>('caddy/global')
    }

    caddyGlobalUpdate(data: CaddyGlobal) {
        return http.put<void>('caddy/global', data)
    }

    caddyConfig() {
        return http.get<unknown>('caddy/config')
    }

    caddyConfigLoad(config: unknown) {
        return http.post<void>('caddy/config', { config })
    }

    caddyRouteList(server?: string) {
        return http.get<CaddyRoute[]>('caddy/routes', { params: server ? { server } : {} })
    }

    caddyRoute(index: number, server?: string) {
        return http.get<CaddyRoute>(`caddy/route/${index}`, { params: server ? { server } : {} })
    }

    caddyRouteCreate(data: CaddyRouteUpsert, server?: string) {
        return http.post<{ index: number }>('caddy/route', data, { params: server ? { server } : {} })
    }

    caddyRouteUpdate(index: number, data: CaddyRouteUpsert, server?: string) {
        return http.put<void>(`caddy/route/${index}`, data, { params: server ? { server } : {} })
    }

    caddyRouteDelete(index: number, server?: string) {
        return http.delete<void>(`caddy/route/${index}`, { params: server ? { server } : {} })
    }

    caddyCertList() {
        return http.get<CaddyCert[]>('caddy/certs')
    }

    caddyCertCreate(data: CaddyCert) {
        return http.post<void>('caddy/cert', data)
    }

    caddyCertUpdate(key: string, data: CaddyCert) {
        return http.put<void>(`caddy/cert/${encodeURIComponent(key)}`, data)
    }

    caddyCertDelete(key: string) {
        return http.delete<void>(`caddy/cert/${encodeURIComponent(key)}`)
    }

    // ==================== Docker 服务相关 ====================

    // Docker 概览信息
    dockerInfo() {
        return http.get<DockerInfo>('docker/info')
    }

    // 容器管理
    dockerContainerList(all = false) {
        return http.get<DockerContainerInfo[]>('docker/containers', { params: { all } })
    }

    dockerContainer(id: string) {
        return http.get<DockerContainerDetail>(`docker/container/${id}`)
    }

    dockerContainerAction(id: string, action: string) {
        return http.post<void>(`docker/container/${id}/action`, { action })
    }

    dockerContainerCreate(data: DockerContainerCreate) {
        return http.post<DockerContainerCreateResult>('docker/container', data)
    }

    dockerContainerLogs(id: string, tail = '100') {
        return http.get<{ logs: string[] }>(`docker/container/${id}/logs`, { params: { tail } })
    }

    dockerContainerStats(id: string) {
        return http.get<DockerContainerStats>(`docker/container/${id}/stats`)
    }

    dockerContainerCompose(name: string) {
        return http.get<DockerContainerCompose>(`compose/docker/${name}`)
    }

    // 镜像管理
    dockerImageList(all = false) {
        return http.get<DockerImageInfo[]>('docker/images', { params: { all } })
    }

    dockerImage(id: string) {
	return http.get<DockerImageDetail>(`docker/image/${id}`)
    }

    dockerImageAction(id: string, action: string) {
        return http.post<void>(`docker/image/${id}/action`, { action })
    }

    dockerImageTag(id: string, repoTag: string) {
        return http.post<void>(`docker/image/${id}/tag`, { repoTag })
    }

    dockerImageSearch(name: string) {
        return http.get<DockerImageSearchResult[]>('docker/images/search', { params: { name } })
    }

    dockerImageBuild(dockerfile: string, tag = '') {
        return http.post<void>('docker/image/build', { dockerfile, tag })
    }

    dockerImagePrune(data: DockerImagePrune = {}) {
        return http.post<DockerImagePruneResult>('docker/image/prune', data)
    }

    dockerImagePush(image: string, registryUrl: string, namespace: string) {
        return http.post<void>('docker/image/push', { image, registryUrl, namespace })
    }

    dockerImagePull(image: string, registryUrl: string, namespace: string) {
        return http.post<void>('docker/image/pull', { image, registryUrl, namespace })
    }

    // 网络管理
    dockerNetworkList() {
        return http.get<DockerNetworkInfo[]>('docker/networks')
    }

    dockerNetwork(id: string) {
	return http.get<DockerNetworkDetail>(`docker/network/${id}`)
    }

    dockerNetworkAction(id: string, action: string) {
        return http.post<void>(`docker/network/${id}/action`, { action })
    }

    dockerNetworkCreate(data: DockerNetworkCreate) {
        return http.post('docker/network', data)
    }

    // 卷管理
    dockerVolumeList() {
        return http.get<DockerVolumeInfo[]>('docker/volumes')
    }

    dockerVolume(name: string) {
	return http.get<DockerVolumeDetail>(`docker/volume/${encodeURIComponent(name)}`)
    }

    dockerVolumeAction(name: string, action: string) {
        return http.post<void>(`docker/volume/${encodeURIComponent(name)}/action`, { action })
    }

    dockerVolumeCreate(data: DockerVolumeCreate) {
        return http.post('docker/volume', data)
    }

    // 镜像仓库管理
    dockerRegistryList() {
        return http.get<DockerRegistryInfo[]>('docker/registries')
    }

    dockerRegistryCreate(data: DockerRegistryUpsert) {
        return http.post<void>('docker/registry', data)
    }

    dockerRegistryUpdate(url: string, data: DockerRegistryUpsert) {
        return http.put<void>('docker/registry', data, { params: { url } })
    }

    dockerRegistryDelete(url: string) {
        return http.delete<void>('docker/registry', { params: { url } })
    }

    // ==================== Docker Swarm 管理相关 ====================

    swarmInfo() {
        return http.get<SwarmInfo>('swarm/info')
    }

    swarmNodeList() {
        return http.get<SwarmNodeInfo[]>('swarm/nodes')
    }

    swarmJoinToken() {
        return http.get<{ worker: string; manager: string }>('swarm/token')
    }

    swarmNode(id: string) {
	return http.get<SwarmNodeDetail>(`swarm/node/${id}`)
    }

    swarmNodeAction(id: string, action: string) {
        return http.post<void>(`swarm/node/${id}/action`, { action })
    }

    // 服务管理
    swarmServiceList() {
        return http.get<SwarmServiceInfo[]>('swarm/services')
    }

    swarmService(id: string) {
        return http.get<SwarmServiceDetail>(`swarm/service/${id}`)
    }

    swarmServiceAction(id: string, action: string, replicas?: number) {
        return http.post<void>(`swarm/service/${id}/action`, { action, replicas })
    }

    swarmServiceCreate(data: SwarmCreateService) {
        return http.post('swarm/service', data)
    }

    swarmServiceRedeploy(id: string) {
        return http.post<void>(`swarm/service/${id}/force-update`)
    }

    swarmServiceCompose(name: string) {
        return http.get<SwarmServiceCompose>(`compose/swarm/${name}`)
    }

    swarmServiceLogs(id: string, tail = '100') {
        return http.get<{ logs: string[] }>(`swarm/service/${id}/logs`, { params: { tail } })
    }

    swarmTaskList(serviceID = '') {
        return http.get<SwarmTask[]>('swarm/tasks', { params: serviceID ? { serviceID } : {} })
    }

    // ==================== Compose 部署 ====================

    composeDockerDeploy(data: ComposeDeploy) {
        return http.post<ComposeDeployResult>('compose/docker/deploy', this.composeDeployForm(data))
    }

    composeSwarmDeploy(data: ComposeDeploy) {
        return http.post<ComposeDeployResult>('compose/swarm/deploy', this.composeDeployForm(data))
    }

    private composeDeployForm(data: ComposeDeploy) {
        const form = new FormData()
        form.append('content', data.content)
        // 文件优先，二者互斥
        if (data.initFile) {
            form.append('initFile', data.initFile)
        } else if (data.initURL) {
            form.append('initURL', data.initURL)
        }
        return form
    }

    composeDockerRedeploy(name: string, data: ComposeRedeploy) {
        return http.post<ComposeDeployResult>(`compose/docker/${name}/redeploy`, data)
    }

    composeSwarmRedeploy(name: string, data: ComposeRedeploy) {
        return http.post<ComposeDeployResult>(`compose/swarm/${name}/redeploy`, data)
    }

    // ==================== Cron 计划任务 ====================

    cronTypes() {
        return http.get<{ types: CronTypeInfo[] }>('cron/types')
    }

    cronJobList() {
        return http.get<CronJobList>('cron/jobs')
    }

    cronJobCreate(data: CronJobCreate) {
        return http.post<{ job: CronJob }>('cron/jobs', data)
    }

    cronJobUpdate(id: string, data: CronJobCreate) {
        return http.put<{ job: CronJob }>(`cron/jobs/${id}`, data)
    }

    cronJobDelete(id: string) {
        return http.delete<void>(`cron/jobs/${id}`)
    }

    cronJobRun(id: string) {
        return http.post<void>(`cron/jobs/${id}/run`, {})
    }

    cronJobStatus(id: string, enabled: boolean) {
        return http.post<void>(`cron/jobs/${id}/enable`, { enabled })
    }

    cronJobLogs(id: string, limit = 50) {
        return http.get<CronJobLogList>(`cron/jobs/${id}/logs`, { params: { limit } })
    }

    // ==================== SSH 主机管理 ====================

    sshHostList() {
        return http.get<SSHHostInfo[]>('ssh/hosts')
    }

    sshHost(id: string) {
        return http.get<SSHHostInfo>(`ssh/host/${id}`)
    }

    sshHostCreate(data: SSHHostUpsert) {
        return http.post<SSHHostInfo>('ssh/host', data)
    }

    sshHostUpdate(id: string, data: SSHHostUpsert) {
        return http.put<SSHHostInfo>(`ssh/host/${id}`, data)
    }

    sshHostDelete(id: string) {
        return http.delete<void>(`ssh/host/${id}`)
    }
}

// 导出单例实例
export default new ApiService()
