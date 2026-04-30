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

export interface SwarmNodeDTO {
    id: string
    hostname: string
    role: string
    availability: string
    state: string
    addr: string
    engineVersion: string
    leader: boolean
}

export interface SwarmNodeInspect {
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

// SwarmServiceInfo 服务列表 DTO
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

// SwarmCreateServiceRequest 直接复用 SwarmServiceSpec
export type SwarmCreateServiceRequest = SwarmServiceSpec

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
