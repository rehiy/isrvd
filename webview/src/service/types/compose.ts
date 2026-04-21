// ─── Compose 统一部署 ───

export type ComposeDeployTarget = 'docker' | 'swarm'

// 统一的 compose 部署请求
//   - target=docker：落盘到 {ContainerRoot}/{projectName}
//     可选 initURL 指定附加运行文件 zip（应用市场一键安装）
//   - target=swarm ：不落盘，projectName 仅作 compose project 名
export interface ComposeDeployRequest {
    target: ComposeDeployTarget
    content: string                                    // 完整 compose yaml 文本（前端已完成 ${VAR} 插值）
    projectName: string                                // 必填：实例名（同时作 compose project 名）；docker 下落盘到 数据目录/实例名
    initURL?: string                                   // 可选：附加运行文件 zip 下载地址（仅 docker 生效）
}

export interface ComposeDeployResult {
    target: ComposeDeployTarget
    items: string[]
    installDir?: string                                // 仅 docker 落盘时返回
}

// ─── 应用市场协议 ───

// 回传给父组件的精简 payload
export interface MarketplacePick {
    name: string
    compose: string
    initURL?: string
}
