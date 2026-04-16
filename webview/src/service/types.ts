// ─── 通用 API 响应 ───

export interface APIResponse<T = any> {
    success: boolean
    message?: string
    payload?: T
}

// ─── 认证相关 ───

export interface LoginRequest {
    username: string
    password: string
}

export interface LoginResponse {
    token: string
    username: string
}

// ─── 文件管理相关 ───

export interface FileInfo {
    name: string
    path: string
    size: number
    mode: string
    modTime: string
    isDir: boolean
}

export interface FileListResponse {
    path: string
    files: FileInfo[]
}

export interface FileReadResponse {
    path: string
    content: string
}

// ─── 系统信息相关 ───

export interface ServiceProbeResponse {
    docker: { available: boolean }
    swarm: { available: boolean }
    apisix: { available: boolean }
}

// ─── Docker 相关 ───

export interface DockerInfo {
    containersRunning: number
    containersStopped: number
    containersPaused: number
    imagesTotal: number
    volumesTotal: number
    networksTotal: number
    registryMirrors: string[]
    indexServerAddress: string
}

export interface ContainerInfo {
    id: string
    name: string
    image: string
    state: string
    status: string
    ports: string[]
    created: number
    isSwarm?: boolean
    labels?: Record<string, string>
}

export interface VolumeMapping {
    hostPath: string
    containerPath: string
    readOnly: boolean
}

export interface ContainerCreateRequest {
    image: string
    name?: string
    cmd?: string[]
    env?: string[]
    ports?: Record<string, string>
    volumes?: VolumeMapping[]
    network?: string
    restart?: string
    memory?: number
    cpus?: number
    workdir?: string
    user?: string
    hostname?: string
    privileged?: boolean
    capAdd?: string[]
    capDrop?: string[]
}

export interface ContainerUpdateRequest {
    name: string
    image: string
    cmd?: string[]
    env?: string[]
    ports?: Record<string, string>
    volumes?: VolumeMapping[]
    network?: string
    restart?: string
    memory?: number
    cpus?: number
    workdir?: string
    user?: string
    hostname?: string
    privileged?: boolean
    capAdd?: string[]
    capDrop?: string[]
}

export interface ContainerConfigResponse {
    image: string
    name: string
    cmd?: string[]
    env?: string[]
    ports?: Record<string, string>
    volumes?: VolumeMapping[]
    network?: string
    restart?: string
    memory?: number
    cpus?: number
    workdir?: string
    user?: string
    hostname?: string
    privileged?: boolean
    capAdd?: string[]
    capDrop?: string[]
}

// ─── Docker 容器统计 ───

export interface CPUThrottledData {
    periods: number
    throttledPeriods: number
    throttledTime: number
}

export interface NetDetail {
    rxBytes: number
    rxPackets: number
    rxErrors: number
    rxDropped: number
    txBytes: number
    txPackets: number
    txErrors: number
    txDropped: number
}

export interface BlockDetail {
    major: number
    minor: number
    read: number
    write: number
}

export interface ContainerProcessList {
    titles: string[]
    processes: string[][]
}

export interface ContainerStatsResponse {
    id: string
    name: string
    cpuPercent: number
    cpuCores: number
    cpuFreq: number
    memoryUsage: number
    memoryLimit: number
    memoryPercent: number
    networkRx: number
    networkTx: number
    blockRead: number
    blockWrite: number
    pids: number
    pidsLimit: number
    cpuThrottled: CPUThrottledData
    networkDetail: Record<string, NetDetail>
    blockDetail: BlockDetail[]
    processList: ContainerProcessList | null
}

// ─── Docker 镜像 ───

export interface ImageInfo {
    id: string
    shortId: string
    repoTags: string[]
    size: number
    created: number
}

export interface ImageLayerInfo {
    digest: string
    createdBy: string
    created: string
    size: number
    empty: boolean
}

