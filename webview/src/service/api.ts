import axios, { AxiosRequestConfig } from 'axios'
import type {
    ContainerCreateRequest, ContainerUpdateRequest,
    NetworkCreateRequest, VolumeCreateRequest,
    SwarmCreateServiceRequest,
    ApisixRoute, ApisixCreateConsumerRequest, ApisixUpdateConsumerRequest
} from './types'

// API 服务类，统一管理所有 API 请求
class ApiService {
    // 认证相关
    login(data: { username: string; password: string }) {
        return axios.post('/api/login', data)
    }

    logout() {
        return axios.post('/api/logout')
    }

    // 文件管理相关
    list(path: string) {
        return axios.post('/api/filer/list', { path })
    }

    delete(path: string) {
        return axios.post('/api/filer/delete', { path })
    }

    rename(path: string, target: string) {
        return axios.post('/api/filer/rename', { path, target })
    }

    mkdir(path: string) {
        return axios.post('/api/filer/mkdir', { path })
    }

    create(path: string, content = '') {
        return axios.post('/api/filer/create', { path, content })
    }

    // 文件编辑相关
    read(path: string) {
        return axios.post('/api/filer/read', { path })
    }

    modify(path: string, content: string) {
        return axios.post('/api/filer/modify', { path, content })
    }

    chmod(path: string, mode: string) {
        return axios.post('/api/filer/chmod', { path, mode })
    }

    // 压缩解压
    zip(path: string) {
        return axios.post('/api/filer/zip', { path })
    }

    unzip(path: string) {
        return axios.post('/api/filer/unzip', { path })
    }

    // 文件上传
    upload(formData: FormData, config: AxiosRequestConfig = {}) {
        return axios.post('/api/filer/upload', formData, {
            headers: {
                'Content-Type': 'multipart/form-data'
            },
            ...config
        })
    }

    // 文件下载
    download(path: string) {
        return axios.post('/api/filer/download', { path }, { responseType: 'blob' })
    }

    // ==================== 服务探测相关 ====================

    serviceProbe() {
        return axios.get('/api/system/probe')
    }

    // ==================== 系统信息相关 ====================

    systemStat() {
        return axios.get('/api/system/stat')
    }

    // ==================== Docker 管理相关 ====================

    // Docker 概览信息
    dockerInfo() {
        return axios.get('/api/docker/info')
    }

    // 容器管理
    listContainers(all = false) {
        return axios.get('/api/docker/containers', { params: { all } })
    }

    containerAction(id: string, action: string) {
        return axios.post('/api/docker/container/action', { id, action })
    }

    createContainer(data: ContainerCreateRequest) {
        return axios.post('/api/docker/container/create', data)
    }

    deployCompose(content: string) {
        return axios.post('/api/docker/container/deploy-compose', { content })
    }

    containerLogs(id: string, tail = '100') {
        return axios.post('/api/docker/container/logs', { id, tail })
    }

    containerStats(id: string) {
        return axios.get('/api/docker/container/stats', { params: { id } })
    }

    getContainerConfig(name: string) {
        return axios.get('/api/docker/container/config', { params: { name } })
    }

    updateContainerConfig(data: ContainerUpdateRequest) {
        return axios.post('/api/docker/container/update', data)
    }

    // 镜像管理
    listImages(all = false) {
        return axios.get('/api/docker/images', { params: { all } })
    }

    imageInspect(id: string) {
        return axios.get('/api/docker/image/inspect', { params: { id } })
    }

    imageAction(id: string, action: string) {
        return axios.post('/api/docker/image/action', { id, action })
    }

    pullImage(image: string, tag = '') {
        return axios.post('/api/docker/image/pull', { image, tag })
    }

    imageTag(id: string, repoTag: string) {
        return axios.post('/api/docker/image/tag', { id, repoTag })
    }

    imageSearch(term: string) {
        return axios.get('/api/docker/image/search', { params: { term } })
    }

    imageBuild(dockerfile: string, tag = '') {
        return axios.post('/api/docker/image/build', { dockerfile, tag })
    }

    // 网络管理
    listNetworks() {
        return axios.get('/api/docker/networks')
    }

    networkInspect(id: string) {
        return axios.get('/api/docker/network/inspect', { params: { id } })
    }

    networkAction(id: string, action: string) {
        return axios.post('/api/docker/network/action', { id, action })
    }

