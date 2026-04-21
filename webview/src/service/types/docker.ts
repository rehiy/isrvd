// ─── Docker 概览 ───

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

// ─── 容器 ───

export interface ContainerInfo {
    id: string
    name: string
    image: string
    state: string
    status: string
    ports: string[]
    networks?: string[]
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

// ─── 容器统计 ───

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

// ─── 镜像 ───

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

// ─── 网络 ───

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

// ─── 卷 ───

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

// ─── 镜像仓库 ───

export interface RegistryInfo {
    name: string
    url: string
    username: string
    description: string
}

export interface RegistryUpsertRequest {
    name: string
    url: string
    username?: string
    password?: string
    description?: string
}
