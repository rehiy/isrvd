// ─── 本机进程 ───

export interface SystemProcessInfo {
    pid: number
    ppid: number
    name: string
    username: string
    status: string
    /** CPU 占用（%，自启动以来的均值，多核可超过 100） */
    cpuPercent: number
    /** 内存占用（%） */
    memoryPercent: number
    /** 常驻内存（字节） */
    memoryRss: number
    /** 启动时间（Unix 毫秒） */
    createTime: number
    cmdline: string
}

export interface SystemProcessList {
    processes: SystemProcessInfo[]
}
