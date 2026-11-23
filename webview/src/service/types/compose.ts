// ─── Compose 统一部署 ───

export type ComposeDeployTarget = 'docker' | 'swarm'

export interface ComposeHistoryInfo {
    id: string
    time: string
    action: 'snapshot' | 'deploy' | 'redeploy'
    success: boolean
    hasSnapshot: boolean
}

export interface ComposeConfigDetail {
    content: string
    envContent?: string
    projectName?: string
    fileModTime?: number
    source?: 'file' | 'runtime'
}

// 统一的 compose 部署请求
// Compose 文件必须包含非空顶层 name，作为项目名
export interface ComposeDeploy {
    content: string         // 完整 compose yaml 文本
    envContent?: string     // 省略表示保留附加文件解压出的 .env；提交内容则以其为准
    initURL?: string        // 可选：附加运行文件 zip 下载地址
    initFile?: File         // 可选：附加运行文件（与 initURL 互斥，文件优先）
}

// Compose Redeploy 请求（部分更新：提交哪个字段就改哪个字段，未提交的保持不变；content 提交空串会被后端拒绝）
export interface ComposeRedeploy {
    content?: string
    envContent?: string     // 省略表示保留现有 .env；空串表示清空；提交内容则覆盖
    serviceName?: string
    image?: string
}

export interface ComposeRedeployPreview {
    target: ComposeDeployTarget
    projectName: string
    changes: Array<{
        service: string
        action: 'create' | 'recreate' | 'update' | 'unchanged' | 'remove'
        configChanged: boolean
    }>
}

export interface ComposeDeployResult {
    target: ComposeDeployTarget
    projectName: string     // 实际使用的项目名
    items: string[]
    installDir?: string     // 仅 docker 落盘时返回
}

// ─── 应用市场协议 ───

// 回传给父组件的精简 payload
export interface ComposeMarketplacePick {
    name: string
    compose: string
    initURL?: string
}
