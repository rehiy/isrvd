// 节点信息（列表/详情）
export interface NodeInfo {
    id: string
    name: string
    status: 'pending' | 'approved' | 'revoked' // 待审批 / 已审批 / 已吊销
    hostname: string
    os: string
    arch: string
    createdAt: string
    online: boolean
    latency: number // 隧道往返延迟（毫秒），离线为 0
    agentVersion?: string
    compatible: boolean // 协议版本是否与中控一致
    connectedAt?: string
    remoteAddr?: string
}

// 节点更新请求
export interface NodeUpdate {
    name: string
}

// 注册码信息（创建结果同此类型；待接入期间可随时查看接入命令）
export interface NodeCodeInfo {
    id: string
    name: string
    code: string
    autoApprove: boolean
    createdBy: string
    createdAt: string
    expiresAt: string
}

// 注册码创建请求
export interface NodeCodeCreate {
    name: string
    ttlMinutes: number
    autoApprove: boolean
}
