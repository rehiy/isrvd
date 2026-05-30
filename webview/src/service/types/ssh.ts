// SSH 主机信息（列表/详情，密码不回显）
export interface SSHHostInfo {
    id: string
    name: string
    addr: string
    user: string
    passwordSet: boolean
    privateKey: string
    description: string
}

// SSH 主机创建/更新请求
export interface SSHHostUpsert {
    name: string
    addr: string
    user: string
    password: string
    privateKey: string
    description: string
}

// SFTP 文件/目录信息
export interface SFTPFileInfo {
    name: string
    size: number
    mode: string
    modTime: number
    isDir: boolean
    isLink: boolean
    linkTarget?: string
}

// SFTP 目录列表结果
export interface SFTPListResult {
    path: string
    files: SFTPFileInfo[]
}

// SFTP 重命名请求
export interface SFTPRename {
    oldPath: string
    newPath: string
}

// SFTP 创建目录请求
export interface SFTPMkdir {
    path: string
}

// SFTP 修改权限请求
export interface SFTPChmod {
    path: string
    mode: string
}
