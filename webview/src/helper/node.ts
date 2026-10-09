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

/** 切换到指定节点（空字符串为本机）。整页刷新，使登录态之外的页面状态按新节点重新初始化 */
export const switchNode = (id: string, hash = '') => {
    window.location.href = nodeLocation().root + (id ? `n/${id}/` : '') + hash
}
