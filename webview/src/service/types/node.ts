// 节点信息（列表/详情）
export interface NodeInfo {
    id: string
    name: string
    status: 'pending' | 'active' | 'revoked' // 待接入 / 可连接 / 已吊销
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

// 节点创建请求，备注必填。
export type NodeCreate = NodeUpdate

// token 仅在创建时返回，节点列表不返回。
export interface NodeCreateResult {
    node: NodeInfo
    token: string
}
