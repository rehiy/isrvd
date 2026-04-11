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
    return axios.post('/api/list', { path })
  }

  delete(path) {
    return axios.post('/api/delete', { path })
  }

  rename(path, target) {
    return axios.post('/api/rename', { path, target })
  }

  mkdir(path) {
    return axios.post('/api/mkdir', { path })
  }

  create(path, content = '') {
    return axios.post('/api/create', { path, content })
  }

  // 文件编辑相关
  read(path) {
    return axios.post('/api/read', { path })
  }

  modify(path, content) {
    return axios.post('/api/modify', { path, content })
  }

  chmod(path, mode) {
    return axios.post('/api/chmod', { path, mode })
  }

  // 压缩解压
  zip(path) {
    return axios.post('/api/zip', { path })
  }

  unzip(path) {
    return axios.post('/api/unzip', { path })
  }

  // 文件上传
  upload(formData, config = {}) {
    return axios.post('/api/upload', formData, {
      headers: {
        'Content-Type': 'multipart/form-data'
      },
      ...config
    })
  }

  // 文件下载
  download(path) {
    return axios.post('/api/download', { path }, { responseType: 'blob' })
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
    return axios.post('/api/docker/containers/action', { id, action })
  }
  createContainer(data) {
    return axios.post('/api/docker/containers/create', data)
  }
  containerLogs(id, tail = '100') {
    return axios.post('/api/docker/containers/logs', { id, tail })
  }

  // 镜像管理
  listImages(all = false) {
    return axios.get('/api/docker/images', { params: { all } })
  }
  imageAction(id, action) {
    return axios.post('/api/docker/images/action', { id, action })
  }
  pullImage(image, tag = '') {
    return axios.post('/api/docker/images/pull', { image, tag })
  }

  // 网络管理
  listNetworks() {
    return axios.get('/api/docker/networks')
  }
  networkAction(id, action) {
    return axios.post('/api/docker/networks/action', { id, action })
  }
  createNetwork(data) {
    return axios.post('/api/docker/networks/create', data)
  }

  // 卷管理
  listVolumes() {
    return axios.get('/api/docker/volumes')
  }
  volumeAction(name, action) {
    return axios.post('/api/docker/volumes/action', { name, action })
  }
  createVolume(data) {
    return axios.post('/api/docker/volumes/create', data)
  }
}

// 导出单例实例
export default new ApiService()
