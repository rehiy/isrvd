import type { AxiosRequestConfig } from 'axios'
import { http, httpBlob } from './axios'
import type {
    DockerContainerCreateRequest,
    DockerContainerInfo, DockerContainerStatsResponse,
    DockerImageInfo, DockerImageInspectResponse, DockerImageSearchResult,
    DockerNetworkInfo, DockerNetworkInspectResponse, DockerNetworkCreateRequest,
    DockerVolumeInfo, DockerVolumeInspectResponse, DockerVolumeCreateRequest,
    DockerRegistryInfo, DockerRegistryUpsertRequest,
    SwarmInfo, SwarmNodeDTO, SwarmNodeInspect,
    SwarmServiceInfo, SwarmServiceDetail, SwarmTask,
    SwarmCreateServiceRequest,
    ApisixRoute, ApisixConsumer, ApisixCreateConsumerRequest, ApisixUpdateConsumerRequest,
    ApisixPluginConfig, ApisixUpstream,
    SystemProbeResponse, DockerInfo,
    FilerListResponse, FilerReadResponse,
    AuthLoginResponse, AuthInfoResponse,
    SystemAllSettings,
    SystemMemberInfo, SystemMemberUpsertRequest,
    ComposeDeployResult,
    SystemStat
} from './types'

// API 服务类，统一管理所有 API 请求
class ApiService {
    // 认证相关
    authInfo() {
        return http.get<AuthInfoResponse>('/api/auth/info')
    }

    login(data: { username: string; password: string }) {
        return http.post<AuthLoginResponse>('/api/auth/login', data)
    }

    logout() {
        return http.post<void>('/api/auth/logout')
    }

    // 文件管理相关
    list(path: string) {
        return http.post<FilerListResponse>('/api/filer/list', { path })
    }

    delete(path: string) {
        return http.post<void>('/api/filer/delete', { path })
    }

    rename(path: string, target: string) {
        return http.post<void>('/api/filer/rename', { path, target })
    }

    mkdir(path: string) {
        return http.post<void>('/api/filer/mkdir', { path })
    }

    create(path: string, content = '') {
        return http.post<void>('/api/filer/create', { path, content })
    }

    // 文件编辑相关
    read(path: string) {
        return http.post<FilerReadResponse>('/api/filer/read', { path })
    }

    modify(path: string, content: string) {
        return http.post<void>('/api/filer/modify', { path, content })
    }

    chmod(path: string, mode: string) {
        return http.post<void>('/api/filer/chmod', { path, mode })
    }

    // 压缩解压
    zip(path: string) {
        return http.post<void>('/api/filer/zip', { path })
    }

    unzip(path: string) {
        return http.post<void>('/api/filer/unzip', { path })
    }

    // 文件上传
    upload(formData: FormData, config: AxiosRequestConfig = {}) {
        return http.post<void>('/api/filer/upload', formData, {
            headers: {
                'Content-Type': 'multipart/form-data'
            },
            ...config
        })
    }

    // 文件下载
    download(path: string) {
        return httpBlob.post('/api/filer/download', { path }, { responseType: 'blob' })
    }

    // ==================== 服务探测相关 ====================

    serviceProbe() {
        return http.get<SystemProbeResponse>('/api/system/probe')
    }

    // ==================== 系统信息相关 ====================

    systemStat() {
        return http.get<SystemStat>('/api/system/stats')
    }

    // ==================== Docker 服务相关 ====================

    // Docker 概览信息
    dockerInfo() {
        return http.get<DockerInfo>('/api/docker/info')
    }

    // 容器管理
    listContainers(all = false) {
        return http.get<DockerContainerInfo[]>('/api/docker/containers', { params: { all } })
    }

    containerAction(id: string, action: string) {
        return http.post<void>(`/api/docker/container/${id}/action`, { action })
    }

    createContainer(data: DockerContainerCreateRequest) {
        return http.post('/api/docker/container/create', data)
    }

    containerLogs(id: string, tail = '100') {
        return http.get<{ logs: string[] }>(`/api/docker/container/${id}/logs`, { params: { tail } })
    }

    containerStats(id: string) {
        return http.get<DockerContainerStatsResponse>(`/api/docker/container/${id}/stats`)
    }

    getContainerCompose(name: string) {
        return http.get<{ content: string }>(`/api/compose/docker/${name}`)
    }

