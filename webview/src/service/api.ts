import type { AxiosRequestConfig } from 'axios'
import { http, httpBlob, absUrl } from './client'
import type {
    // Overview
    BootstrapData,
    SystemVersionInfo,
    MonitorHostRecord,
    MonitorContainerRecord,
    // System
    AllConfig,
    AuditLog,
    // Account
    AuthLogin,
    AuthLoginResult,
    OIDCExchange,
    MemberInfo,
    MemberUpsert,
    Route,
    ApiTokenCreate,
    ApiTokenResult,
    ChangePassword,
    TOTPBeginResult,
    TOTPVerify,
    TwoFactorStatus,
    // Passkey
    PasskeyBeginLoginData,
    PasskeyBeginData,
    PasskeyCredentialCreationOptionsJSON,
    PasskeyCredentialRequestOptionsJSON,
    PasskeyLoginCredential,
    PasskeyLoginResult,
    PasskeyRegisterCredential,
    PasskeyCredential,
    // Filer
    FilerList,
    FilerRead,
    // APISIX
    ApisixRoute,
    ApisixConsumer,
    ApisixConsumerCreate,
    ApisixConsumerUpdate,
    ApisixPluginConfig,
    ApisixPluginConfigUpsert,
    ApisixUpstream,
    ApisixUpstreamCreate,
    ApisixUpstreamUpdate,
    ApisixSSL,
    ApisixSSLCreate,
    ApisixSSLUpdate,
    ApisixWhitelistCreate,
    ApisixWhitelistUserCreate,
    // Caddy
    CaddyInfo,
    CaddyGlobal,
    CaddyRoute,
    CaddyRouteUpsert,
    CaddyCert,
    CaddyBasicAuthRoute,
    CaddyBasicAuthUserCreate,
    CaddyBasicAuthConfigUpdate,
    // Docker
    DockerInfo,
    ContainerFileInfo,
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
    SSHCredentialInfo,
    SSHCredentialUpsert,
    SSHHostInfo,
    SSHHostUpsert,
    SFTPListResult,
    SFTPRename,
    SFTPMkdir,
    SFTPChmod,
    SFTPChown,
    SFTPWrite,
} from './types'

// API 服务类，统一管理所有 API 请求
class ApiService {
    // ==================== Overview 系统概览 ====================

    overviewBootstrap() {
        return http.get<BootstrapData>('overview/bootstrap')
    }

    overviewMonitor(params: { type?: 'host'; id?: string; since: 0 }): Promise<{ payload?: MonitorHostRecord | null }>
    overviewMonitor(params: { type: 'container'; id: string; since: 0 }): Promise<{ payload?: MonitorContainerRecord | null }>
    overviewMonitor(params?: { type?: 'host'; id?: string; since?: number }): Promise<{ payload?: MonitorHostRecord[] }>
    overviewMonitor(params?: { type: 'container'; id: string; since?: number }): Promise<{ payload?: MonitorContainerRecord[] }>
    overviewMonitor(params?: { type?: 'host' | 'container'; id?: string; since?: number }): Promise<{ payload?: MonitorHostRecord | MonitorContainerRecord | MonitorHostRecord[] | MonitorContainerRecord[] | null }> {
        return http.get<MonitorHostRecord | MonitorContainerRecord | MonitorHostRecord[] | MonitorContainerRecord[] | null>('overview/monitor', { params })
    }

