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
}

// 导出单例实例
export default new ApiService()
