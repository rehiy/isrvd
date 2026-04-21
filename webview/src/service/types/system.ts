// ─── 系统探测 ───

export interface ServiceProbeResponse {
    docker: { available: boolean }
    swarm: { available: boolean }
    apisix: { available: boolean }
}

// ─── 系统设置 ───

export interface ServerSettings {
    debug: boolean
    listenAddr: string
    // 写入时为空表示保留原值
    jwtSecret: string
    // 读取时后端返回，标识是否已设置
    jwtSecretSet?: boolean
    proxyHeaderName: string
    rootDirectory: string
}

export interface ApisixSettings {
    adminUrl: string
    // 写入时为空表示保留原值
    adminKey: string
    // 读取时后端返回，标识是否已设置
    adminKeySet?: boolean
}

export interface AgentSettings {
    model: string
    baseUrl: string
    // 写入时为空表示保留原值
    apiKey: string
    // 读取时后端返回，标识是否已设置
    apiKeySet?: boolean
}

export interface DockerSettings {
    host: string
    containerRoot: string
}

export interface MarketplaceSettings {
    url: string
}

export interface AllSettings {
    server: ServerSettings
    agent: AgentSettings
    apisix: ApisixSettings
    docker: DockerSettings
    marketplace: MarketplaceSettings
}

// ─── 成员管理 ───

export interface MemberInfo {
    username: string
    homeDirectory: string
    allowTerminal: boolean
    passwordSet: boolean
    isPrimary: boolean
}

export interface MemberUpsertRequest {
    username: string
    // 写入时为空表示保留原值（仅更新场景）
    password: string
    homeDirectory: string
    allowTerminal: boolean
}

// ─── 系统统计（/api/system/stats 响应） ───

export interface NetInterface {
    Name: string
    BytesRecv: number
    BytesSent: number
}

export interface DiskPartition {
    Device: string
    Mountpoint: string
    Fstype: string
    Used: number
    Total: number
}

export interface DiskIO {
    Name: string
    ReadBytes: number
    WriteBytes: number
}

export interface GoRuntimeStat {
    version: string
    numCPU: number
    numGoroutine: number
    HeapAlloc: number
    HeapInuse: number
    Sys: number
    StackInuse: number
    TotalAlloc: number
    NumGC: number
    LastGC: number
}

export interface SystemInfo {
    HostName: string
    Platform: string
    KernelArch: string
    Uptime: number
    CpuCore: number
    CpuCoreLogic: number
    CpuModel: string[]
    CpuPercent: number[]
    MemoryUsed: number
    MemoryTotal: number
    DiskTotal: number
    DiskUsed: number
    NetInterface: NetInterface[]
    DiskPartition: DiskPartition[]
    [key: string]: unknown
}

export interface SystemStat {
    system: SystemInfo
    diskIO: DiskIO[]
    go: GoRuntimeStat
}
