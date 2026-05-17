// ─── 系统探测 ───

export interface SystemProbe {
    agent: boolean
    apisix: boolean
    caddy: boolean
    docker: boolean
    swarm: boolean
    compose: boolean
}

// ─── 系统统计 ───

export interface SystemVersionCheck {
    latest: string
    update: boolean
    release: string
}

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
    Name: string
    ReadBytes: number
    WriteBytes: number
}

export interface SystemGoRuntimeStat {
    version: string
    numCPU: number
    numGoroutine: number
    heapAlloc: number
    heapInuse: number
    sys: number
    stackInuse: number
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
    version: string
    versionCheck: SystemVersionCheck
}
