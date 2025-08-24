import axios from 'axios'

// API服务类，统一管理所有API请求
class ApiService {
  // 认证相关
  login(loginData) {
    return axios.post('/api/login', loginData)
  }

  logout() {
    return axios.post('/api/logout')
  }

  // 文件管理相关
  getFiles(path) {
    return axios.post('/api/list', { path })
  }

  deleteFile(path) {
    return axios.post('/api/delete', { path })
  }

  renameFile(path, newPath) {
    return axios.post('/api/rename', { path, newPath })
  }

  createDirectory(path, name) {
    return axios.post('/api/mkdir', { path, name })
  }

  createFile(path, name, content = '') {
    return axios.post('/api/create', { path, name, content })
  }

  // 文件编辑相关
  getFileContent(path) {
    return axios.post('/api/read', { path })
  }

  saveFileContent(path, content) {
    return axios.post('/api/write', { path, content })
  }

  // 权限管理
  getFilePermissions(path) {
    return axios.post('/api/chmod', { path })
  }

  setFilePermissions(path, mode) {
    return axios.post('/api/chmod', { path, mode })
  }

  // 压缩解压
  zipFiles(path, zipName) {
    return axios.post('/api/zip', { path, zipName })
  }

  unzipFile(path, zipName) {
    return axios.post('/api/unzip', { path, zipName })
  }

  // Zip 信息和检查
  getZipInfo(path) {
    return axios.post('/api/zip/info', { path })
  }

  isZipFile(path) {
    return axios.post('/api/zip/check', { path })
  }

  // 文件下载
  downloadFile(path) {
    return axios.post('/api/download', { path }, { responseType: 'blob' })
  }

  // 文件上传
  uploadFile(formData, config = {}) {
    return axios.post('/api/upload', formData, {
      headers: {
        'Content-Type': 'multipart/form-data'
      },
      ...config
    })
  }
}

// 导出单例实例
export default new ApiService()