    // 镜像管理
    listImages(all = false) {
        return http.get<DockerImageInfo[]>('/api/docker/images', { params: { all } })
    }

    imageInspect(id: string) {
        return http.get<DockerImageInspectResponse>(`/api/docker/image/${id}`)
    }

    imageAction(id: string, action: string) {
        return http.post<void>(`/api/docker/image/${id}/action`, { action })
    }

    pullImage(image: string, tag = '') {
        return http.post<void>('/api/docker/image/pull', { image, tag })
    }

    imageTag(id: string, repoTag: string) {
        return http.post<void>('/api/docker/image/tag', { id, repoTag })
    }

    imageSearch(term: string) {
        return http.get<DockerImageSearchResult[]>(`/api/docker/image/search/${encodeURIComponent(term)}`)
    }

    imageBuild(dockerfile: string, tag = '') {
        return http.post<void>('/api/docker/image/build', { dockerfile, tag })
    }

    // 网络管理
    listNetworks() {
        return http.get<DockerNetworkInfo[]>('/api/docker/networks')
    }

    networkInspect(id: string) {
        return http.get<DockerNetworkInspectResponse>(`/api/docker/network/${id}`)
    }

    networkAction(id: string, action: string) {
        return http.post<void>(`/api/docker/network/${id}/action`, { action })
    }

    createNetwork(data: DockerNetworkCreateRequest) {
        return http.post('/api/docker/network/create', data)
    }

    // 卷管理
    listVolumes() {
        return http.get<DockerVolumeInfo[]>('/api/docker/volumes')
    }

    volumeInspect(name: string) {
        return http.get<DockerVolumeInspectResponse>(`/api/docker/volume/${encodeURIComponent(name)}`)
    }

    volumeAction(name: string, action: string) {
        return http.post<void>(`/api/docker/volume/${encodeURIComponent(name)}/action`, { action })
    }

    createVolume(data: DockerVolumeCreateRequest) {
        return http.post('/api/docker/volume/create', data)
    }

    // 镜像仓库管理
    listRegistries() {
        return http.get<DockerRegistryInfo[]>('/api/docker/registries')
    }

    createRegistry(data: DockerRegistryUpsertRequest) {
        return http.post<void>('/api/docker/registries', data)
    }

    updateRegistry(url: string, data: DockerRegistryUpsertRequest) {
        return http.put<void>('/api/docker/registries', data, { params: { url } })
    }

    deleteRegistry(url: string) {
        return http.delete<void>('/api/docker/registries', { params: { url } })
    }

    pushImage(image: string, registryUrl: string, namespace: string) {
        return http.post<void>('/api/docker/registry/push', { image, registryUrl, namespace })
    }

    pullFromRegistry(image: string, registryUrl: string, namespace: string) {
        return http.post<void>('/api/docker/registry/pull', { image, registryUrl, namespace })
    }

    // ==================== Docker Swarm 管理相关 ====================

    swarmInfo() {
        return http.get<SwarmInfo>('/api/swarm/info')
    }

    swarmListNodes() {
        return http.get<SwarmNodeDTO[]>('/api/swarm/nodes')
    }

    swarmGetJoinTokens() {
        return http.get<{ worker: string; manager: string }>('/api/swarm/join-tokens')
    }

    swarmInspectNode(id: string) {
        return http.get<SwarmNodeInspect>(`/api/swarm/node/${id}`)
    }

    NodeDTOAction(id: string, action: string) {
        return http.post<void>(`/api/swarm/node/${id}/action`, { action })
    }

    swarmListServices() {
        return http.get<SwarmServiceInfo[]>('/api/swarm/services')
    }

    swarmInspectService(id: string) {
        return http.get<SwarmServiceDetail>(`/api/swarm/service/${id}`)
    }

    swarmServiceAction(id: string, action: string, replicas?: number) {
        return http.post<void>(`/api/swarm/service/${id}/action`, { action, replicas })
    }

    swarmListTasks(serviceID = '') {
        return http.get<SwarmTask[]>('/api/swarm/tasks', { params: serviceID ? { serviceID } : {} })
    }

    swarmCreateService(data: SwarmCreateServiceRequest) {
        return http.post('/api/swarm/service/create', data)
    }

    swarmRedeployService(id: string) {
        return http.post<void>(`/api/swarm/service/${id}/redeploy`)
    }

