// SSH 凭据信息
export interface SSHCredentialInfo {
    id: string
    name: string
    description: string
    user: string
    authType?: string // "password" | "privateKey" | ""
    password?: string
    privateKey?: string
}

// SSH 凭据创建/更新请求
export interface SSHCredentialUpsert {
    name: string
    description: string
    user: string
    password: string
    privateKey: string
}

// SSH 主机信息（列表/详情）
export interface SSHHostInfo {
    id: string
    name: string
    addr: string
    credentialId?: string
    credentialName?: string
    user: string
    description: string
}

// SSH 主机创建/更新请求
// 注意：credentialId 和 (user + password/privateKey) 二选一
// user/password/privateKey 在 credentialId 为空时需要提供
export interface SSHHostUpsert {
    name: string
    addr: string
    credentialId?: string
    user?: string
    password?: string
    privateKey?: string
    description?: string
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

// SFTP 修改所有者请求
export interface SFTPChown {
    path: string
    uid: number
    gid: number
}

// SFTP 读取文件响应
export interface SFTPRead {
    content: string
}

// SFTP 写入文件请求
export interface SFTPWrite {
    path: string
    content: string
}
