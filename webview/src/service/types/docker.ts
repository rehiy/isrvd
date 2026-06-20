// ─── Compose 标签常量 ───

export const COMPOSE_PROJECT_LABEL = 'com.docker.compose.project'
export const COMPOSE_SERVICE_LABEL = 'com.docker.compose.service'

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

// ─── 容器文件管理 ───

export interface ContainerFileInfo {
    name: string
    size: number
    mode: string    // 如 "-rw-r--r--" 或 "drwxr-xr-x"
    modTime: number // Unix 时间戳
    isDir: boolean
    isLink: boolean
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
    isSelf?: boolean
    labels?: Record<string, string>
}

export interface DockerVolumeMapping {
    type?: 'bind' | 'volume'
    source?: string
    hostPath?: string
    containerPath: string
    readOnly: boolean
}

export interface DockerContainerCreate {
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
    autoRemove?: boolean
    labels?: Record<string, string>
}

export interface DockerContainerCreateResult {
    id: string
    name: string
}

export interface DockerContainerDetail extends DockerContainerCreate {
    id: string
    name: string
    state: string
    createdAt: string
    labels?: Record<string, string>
}

// 容器 Compose
export interface DockerContainerCompose {
    content: string
    projectName?: string
    fileModTime?: number
    source?: 'file' | 'runtime'
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

export interface DockerContainerStats {
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
    repoDigests: string[]
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

export interface DockerImageDetail {
    id: string
    shortId: string
    repoTags: string[]
    repoDigests: string[]
    size: number
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
    starCount: number
}

export interface DockerImagePrune {
    all?: boolean
    until?: string
}

export interface DockerImagePruneDeleted {
    untagged?: string
    deleted?: string
}

export interface DockerImagePruneResult {
    imagesDeleted: DockerImagePruneDeleted[]
    spaceReclaimed: number
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

export interface DockerNetworkDetail {
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

export interface DockerNetworkCreate {
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

export interface DockerVolumeDetail {
    name: string
    driver: string
    mountpoint: string
    createdAt: string
    scope: string
    size: number
    refCount: number
    usedBy: DockerVolumeUsedByContainer[]
}

export interface DockerVolumeCreate {
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

export interface DockerRegistryUpsert {
    name: string
    url: string
    username?: string
    password?: string
    description?: string
}
