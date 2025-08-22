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
    return axios.get('/api/files', {
      params: { path }
    })
  }

  deleteFile(filePath) {
    return axios.delete('/api/delete', {
      params: { file: filePath }
    })
  }

  renameFile(oldPath, newName) {
    return axios.put('/api/rename', {
      oldPath: oldPath,
      newName: newName
    })
  }

  createDirectory(path, name) {
    return axios.post('/api/mkdir', {
      path: path,
      name: name
    })
  }

  createFile(path, name, content = '') {
    return axios.post('/api/newfile', {
      path: path,
      name: name,
      content: content
    })
  }

  // 文件编辑相关
  getFileContent(filePath) {
    return axios.get('/api/edit', {
      params: { file: filePath }
    })
  }

  saveFileContent(filePath, content) {
    return axios.put('/api/edit', {
      content: content
    }, {
      params: { file: filePath }
    })
  }

  // 权限管理
  getFilePermissions(filePath) {
    return axios.get('/api/chmod', {
      params: { file: filePath }
    })
  }

  setFilePermissions(filePath, mode) {
    return axios.put('/api/chmod', {
      mode: mode
    }, {
      params: { file: filePath }
    })
  }

  // 压缩解压
  zipFiles(filePath, zipName) {
    return axios.post('/api/zip', {
      path: filePath,
      zipName: zipName
    })
  }

  unzipFile(filePath, targetPath) {
    return axios.post('/api/unzip', {
      path: targetPath,
      zipName: filePath
    })
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