    overviewVersion() {
        return http.get<SystemVersionInfo>('overview/version')
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

    accountTwoFactorStatus() {
        return http.get<TwoFactorStatus>('account/2fa/status')
    }

    accountTOTPBegin() {
        return http.post<TOTPBeginResult>('account/2fa/totp/begin', {})
    }

    accountTOTPEnable(data: TOTPVerify) {
        return http.post<void>('account/2fa/totp/enable', data)
    }

    accountTOTPDisable(data: TOTPVerify) {
        return http.post<void>('account/2fa/totp/disable', data)
    }

    // ==================== Passkey 认证相关 ====================

    accountPasskeyRegisterBegin(data: { displayName?: string } = {}) {
        return http.post<PasskeyBeginData<PasskeyCredentialCreationOptionsJSON>>(
            'account/passkey/register/begin',
            data,
        )
    }

    accountPasskeyRegisterFinish(sessionId: string, credential: PasskeyRegisterCredential) {
        return http.post<void>(`account/passkey/register/finish?sessionId=${encodeURIComponent(sessionId)}`, credential)
    }

    accountPasskeyLoginBegin(data: PasskeyBeginLoginData) {
        return http.post<PasskeyBeginData<PasskeyCredentialRequestOptionsJSON>>(
            'account/passkey/login/begin',
            data,
        )
    }

    accountPasskeyLoginFinish(sessionId: string, credential: PasskeyLoginCredential) {
        return http.post<PasskeyLoginResult>(`account/passkey/login/finish?sessionId=${encodeURIComponent(sessionId)}`, credential)
    }

    accountPasskeyListCredentials() {
        return http.get<PasskeyCredential[]>('account/passkey/credentials')
    }

    accountPasskeyRenameCredential(credentialId: string, displayName: string) {
        return http.put<void>(`account/passkey/credential/${credentialId}`, { displayName })
    }

    accountPasskeyDeleteCredential(credentialId: string) {
        return http.delete<void>(`account/passkey/credential/${credentialId}`)
    }

    // ==================== Filer 文件管理相关 ====================

    filerList(path: string) {
        return http.get<FilerList>('filer/files', { params: { path } })
    }

    filerDelete(path: string) {
        return http.delete<void>('filer/file', { params: { path } })
    }

    filerRename(path: string, target: string) {
        return http.post<void>('filer/rename', { path, target })
    }

    filerMkdir(path: string, config?: AxiosRequestConfig) {
        return http.post<void>('filer/dir', { path }, config)
    }

    filerCreate(path: string, content = '') {
        return http.post<void>('filer/file', { path, content })
    }

    filerRead(path: string) {
        return http.get<FilerRead>('filer/file', { params: { path } })
    }

    filerModify(path: string, content: string) {
        return http.put<void>('filer/file', { path, content })
    }

    filerChmod(path: string, mode: string) {
        return http.put<void>('filer/chmod', { path, mode })
    }

    filerDirSize(path: string) {
        return http.get<{ path: string; size: number }>('filer/dir-size', { params: { path } })
    }

    filerZip(path: string) {
        return http.post<void>('filer/zip', { path })
    }

    filerUnzip(path: string, targetDir?: string) {
        return http.post<void>('filer/unzip', { path, targetDir: targetDir || undefined })
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
        return absUrl(`filer/download?${params.toString()}`)
    }

    // ==================== APISIX 管理相关 ====================

    // 路由管理
    apisixRouteList() {
        return http.get<ApisixRoute[]>('apisix/routes')
    }

    apisixRouteInspect(id: string) {
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
        return http.post<ApisixConsumer>('apisix/consumer', data)
    }

    apisixConsumerUpdate(username: string, data: ApisixConsumerUpdate) {
        return http.put(`apisix/consumer/${username}`, data)
    }

    apisixConsumerDelete(username: string) {
        return http.delete<void>(`apisix/consumer/${username}`)
    }

    // 访问授权管理
    apisixWhitelistInspect() {
        return http.get<ApisixRoute[]>('apisix/whitelist')
    }

    apisixWhitelistCreate(payload: ApisixWhitelistCreate) {
        return http.post<ApisixRoute>('apisix/whitelist', payload)
    }

    apisixWhitelistUserCreate(payload: ApisixWhitelistUserCreate) {
        return http.post<ApisixRoute>('apisix/whitelist/user', payload)
    }

    // PluginConfig 管理
    apisixPluginConfigList() {
        return http.get<ApisixPluginConfig[]>('apisix/plugin-configs')
    }

    apisixPluginConfigInspect(id: string) {
        return http.get<ApisixPluginConfig>(`apisix/plugin-config/${id}`)
    }

    apisixPluginConfigCreate(data: ApisixPluginConfigUpsert) {
        return http.post('apisix/plugin-config', data)
    }

    apisixPluginConfigUpdate(id: string, data: ApisixPluginConfigUpsert) {
        return http.put(`apisix/plugin-config/${id}`, data)
    }

    apisixPluginConfigDelete(id: string) {
        return http.delete<void>(`apisix/plugin-config/${id}`)
    }

    // Upstream 管理
    apisixUpstreamList() {
        return http.get<ApisixUpstream[]>('apisix/upstreams')
    }

    apisixUpstreamInspect(id: string) {
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

    apisixSSLInspect(id: string) {
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

    caddyInfoInspect() {
        return http.get<CaddyInfo>('caddy/info')
    }

    caddyGlobalInspect() {
        return http.get<CaddyGlobal>('caddy/global')
    }

    caddyGlobalUpdate(data: CaddyGlobal) {
        return http.put<void>('caddy/global', data)
    }

    caddyConfigInspect() {
        return http.get<unknown>('caddy/config')
    }

    caddyConfigLoad(config: unknown) {
        return http.post<void>('caddy/config', { config })
    }

    caddyRouteList(server?: string) {
        return http.get<CaddyRoute[]>('caddy/routes', { params: server ? { server } : {} })
    }

    caddyRouteInspect(index: number, server?: string) {
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

    // ─── Caddy Basic Auth ───

    caddyBasicAuthList() {
        return http.get<CaddyBasicAuthRoute[]>('caddy/basic-auth')
    }

    caddyBasicAuthUserCreate(routeIndex: number, data: CaddyBasicAuthUserCreate) {
        return http.post<void>(`caddy/basic-auth/${routeIndex}/users`, data)
    }

    caddyBasicAuthUserDelete(routeIndex: number, username: string) {
        return http.delete<void>(`caddy/basic-auth/${routeIndex}/users/${encodeURIComponent(username)}`)
    }

    caddyBasicAuthConfigUpdate(routeIndex: number, data: CaddyBasicAuthConfigUpdate) {
        return http.put<void>(`caddy/basic-auth/${routeIndex}/config`, data)
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

    dockerContainerInspect(id: string) {
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

    // ==================== 容器文件管理 ====================

    dockerContainerFileLs(id: string, path: string) {
        return http.get<{ path: string; files: ContainerFileInfo[] }>(`docker/container/${id}/file/ls`, { params: { path } })
    }

    dockerContainerFileDownload(id: string, path: string, onProgress?: (percent: number) => void) {
        return httpBlob.get(`docker/container/${id}/file/download`, {
            params: { path },
            responseType: 'blob',
            onDownloadProgress: onProgress
                ? (e) => { if (e.total) onProgress(Math.round((e.loaded / e.total) * 100)) }
                : undefined,
        })
    }

    dockerContainerFileDownloadURL(id: string, path: string, token = '') {
        const params = new URLSearchParams({ path })
        if (token) params.set('token', token)
        return absUrl(`docker/container/${id}/file/download?${params.toString()}`)
    }

    dockerContainerFileUpload(id: string, path: string, formData: FormData, onProgress?: (percent: number) => void, config: AxiosRequestConfig = {}) {
        return http.post<void>(`docker/container/${id}/file/upload`, formData, {
            ...config,
            params: { ...config.params, path },
            headers: { ...config.headers, 'Content-Type': 'multipart/form-data' },
            onUploadProgress: onProgress
                ? (e) => { if (e.total) onProgress(Math.round((e.loaded / e.total) * 100)) }
                : undefined,
        })
    }

    dockerContainerFileRemove(id: string, path: string, recursive = false) {
        return http.delete<void>(`docker/container/${id}/file/rm`, { params: { path, recursive: recursive || undefined } })
    }

    dockerContainerFileMkdir(id: string, path: string, config?: AxiosRequestConfig) {
        return http.post<void>(`docker/container/${id}/file/mkdir`, { path }, config)
    }

    dockerContainerFileRename(id: string, oldPath: string, newPath: string) {
        return http.post<void>(`docker/container/${id}/file/rename`, { oldPath, newPath })
    }

    dockerContainerFileRead(id: string, path: string) {
        return http.get<{ content: string }>(`docker/container/${id}/file/read`, { params: { path } })
    }

    dockerContainerFileWrite(id: string, path: string, content: string) {
        return http.post<void>(`docker/container/${id}/file/write`, { path, content })
    }

    dockerContainerFileChmod(id: string, path: string, mode: string) {
        return http.post<void>(`docker/container/${id}/file/chmod`, { path, mode })
    }

    composeDockerInspect(name: string) {
        return http.get<DockerContainerCompose>(`compose/docker/${name}`)
    }

    // 镜像管理
    dockerImageList(all = false) {
        return http.get<DockerImageInfo[]>('docker/images', { params: { all } })
    }

    dockerImageInspect(id: string) {
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

    dockerNetworkInspect(id: string) {
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

    dockerVolumeInspect(name: string) {
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
        return http.put<void>(`docker/registry?url=${encodeURIComponent(url)}`, data)
    }

    dockerRegistryDelete(url: string) {
        return http.delete<void>(`docker/registry?url=${encodeURIComponent(url)}`)
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

    swarmNodeInspect(id: string) {
	return http.get<SwarmNodeDetail>(`swarm/node/${id}`)
    }

    swarmNodeAction(id: string, action: string) {
        return http.post<void>(`swarm/node/${id}/action`, { action })
    }

    // 服务管理
    swarmServiceList() {
        return http.get<SwarmServiceInfo[]>('swarm/services')
    }

    swarmServiceInspect(id: string) {
        return http.get<SwarmServiceDetail>(`swarm/service/${id}`)
    }

    swarmServiceAction(id: string, action: string, replicas?: number) {
        return http.post<void>(`swarm/service/${id}/action`, { action, replicas })
    }

    swarmServiceCreate(data: SwarmCreateService) {
        return http.post('swarm/service', data)
    }

    composeSwarmInspect(name: string) {
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
        return http.post<ComposeDeployResult>('compose/docker', this.composeDeployForm(data))
    }

    composeSwarmDeploy(data: ComposeDeploy) {
        return http.post<ComposeDeployResult>('compose/swarm', this.composeDeployForm(data))
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
        return http.put<ComposeDeployResult>(`compose/docker/${name}`, data)
    }

    composeSwarmRedeploy(name: string, data: ComposeRedeploy) {
        return http.put<ComposeDeployResult>(`compose/swarm/${name}`, data)
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

    cronJobStatusPatch(id: string, enabled: boolean) {
        return http.patch<void>(`cron/jobs/${id}`, { enabled })
    }

    cronJobLogs(id: string, limit = 50) {
        return http.get<CronJobLogList>(`cron/jobs/${id}/logs`, { params: { limit } })
    }

    // ==================== SSH 凭据管理 ====================

    sshCredentialList() {
        return http.get<SSHCredentialInfo[]>('ssh/credentials')
    }

    sshCredentialInspect(id: string) {
        return http.get<SSHCredentialInfo>(`ssh/credential/${id}`)
    }

    sshCredentialCreate(data: SSHCredentialUpsert) {
        return http.post<SSHCredentialInfo>('ssh/credential', data)
    }

    sshCredentialUpdate(id: string, data: SSHCredentialUpsert) {
        return http.put<SSHCredentialInfo>(`ssh/credential/${id}`, data)
    }

    sshCredentialDelete(id: string) {
        return http.delete<void>(`ssh/credential/${id}`)
    }

    // ==================== SSH 主机管理 ====================

    sshHostList() {
        return http.get<SSHHostInfo[]>('ssh/hosts')
    }

    sshHostInspect(id: string) {
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

    // ==================== SFTP 文件管理 ====================

    sftpList(hostId: string, path: string) {
        return http.get<SFTPListResult>(`sftp/${hostId}/ls`, { params: { path } })
    }

    sftpUpload(hostId: string, path: string, formData: FormData, onProgress?: (percent: number) => void, config: AxiosRequestConfig = {}) {
        return http.post<void>(`sftp/${hostId}/upload`, formData, {
            ...config,
            params: { ...config.params, path },
            headers: { ...config.headers, 'Content-Type': 'multipart/form-data' },
            onUploadProgress: onProgress
                ? (e) => { if (e.total) onProgress(Math.round((e.loaded / e.total) * 100)) }
                : undefined,
        })
    }

    sftpDownload(hostId: string, path: string, onProgress?: (percent: number) => void) {
        return httpBlob.get(`sftp/${hostId}/download`, {
            params: { path },
            responseType: 'blob',
            onDownloadProgress: onProgress
                ? (e) => { if (e.total) onProgress(Math.round((e.loaded / e.total) * 100)) }
                : undefined,
        })
    }

    sftpDownloadURL(hostId: string, path: string, token = '') {
        const params = new URLSearchParams({ path })
        if (token) params.set('token', token)
        return absUrl(`sftp/${hostId}/download?${params.toString()}`)
    }

    sftpRemove(hostId: string, path: string, recursive = false) {
        return http.delete<void>(`sftp/${hostId}/rm`, { params: { path, recursive: recursive || undefined } })
    }

    sftpMkdir(hostId: string, data: SFTPMkdir, config?: AxiosRequestConfig) {
        return http.post<void>(`sftp/${hostId}/mkdir`, data, config)
    }

    sftpRename(hostId: string, data: SFTPRename) {
        return http.post<void>(`sftp/${hostId}/rename`, data)
    }

    // ─── SFTP 文件操作 ───
    sftpFileChmod(hostId: string, data: SFTPChmod) {
        return http.post<void>(`sftp/${hostId}/chmod`, data)
    }

    sftpFileChown(hostId: string, data: SFTPChown) {
        return http.post<void>(`sftp/${hostId}/chown`, data)
    }

    sftpRead(hostId: string, path: string) {
        return http.get<{ content: string }>(`sftp/${hostId}/read`, { params: { path } })
    }

    sftpWrite(hostId: string, data: SFTPWrite) {
        return http.post<void>(`sftp/${hostId}/write`, data)
    }

    sftpDirSize(hostId: string, path: string) {
        return http.get<{ path: string; size: number }>(`sftp/${hostId}/dir-size`, { params: { path } })
    }
}

// 导出单例实例
export default new ApiService()
