/**
 * 多节点（中控模式）的地址约定。
 *
 * 中控网关把 /n/<节点ID>/ 下的接口转发给所选节点，页面本身仍由中控提供。
 * 前端使用相对路径（baseURL 为 api/、hash 路由），因此切换节点只需要换页面所在的路径前缀。
 */

const NODE_PATH = /^(.*\/)n\/([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})\/(?:index\.html)?$/

/** 当前页面所在的中控根路径（以 / 结尾）与所选节点 ID（本机为空字符串） */
export const nodeLocation = (): { root: string; id: string } => {
    const match = window.location.pathname.match(NODE_PATH)
    if (match) return { root: match[1], id: match[2] }
    return { root: window.location.pathname.replace(/index\.html$/, ''), id: '' }
}

/** 中控对外地址（不含节点前缀与末尾斜杠），受管机用它接入 */
export const centerUrl = (): string =>
    window.location.origin + nodeLocation().root.replace(/\/+$/, '')

/** 仅在中控视角展示的管理页面；个人账户设置仍可在节点视角访问 */
export const isCenterPage = (path: string): boolean =>
    /^\/(?:node|ssh|system\/config|system\/audit\/logs|account\/members)(?:\/|$)/.test(path)

/** 节点可选能力：key 与受管机上报的 features 一致，value 为界面显示名 */
const NODE_FEATURES: Record<string, string> = {
    portmap: '端口转发'
}

/** 协商成功但缺少的可选能力名称（供界面提示「部分功能不可用」）；未协商成功时无从判断，返回空 */
export const nodeMissingFeatures = (node: { negotiated: boolean; features?: string[] }): string[] =>
    node.negotiated ? Object.keys(NODE_FEATURES).filter(key => !node.features?.includes(key)).map(key => NODE_FEATURES[key]) : []

/** 切换到指定节点（空字符串为本机）。整页刷新，使登录态之外的页面状态按新节点重新初始化 */
export const switchNode = (id: string, hash = '') => {
    window.location.href = nodeLocation().root + (id ? `n/${id}/` : '') + hash
}
