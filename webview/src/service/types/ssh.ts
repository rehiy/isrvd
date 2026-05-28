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
