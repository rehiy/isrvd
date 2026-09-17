// ─── 本机进程 ───

export interface SystemProcessInfo {
    pid: number
    ppid: number
    name: string
    username: string
    status: string
    /** 累计 CPU 时间（毫秒） */
    cpuMillis: number
    /** CPU 占用（%，相邻采样区间均值，多核可超过 100）；首次采样或不可用时省略 */
    cpuPercent?: number
    /** 内存占用（%） */
    memoryPercent: number
    /** 常驻内存（字节） */
    memoryRss: number
    /** 磁盘读取速率（字节/秒）；首次采样或不可用时省略 */
    ioReadBps?: number
    /** 磁盘写入速率（字节/秒）；首次采样或不可用时省略 */
    ioWriteBps?: number
    /** 启动时间（Unix 毫秒） */
    createTime: number
    cmdline?: string
}

export interface SystemProcessList {
    processes: SystemProcessInfo[]
}