export interface ImageInspectResponse {
    id: string
    shortId: string
    repoTags: string[]
    repoDigests: string[]
    size: number
    virtualSize: number
    created: string
    author: string
    architecture: string
    os: string
    cmd: string[]
    entrypoint: string[]
    env: string[]
    workingDir: string
    exposedPorts: string[]
    labels: Record<string, string>
    layers: number
    layerDetails: ImageLayerInfo[]
}

export interface ImageSearchResult {
    name: string
    description: string
    isOfficial: boolean
    isAutomated: boolean
    starCount: number
}

// ─── Docker 网络 ───

export interface NetworkInfo {
    id: string
    name: string
    driver: string
    subnet: string
    scope: string
}

export interface NetworkContainerInfo {
    id: string
    name: string
    ipv4: string
    ipv6: string
    macAddress: string
}

export interface NetworkInspectResponse {
    id: string
    name: string
    driver: string
    scope: string
    subnet: string
    gateway: string
    internal: boolean
    enableIPv6: boolean
    containers: NetworkContainerInfo[]
}

export interface NetworkCreateRequest {
    name: string
    driver?: string
    subnet?: string
}

// ─── Docker 卷 ───

export interface VolumeInfo {
    name: string
    driver: string
    mountpoint: string
    createdAt: string
    size: number
}

export interface VolumeUsedByContainer {
    id: string
    name: string
    mountPath: string
    readOnly: boolean
}

export interface VolumeInspectResponse {
    name: string
    driver: string
    mountpoint: string
    createdAt: string
    scope: string
    size: number
    refCount: number
    usedBy: VolumeUsedByContainer[]
}

export interface VolumeCreateRequest {
    name: string
    driver?: string
}

// ─── Docker 镜像仓库 ───

export interface RegistryInfo {
    name: string
    url: string
    username: string
    description: string
}

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

export interface SwarmNode {
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

export interface SwarmPortConfig {
    published: number
    target: number
    protocol: string
}

export interface SwarmMount {
    type: string
    source: string
    target: string
}

export interface SwarmService {
    id: string
    name: string
    image: string
    mode: string
    replicas: number | null
    runningTasks: number
    ports: any
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

export interface SwarmServiceInspect {
    id: string
    name: string
    image: string
    mode: string
    replicas: number | null
    runningTasks: number
    ports: SwarmServicePort[]
    env: string[]
    args: string[]
    networks: string[]
    mounts: SwarmServiceMount[]
    labels: Record<string, string>
    constraints: string[]
    createdAt: string
    updatedAt: string
}

export interface SwarmCreateServiceRequest {
    name: string
    image: string
    mode?: string
    replicas?: number
    env?: string[]
    args?: string[]
    networks?: string[]
    ports?: SwarmPortConfig[]
    mounts?: SwarmMount[]
}

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

// ─── Apisix 相关 ───

export interface ApisixRoute {
    id?: string
    name: string
    uri?: string
    uris?: string[]
    host?: string
    hosts?: string[]
    desc?: string
    status: number
    priority: number
    enable_websocket: boolean
    plugin_config_id?: string
    upstream_id?: string
    upstream?: Record<string, any>
    plugins?: Record<string, any>
    consumers?: string[]
    create_time: number
    update_time: number
}

export interface ApisixConsumer {
    username: string
    desc: string
    plugins?: Record<string, any>
    create_time: number
    update_time: number
}

export interface ApisixCreateConsumerRequest {
    username: string
    desc?: string
}

export interface ApisixUpdateConsumerRequest {
    desc?: string
}

export interface ApisixPluginConfig {
    id: string
    desc: string
    plugins?: Record<string, any>
    create_time: number
    update_time: number
}

export interface ApisixUpstream {
    id: string
    name: string
    desc: string
    type: string
    create_time: number
    update_time: number
}

export interface ApisixRevokeWhitelistRequest {
    route_id: string
    consumer_name: string
}
