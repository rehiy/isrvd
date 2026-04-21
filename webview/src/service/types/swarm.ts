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

export interface NodeDTO {
    id: string
    hostname: string
    role: string
    availability: string
    state: string
    addr: string
    engineVersion: string
    leader: boolean
}

export interface NodeInspect {
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

export interface ServicePort {
    protocol: string
    targetPort: number
    publishedPort: number
    publishMode: string
}

export interface ServiceMount {
    type: string
    source: string
    target: string
    readOnly: boolean
}

// ServiceSpec 服务可写配置（创建/更新共用）
export interface ServiceSpec {
    name: string
    image: string
    mode?: string
    replicas?: number | null
    env?: string[]
    args?: string[]
    networks?: string[]
    ports?: ServicePort[]
    mounts?: ServiceMount[]
    labels?: Record<string, string>
    constraints?: string[]
}

// ServiceInfo 服务详情（ServiceSpec + 运行时信息）
export interface ServiceInfo extends ServiceSpec {
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
    ports: ServicePort[]
    createdAt: string
    updatedAt: string
}

// CreateServiceRequest 直接复用 ServiceSpec
export type CreateServiceRequest = ServiceSpec

export interface SwarmTask {
    id: string
    serviceID: string
    serviceName: string
    nodeID: string
    slot: number
    image: string
    state: string
    message: string
    err: string
    updatedAt: string
}