    swarmGetServiceCompose(name: string) {
        return http.get<{ content: string }>(`/api/compose/swarm/${name}`)
    }

    swarmServiceLogs(id: string, tail = '100') {
        return http.get<{ logs: string[] }>(`/api/swarm/service/${id}/logs`, { params: { tail } })
    }

    // ==================== APISIX 管理相关 ====================

    // 路由管理
    apisixListRoutes() {
        return http.get<ApisixRoute[]>('/api/apisix/routes')
    }

    apisixGetRoute(id: string) {
        return http.get<ApisixRoute>(`/api/apisix/route/${id}`)
    }

    apisixCreateRoute(data: ApisixRoute) {
        return http.post('/api/apisix/routes', data)
    }

    apisixUpdateRoute(id: string, data: ApisixRoute) {
        return http.put(`/api/apisix/route/${id}`, data)
    }

    apisixPatchRouteStatus(id: string, status: number) {
        return http.patch<void>(`/api/apisix/route/${id}/status`, { status })
    }

    apisixDeleteRoute(id: string) {
        return http.delete<void>(`/api/apisix/route/${id}`)
    }

    // Consumer 管理
    apisixListConsumers() {
        return http.get<ApisixConsumer[]>('/api/apisix/consumers')
    }

    apisixCreateConsumer(data: ApisixCreateConsumerRequest) {
        return http.post('/api/apisix/consumers', data)
    }

    apisixUpdateConsumer(username: string, data: ApisixUpdateConsumerRequest) {
        return http.put(`/api/apisix/consumer/${username}`, data)
    }

    apisixDeleteConsumer(username: string) {
        return http.delete<void>(`/api/apisix/consumer/${username}`)
    }

    // 白名单管理
    apisixGetWhitelist() {
        return http.get<ApisixRoute[]>('/api/apisix/whitelist')
    }

    apisixRevokeWhitelist(routeId: string, consumerName: string) {
        return http.put<void>('/api/apisix/whitelist/revoke', { route_id: routeId, consumer_name: consumerName })
    }

    // 辅助资源
    apisixListPluginConfigs() {
        return http.get<ApisixPluginConfig[]>('/api/apisix/plugin_configs')
    }

    apisixListUpstreams() {
        return http.get<ApisixUpstream[]>('/api/apisix/upstreams')
    }

    apisixListPlugins() {
        return http.get<Record<string, { schema: Record<string, unknown> }>>('/api/apisix/plugins')
    }

    // ==================== 系统设置 ====================

    getMe() {
        return http.get<AuthInfoResponse>('/api/auth/info')
    }

    getSettings() {
        return http.get<SystemAllSettings>('/api/system/settings')
    }

    updateAllSettings(data: Partial<SystemAllSettings>) {
        return http.put<void>('/api/system/settings', data)
    }

    // 成员管理
    listMembers() {
        return http.get<SystemMemberInfo[]>('/api/system/members')
    }

    createMember(data: SystemMemberUpsertRequest) {
        return http.post<void>('/api/system/members', data)
    }

    updateMember(username: string, data: SystemMemberUpsertRequest) {
        return http.put<void>(`/api/system/member/${encodeURIComponent(username)}`, data)
    }

    deleteMember(username: string) {
        return http.delete<void>(`/api/system/member/${encodeURIComponent(username)}`)
    }

    // ==================== Compose 部署 ====================

    composeDeployDocker(data: { content: string; projectName: string; initURL?: string; initFile?: File }) {
        const form = new FormData()
        form.append('content', data.content)
        form.append('projectName', data.projectName)
        // 文件优先，二者互斥
        if (data.initFile) {
            form.append('initFile', data.initFile)
        } else if (data.initURL) {
            form.append('initURL', data.initURL)
        }
        return http.post<ComposeDeployResult>('/api/compose/docker/deploy', form)
    }

    composeDeploySwarm(data: { content: string; projectName: string }) {
        return http.post<ComposeDeployResult>('/api/compose/swarm/deploy', data)
    }

    composeRedeployDocker(name: string, data: { content: string }) {
        return http.put<ComposeDeployResult>(`/api/compose/docker/${name}`, data)
    }

    composeRedeploySwarm(name: string, data: { content: string }) {
        return http.put<ComposeDeployResult>(`/api/compose/swarm/${name}`, data)
    }
}

// 导出单例实例
export default new ApiService()
