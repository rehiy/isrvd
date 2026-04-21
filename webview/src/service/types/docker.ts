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

export interface DockerContainerInfo {
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

export interface DockerVolumeMapping {
    hostPath: string
    containerPath: string
    readOnly: boolean
}

export interface DockerContainerCreateRequest {
    image: string
    name?: string
    cmd?: string[]
    env?: string[]
    ports?: Record<string, string>
    volumes?: DockerVolumeMapping[]
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

export interface DockerCPUThrottledData {
    periods: number
    throttledPeriods: number
    throttledTime: number
}

export interface DockerNetDetail {
    rxBytes: number
    rxPackets: number
    rxErrors: number
    rxDropped: number
    txBytes: number
    txPackets: number
    txErrors: number
    txDropped: number
}

export interface DockerBlockDetail {
    major: number
    minor: number
    read: number
    write: number
}

export interface DockerContainerProcessList {
    titles: string[]
    processes: string[][]
}

export interface DockerContainerStatsResponse {
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
    cpuThrottled: DockerCPUThrottledData
    networkDetail: Record<string, DockerNetDetail>
    blockDetail: DockerBlockDetail[]
    processList: DockerContainerProcessList | null
}

// ─── 镜像 ───

export interface DockerImageInfo {
    id: string
    shortId: string
    repoTags: string[]
    size: number
    created: number
}

export interface DockerImageLayerInfo {
    digest: string
    createdBy: string
    created: string
    size: number
    empty: boolean
}

export interface DockerImageInspectResponse {
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
    layerDetails: DockerImageLayerInfo[]
}

export interface DockerImageSearchResult {
    name: string
    description: string
    isOfficial: boolean
    isAutomated: boolean
    starCount: number
}

// ─── 网络 ───

export interface DockerNetworkInfo {
    id: string
    name: string
    driver: string
    subnet: string
    scope: string
}

export interface DockerNetworkContainerInfo {
    id: string
    name: string
    ipv4: string
    ipv6: string
    macAddress: string
}

export interface DockerNetworkInspectResponse {
    id: string
    name: string
    driver: string
    scope: string
    subnet: string
    gateway: string
    internal: boolean
    enableIPv6: boolean
    containers: DockerNetworkContainerInfo[]
}

export interface DockerNetworkCreateRequest {
    name: string
    driver?: string
    subnet?: string
}

// ─── 卷 ───

export interface DockerVolumeInfo {
    name: string
    driver: string
    mountpoint: string
    createdAt: string
    size: number
}

export interface DockerVolumeUsedByContainer {
    id: string
    name: string
    mountPath: string
    readOnly: boolean
}

export interface DockerVolumeInspectResponse {
    name: string
    driver: string
    mountpoint: string
    createdAt: string
    scope: string
    size: number
    refCount: number
    usedBy: DockerVolumeUsedByContainer[]
}

export interface DockerVolumeCreateRequest {
    name: string
    driver?: string
}

// ─── 镜像仓库 ───

export interface DockerRegistryInfo {
    name: string
    url: string
    username: string
    description: string
}

export interface DockerRegistryUpsertRequest {
    name: string
    url: string
    username?: string
    password?: string
    description?: string
}
