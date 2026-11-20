// 端口转发规则：中控监听 listen，经隧道连到节点侧 target
export interface NodePortMap {
    name?: string
    proto: 'tcp' | 'udp'
    listen: number // 中控监听端口
    target: string // 节点侧目标 host:port，host 缺省为 127.0.0.1
}

// 端口转发规则整体替换请求
export interface NodePortMapUpdate {
    portmaps: NodePortMap[]
}

// 端口转发列表项：规则及其所属节点（由节点列表展开得到，后端不返回）
export interface NodePortMapInfo extends NodePortMap {
    node: NodeInfo
}

// 节点信息（列表/详情）
export interface NodeInfo {
    id: string
    name: string
    status: 'pending' | 'active' | 'revoked' // 待接入 / 可连接 / 已吊销
    hostname: string
    os: string
    arch: string
    createdAt: string
    portmaps?: NodePortMap[]
    online: boolean
    latency: number // 隧道往返延迟（毫秒），离线为 0
    agentVersion?: string
    negotiated: boolean // 是否成功取得节点的版本与能力信息
    features?: string[] // 节点声明的可选能力，如 portmap；negotiated 为 false 时无从判断
}

// 中控全局概览：只读快照，不包含节点 token、端口映射等配置。
export interface NodeOverviewItem {
    id: string
    name: string
    hostname: string
    status: NodeInfo['status']
    os: string
    arch: string
    agentVersion?: string
    online: boolean
    latency: number
    system?: {
        collectedAt: number
        cpuPercent?: number
        memoryUsed: number
        memoryTotal: number
        diskUsed: number
        diskTotal: number
    }
    docker?: { containersRunning: number; containersStopped: number; containersPaused: number }
    swarm?: { services: number; nodes: number }
    monitorError?: boolean
}

export interface NodeOverviewInfo {
    collectedAt: number
    online: number
    nodes: NodeOverviewItem[]
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
