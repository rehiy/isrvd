import axios from 'axios'

// API服务类，统一管理所有API请求
class ApiService {
  // 认证相关
  login(data) {
    return axios.post('/api/login', data)
  }

  logout() {
    return axios.post('/api/logout')
  }

  // 文件管理相关
  list(path) {
    return axios.post('/api/filer/list', { path })
  }

  delete(path) {
    return axios.post('/api/filer/delete', { path })
  }

  rename(path, target) {
    return axios.post('/api/filer/rename', { path, target })
  }

  mkdir(path) {
    return axios.post('/api/filer/mkdir', { path })
  }

  create(path, content = '') {
    return axios.post('/api/filer/create', { path, content })
  }

  // 文件编辑相关
  read(path) {
    return axios.post('/api/filer/read', { path })
  }

  modify(path, content) {
    return axios.post('/api/filer/modify', { path, content })
  }

  chmod(path, mode) {
    return axios.post('/api/filer/chmod', { path, mode })
  }

  // 压缩解压
  zip(path) {
    return axios.post('/api/filer/zip', { path })
  }

  unzip(path) {
    return axios.post('/api/filer/unzip', { path })
  }

  // 文件上传
  upload(formData, config = {}) {
    return axios.post('/api/filer/upload', formData, {
      headers: {
        'Content-Type': 'multipart/form-data'
      },
      ...config
    })
  }

  // 文件下载
  download(path) {
    return axios.post('/api/filer/download', { path }, { responseType: 'blob' })
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
  containerAction(id, action) {
    return axios.post('/api/docker/container/action', { id, action })
  }
  createContainer(data) {
    return axios.post('/api/docker/container/create', data)
  }
  containerLogs(id, tail = '100') {
    return axios.post('/api/docker/container/logs', { id, tail })
  }
  containerStats(id) {
    return axios.get('/api/docker/container/stats', { params: { id } })
  }
  getContainerConfig(name) {
    return axios.get('/api/docker/container/config', { params: { name } })
  }
  updateContainerConfig(data) {
    return axios.post('/api/docker/container/update', data)
  }

  // 镜像管理
  listImages(all = false) {
    return axios.get('/api/docker/images', { params: { all } })
  }
  imageInspect(id) {
    return axios.get('/api/docker/image/inspect', { params: { id } })
  }
  imageAction(id, action) {
    return axios.post('/api/docker/image/action', { id, action })
  }
  pullImage(image, tag = '') {
    return axios.post('/api/docker/image/pull', { image, tag })
  }
  imageTag(id, repoTag) {
    return axios.post('/api/docker/image/tag', { id, repoTag })
  }
  imageSearch(term) {
    return axios.get('/api/docker/image/search', { params: { term } })
  }
  imageBuild(dockerfile, tag = '') {
    return axios.post('/api/docker/image/build', { dockerfile, tag })
  }

  // 网络管理
  listNetworks() {
    return axios.get('/api/docker/networks')
  }
  networkInspect(id) {
    return axios.get('/api/docker/network/inspect', { params: { id } })
  }
  networkAction(id, action) {
    return axios.post('/api/docker/network/action', { id, action })
  }
  createNetwork(data) {
    return axios.post('/api/docker/network/create', data)
  }

  // 卷管理
  listVolumes() {
    return axios.get('/api/docker/volumes')
  }
  volumeInspect(name) {
    return axios.get('/api/docker/volume/inspect', { params: { name } })
  }
  volumeAction(name, action) {
    return axios.post('/api/docker/volume/action', { name, action })
  }
  createVolume(data) {
    return axios.post('/api/docker/volume/create', data)
  }

  // 镜像仓库管理
  listRegistries() {
    return axios.get('/api/docker/registries')
  }
  pushImage(image, registryUrl, namespace) {
    return axios.post('/api/docker/registry/push', { image, registryUrl, namespace })
  }
  pullFromRegistry(image, registryUrl, namespace) {
    return axios.post('/api/docker/registry/pull', { image, registryUrl, namespace })
  }

  // ==================== Docker Swarm 管理相关 ====================

  swarmInfo() {
    return axios.get('/api/swarm/info')
  }
  swarmListNodes() {
    return axios.get('/api/swarm/nodes')
  }
  swarmInspectNode(id) {
    return axios.get('/api/swarm/node/inspect', { params: { id } })
  }
  swarmNodeAction(id, action) {
    return axios.post('/api/swarm/node/action', { id, action })
  }
  swarmListServices() {
    return axios.get('/api/swarm/services')
  }
  swarmInspectService(id) {
    return axios.get('/api/swarm/service/inspect', { params: { id } })
  }
  swarmServiceAction(id, action, replicas) {
    return axios.post('/api/swarm/service/action', { id, action, replicas })
  }
  swarmListTasks(serviceID = '') {
    return axios.get('/api/swarm/tasks', { params: serviceID ? { serviceID } : {} })
  }
  swarmCreateService(data) {
    return axios.post('/api/swarm/service/create', data)
  }
  swarmRedeployService(id) {
    return axios.post('/api/swarm/service/redeploy', { id })
  }
  swarmServiceLogs(id, tail = '100') {
    return axios.get('/api/swarm/service/logs', { params: { id, tail } })
  }


  // ==================== Apisix 管理相关 ====================

  // 路由管理
  apisixListRoutes() {
    return axios.get('/api/apisix/routes')
  }
  apisixGetRoute(id) {
    return axios.get(`/api/apisix/routes/${id}`)
  }
  apisixCreateRoute(data) {
    return axios.post('/api/apisix/routes', data)
  }
  apisixUpdateRoute(id, data) {
    return axios.put(`/api/apisix/routes/${id}`, data)
  }
  apisixPatchRouteStatus(id, status) {
    return axios.patch(`/api/apisix/routes/${id}/status`, { status })
  }
  apisixDeleteRoute(id) {
    return axios.delete(`/api/apisix/routes/${id}`)
  }

  // Consumer 管理
  apisixListConsumers() {
    return axios.get('/api/apisix/consumers')
  }
  apisixCreateConsumer(data) {
    return axios.post('/api/apisix/consumers', data)
  }
  apisixUpdateConsumer(username, data) {
    return axios.put(`/api/apisix/consumers/${username}`, data)
  }
  apisixDeleteConsumer(username) {
    return axios.delete(`/api/apisix/consumers/${username}`)
  }

  // 白名单管理
  apisixGetWhitelist() {
    return axios.get('/api/apisix/whitelist')
  }
  apisixRevokeWhitelist(routeId, consumerName) {
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
