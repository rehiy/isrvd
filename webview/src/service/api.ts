import type { AxiosRequestConfig } from 'axios'
import { http, httpBlob } from './axios'
import type {
    ContainerCreateRequest, ContainerUpdateRequest, ContainerConfigResponse,
    ContainerInfo, ContainerStatsResponse,
    ImageInfo, ImageInspectResponse, ImageSearchResult,
    NetworkInfo, NetworkInspectResponse, NetworkCreateRequest,
    VolumeInfo, VolumeInspectResponse, VolumeCreateRequest,
    RegistryInfo, RegistryUpsertRequest,
    SwarmInfo, SwarmNode, SwarmNodeInspect,
    SwarmService, SwarmServiceInspect, SwarmTask,
    SwarmCreateServiceRequest,
    ApisixRoute, ApisixConsumer, ApisixCreateConsumerRequest, ApisixUpdateConsumerRequest,
    ApisixPluginConfig, ApisixUpstream,
    ServiceProbeResponse, DockerInfo,
    FileListResponse, FileReadResponse,
    LoginResponse,
    AllSettings, ServerSettings, ApisixSettings, DockerSettings,
    MemberInfo, MemberUpsertRequest
} from './types'

// API 服务类，统一管理所有 API 请求
class ApiService {
    // 认证相关
    login(data: { username: string; password: string }) {
        return http.post<LoginResponse>('/api/login', data)
    }

    logout() {
        return http.post<void>('/api/logout')
    }

    // 文件管理相关
    list(path: string) {
        return http.post<FileListResponse>('/api/filer/list', { path })
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
        return http.post<FileReadResponse>('/api/filer/read', { path })
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
        return http.get<ServiceProbeResponse>('/api/system/probe')
    }

    // ==================== 系统信息相关 ====================

    systemStat() {
        return http.get<Record<string, unknown>>('/api/system/stat')
    }

    // ==================== Docker 管理相关 ====================

    // Docker 概览信息
    dockerInfo() {
        return http.get<DockerInfo>('/api/docker/info')
    }

    // 容器管理
    listContainers(all = false) {
        return http.get<ContainerInfo[]>('/api/docker/containers', { params: { all } })
    }

    containerAction(id: string, action: string) {
        return http.post<void>('/api/docker/container/action', { id, action })
    }

    createContainer(data: ContainerCreateRequest) {
        return http.post('/api/docker/container/create', data)
    }

    deployCompose(content: string) {
        return http.post<string[]>('/api/docker/container/deploy-compose', { content })
    }

    containerLogs(id: string, tail = '100') {
        return http.post<{ logs: string[] }>('/api/docker/container/logs', { id, tail })
    }

    containerStats(id: string) {
        return http.get<ContainerStatsResponse>('/api/docker/container/stats', { params: { id } })
    }

    getContainerConfig(name: string) {
        return http.get<ContainerConfigResponse>('/api/docker/container/config', { params: { name } })
    }

    updateContainerConfig(data: ContainerUpdateRequest) {
        return http.post('/api/docker/container/update', data)
    }

    // 镜像管理
    listImages(all = false) {
        return http.get<ImageInfo[]>('/api/docker/images', { params: { all } })
    }

    imageInspect(id: string) {
        return http.get<ImageInspectResponse>('/api/docker/image/inspect', { params: { id } })
    }

    imageAction(id: string, action: string) {
        return http.post<void>('/api/docker/image/action', { id, action })
    }

    pullImage(image: string, tag = '') {
        return http.post<void>('/api/docker/image/pull', { image, tag })
    }

    imageTag(id: string, repoTag: string) {
        return http.post<void>('/api/docker/image/tag', { id, repoTag })
    }

    imageSearch(term: string) {
        return http.get<ImageSearchResult[]>('/api/docker/image/search', { params: { term } })
    }

    imageBuild(dockerfile: string, tag = '') {
        return http.post<void>('/api/docker/image/build', { dockerfile, tag })
    }

    // 网络管理
    listNetworks() {
        return http.get<NetworkInfo[]>('/api/docker/networks')
    }

    networkInspect(id: string) {
        return http.get<NetworkInspectResponse>('/api/docker/network/inspect', { params: { id } })
    }

    networkAction(id: string, action: string) {
        return http.post<void>('/api/docker/network/action', { id, action })
    }

    createNetwork(data: NetworkCreateRequest) {
        return http.post('/api/docker/network/create', data)
    }

    // 卷管理
    listVolumes() {
        return http.get<VolumeInfo[]>('/api/docker/volumes')
    }