    createNetwork(data: NetworkCreateRequest) {
        return axios.post('/api/docker/network/create', data)
    }

    // 卷管理
    listVolumes() {
        return axios.get('/api/docker/volumes')
    }

    volumeInspect(name: string) {
        return axios.get('/api/docker/volume/inspect', { params: { name } })
    }

    volumeAction(name: string, action: string) {
        return axios.post('/api/docker/volume/action', { name, action })
    }

    createVolume(data: VolumeCreateRequest) {
        return axios.post('/api/docker/volume/create', data)
    }

    // 镜像仓库管理
    listRegistries() {
        return axios.get('/api/docker/registries')
    }

    pushImage(image: string, registryUrl: string, namespace: string) {
        return axios.post('/api/docker/registry/push', { image, registryUrl, namespace })
    }

    pullFromRegistry(image: string, registryUrl: string, namespace: string) {
        return axios.post('/api/docker/registry/pull', { image, registryUrl, namespace })
    }

    // ==================== Docker Swarm 管理相关 ====================

    swarmInfo() {
        return axios.get('/api/swarm/info')
    }

    swarmListNodes() {
        return axios.get('/api/swarm/nodes')
    }

    swarmInspectNode(id: string) {
        return axios.get('/api/swarm/node/inspect', { params: { id } })
    }

    swarmNodeAction(id: string, action: string) {
        return axios.post('/api/swarm/node/action', { id, action })
    }

    swarmListServices() {
        return axios.get('/api/swarm/services')
    }

    swarmInspectService(id: string) {
        return axios.get('/api/swarm/service/inspect', { params: { id } })
    }

    swarmServiceAction(id: string, action: string, replicas?: number) {
        return axios.post('/api/swarm/service/action', { id, action, replicas })
    }

    swarmListTasks(serviceID = '') {
        return axios.get('/api/swarm/tasks', { params: serviceID ? { serviceID } : {} })
    }

    swarmCreateService(data: SwarmCreateServiceRequest) {
        return axios.post('/api/swarm/service/create', data)
    }

    swarmDeployComposeService(content: string) {
        return axios.post('/api/swarm/service/deploy-compose', { content })
    }

    swarmRedeployService(id: string) {
        return axios.post('/api/swarm/service/redeploy', { id })
    }

    swarmServiceLogs(id: string, tail = '100') {
        return axios.get('/api/swarm/service/logs', { params: { id, tail } })
    }

    // ==================== Apisix 管理相关 ====================

    // 路由管理
    apisixListRoutes() {
        return axios.get('/api/apisix/routes')
    }

    apisixGetRoute(id: string) {
        return axios.get(`/api/apisix/routes/${id}`)
    }

    apisixCreateRoute(data: ApisixRoute) {
        return axios.post('/api/apisix/routes', data)
    }

    apisixUpdateRoute(id: string, data: ApisixRoute) {
        return axios.put(`/api/apisix/routes/${id}`, data)
    }

    apisixPatchRouteStatus(id: string, status: number) {
        return axios.patch(`/api/apisix/routes/${id}/status`, { status })
    }

    apisixDeleteRoute(id: string) {
        return axios.delete(`/api/apisix/routes/${id}`)
    }

    // Consumer 管理
    apisixListConsumers() {
        return axios.get('/api/apisix/consumers')
    }

    apisixCreateConsumer(data: ApisixCreateConsumerRequest) {
        return axios.post('/api/apisix/consumers', data)
    }

    apisixUpdateConsumer(username: string, data: ApisixUpdateConsumerRequest) {
        return axios.put(`/api/apisix/consumers/${username}`, data)
    }

    apisixDeleteConsumer(username: string) {
        return axios.delete(`/api/apisix/consumers/${username}`)
    }

    // 白名单管理
    apisixGetWhitelist() {
        return axios.get('/api/apisix/whitelist')
    }

    apisixRevokeWhitelist(routeId: string, consumerName: string) {
        return axios.put('/api/apisix/whitelist/revoke', { route_id: routeId, consumer_name: consumerName })
    }

    // 辅助资源
    apisixListPluginConfigs() {
        return axios.get('/api/apisix/plugin_configs')
    }

    apisixListUpstreams() {
        return axios.get('/api/apisix/upstreams')
    }

    apisixListPlugins() {
        return axios.get('/api/apisix/plugins')
    }
}

// 导出单例实例
export default new ApiService()
