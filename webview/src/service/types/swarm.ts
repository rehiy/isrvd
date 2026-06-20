// ─── Docker Swarm ───

export interface SwarmInfo {
    clusterID: string
    createdAt: string
    nodes: number
    managers: number
    workers: number
    services: number
    tasks: number
}

// 节点列表项
export interface SwarmNodeInfo {
    id: string
    hostname: string
    role: string
    availability: string
    state: string
    addr: string
    engineVersion: string
    leader: boolean
}

// 节点详情
export interface SwarmNodeDetail {
    id: string
    hostname: string
    role: string
    availability: string
    state: string
    addr: string
    engineVersion: string
    leader: boolean
    os: string
    architecture: string
    cpus: number
    memoryBytes: number
    labels: Record<string, string>
    createdAt: string
    updatedAt: string
}

export interface SwarmServicePort {
    protocol: string
    targetPort: number
    publishedPort: number
    publishMode: string
}

export interface SwarmServiceMount {
    type: string
    source: string
    target: string
    readOnly: boolean
}

// SwarmServiceSpec 服务可写配置（创建/更新共用）
export interface SwarmServiceSpec {
    name: string
    image: string
    mode?: string
    replicas?: number | null
    env?: string[]
    args?: string[]
    networks?: string[]
    ports?: SwarmServicePort[]
    mounts?: SwarmServiceMount[]
    labels?: Record<string, string>
    constraints?: string[]
}

// SwarmServiceDetail 服务详情（SwarmServiceSpec + 运行时信息）
export interface SwarmServiceDetail extends SwarmServiceSpec {
    id: string
    runningTasks: number
    createdAt: string
    updatedAt: string
}

// SwarmServiceInfo 服务列表信息
export interface SwarmServiceInfo {
    id: string
    name: string
    image: string
    mode: string
    replicas: number | null
    runningTasks: number
    ports: SwarmServicePort[]
    createdAt: string
    updatedAt: string
}

// SwarmCreateService 直接复用 SwarmServiceSpec
export type SwarmCreateService = SwarmServiceSpec

// Swarm 服务 Compose
export interface SwarmServiceCompose {
    content: string
    projectName?: string
    fileModTime?: number
    source?: 'file' | 'runtime'
}

export interface SwarmTask {
    id: string
    serviceID: string
    serviceName: string
    nodeID: string
    nodeName: string
    slot: number
    image: string
    state: string
    message: string
    err: string
    updatedAt: string
}
