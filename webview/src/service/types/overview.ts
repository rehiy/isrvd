import type { DockerContainerStats } from './docker'
import type { AuthInfo } from './account'
import type { LinkConfig } from './system'

// ─── 启动聚合 ───

export interface BootstrapConfig {
    maxUploadSize: number
    marketplaceUrl: string
    links: LinkConfig[]
}

export interface BootstrapData {
    auth: AuthInfo
    probe?: SystemProbe
    config?: BootstrapConfig
}

// ─── 系统探测 ───

export interface SystemVersionInfo {
    current: string
    latest: string
    hasUpdate: boolean
    release: string
    updaterImage: string
}

export interface SystemProbe {
    agent: boolean
    apisix: boolean
    caddy: boolean
    docker: boolean
    swarm: boolean
    compose: boolean
}

// ─── 系统统计 ───

export interface SystemNetInterface {
    name: string
    bytesRecv: number
    bytesSent: number
}

export interface SystemDiskPartition {
    device: string
    mountpoint: string
    fstype: string
    used: number
    total: number
}

export interface SystemDiskIO {
    name: string
    readBytes: number
    writeBytes: number
    readCount: number
    writeCount: number
}

export interface SystemGoRuntimeStat {
    version: string
    numCPU: number
    numGoroutine: number
    alloc: number
    sys: number
    heapAlloc: number
    heapInuse: number
    heapIdle: number
    heapReleased: number
    heapObjects: number
    heapSys: number
    stackInuse: number
    stackSys: number
    totalAlloc: number
    numGC: number
    lastGC: number
}

export interface SystemInfo {
    hostName: string
    platform: string
    kernelArch: string
    uptime: number
    cpuCore: number
    cpuCoreLogic: number
    cpuModel: string[]
    cpuPercent: number[]
    memoryUsed: number
    memoryTotal: number
    diskTotal: number
    diskUsed: number
    netInterface: SystemNetInterface[]
    diskPartition: SystemDiskPartition[]
}

export interface SystemGPU {
    index: number
    deviceKey?: string
    name: string
    vendor: string         // "nvidia" | "amd" | "intel"
    memoryUsed: number
    memoryTotal: number
    utilization: number    // 0-100
    temperature: number    // -1 = N/A
    powerUsage: number     // watts, -1 = N/A
    fanSpeed: number       // percent, -1 = N/A
}

export interface SystemStat {
    system: SystemInfo
    diskIO: SystemDiskIO[]
    gpu: SystemGPU[]
    go: SystemGoRuntimeStat
}

// ─── 监控历史 ───

export interface MonitorHostRecord {
    ts: number
    data: SystemStat
}

export interface MonitorContainerRecord {
    ts: number
    data: DockerContainerStats
}