    volumeInspect(name: string) {
        return http.get<VolumeInspectResponse>('/api/docker/volume/inspect', { params: { name } })
    }

    volumeAction(name: string, action: string) {
        return http.post<void>('/api/docker/volume/action', { name, action })
    }

    createVolume(data: VolumeCreateRequest) {
        return http.post('/api/docker/volume/create', data)
    }

    // 镜像仓库管理
    listRegistries() {
        return http.get<RegistryInfo[]>('/api/docker/registries')
    }

    createRegistry(data: RegistryUpsertRequest) {
        return http.post<void>('/api/docker/registries', data)
    }

    updateRegistry(url: string, data: RegistryUpsertRequest) {
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
        return http.get<SwarmNode[]>('/api/swarm/nodes')
    }

    swarmInspectNode(id: string) {
        return http.get<SwarmNodeInspect>('/api/swarm/node/inspect', { params: { id } })
    }

    swarmNodeAction(id: string, action: string) {
        return http.post<void>('/api/swarm/node/action', { id, action })
    }

    swarmListServices() {
        return http.get<SwarmService[]>('/api/swarm/services')
    }

    swarmInspectService(id: string) {
        return http.get<SwarmServiceInspect>('/api/swarm/service/inspect', { params: { id } })
    }

    swarmServiceAction(id: string, action: string, replicas?: number) {
        return http.post<void>('/api/swarm/service/action', { id, action, replicas })
    }

    swarmListTasks(serviceID = '') {
        return http.get<SwarmTask[]>('/api/swarm/tasks', { params: serviceID ? { serviceID } : {} })
    }

    swarmCreateService(data: SwarmCreateServiceRequest) {
        return http.post('/api/swarm/service/create', data)
    }

    swarmDeployComposeService(content: string) {
        return http.post<string[]>('/api/swarm/service/deploy-compose', { content })
    }

    swarmRedeployService(id: string) {
        return http.post<void>('/api/swarm/service/redeploy', { id })
    }

    swarmServiceLogs(id: string, tail = '100') {
        return http.get<{ logs: string[] }>('/api/swarm/service/logs', { params: { id, tail } })
    }

    // ==================== APISIX 管理相关 ====================

    // 路由管理
    apisixListRoutes() {
        return http.get<ApisixRoute[]>('/api/apisix/routes')
    }

    apisixGetRoute(id: string) {
        return http.get<ApisixRoute>(`/api/apisix/routes/${id}`)
    }

    apisixCreateRoute(data: ApisixRoute) {
        return http.post('/api/apisix/routes', data)
    }

    apisixUpdateRoute(id: string, data: ApisixRoute) {
        return http.put(`/api/apisix/routes/${id}`, data)
    }

    apisixPatchRouteStatus(id: string, status: number) {
        return http.patch<void>(`/api/apisix/routes/${id}/status`, { status })
    }

    apisixDeleteRoute(id: string) {
        return http.delete<void>(`/api/apisix/routes/${id}`)
    }

    // Consumer 管理
    apisixListConsumers() {
        return http.get<ApisixConsumer[]>('/api/apisix/consumers')
    }

    apisixCreateConsumer(data: ApisixCreateConsumerRequest) {
        return http.post('/api/apisix/consumers', data)
    }

    apisixUpdateConsumer(username: string, data: ApisixUpdateConsumerRequest) {
        return http.put(`/api/apisix/consumers/${username}`, data)
    }

    apisixDeleteConsumer(username: string) {
        return http.delete<void>(`/api/apisix/consumers/${username}`)
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

    getSettings() {
        return http.get<AllSettings>('/api/settings')
    }

    updateServerSettings(data: ServerSettings) {
        return http.put<void>('/api/settings/server', data)
    }

    updateApisixSettings(data: ApisixSettings) {
        return http.put<void>('/api/settings/apisix', data)
    }

    updateDockerSettings(data: DockerSettings) {
        return http.put<void>('/api/settings/docker', data)
    }

    // 成员管理
    listMembers() {
        return http.get<MemberInfo[]>('/api/settings/members')
    }

    createMember(data: MemberUpsertRequest) {
        return http.post<void>('/api/settings/members', data)
    }

    updateMember(username: string, data: MemberUpsertRequest) {
        return http.put<void>(`/api/settings/members/${encodeURIComponent(username)}`, data)
    }

    deleteMember(username: string) {
        return http.delete<void>(`/api/settings/members/${encodeURIComponent(username)}`)
    }
}

// 导出单例实例
export default new ApiService()
