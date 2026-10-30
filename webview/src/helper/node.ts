/**
 * 多节点（中控模式）的地址约定。
 *
 * 中控网关把 /n/<节点ID>/ 下的接口转发给所选节点，页面本身仍由中控提供。
 * 前端使用相对路径（baseURL 为 api/、hash 路由），因此切换节点只需要换页面所在的路径前缀。
 */

import type { Router } from 'vue-router'

const NODE_PATH = /^(.*\/)n\/([0-9A-Za-z-]{10,})\/(?:index\.html)?$/

/** 节点列表的查看权限：仅中控模式启用且当前成员可管理节点时才有，用于判断集中管控入口与页面是否可用 */
export const NODE_LIST_PERM = 'GET /api/node/nodes'

/**
 * 节点列表类页面（节点管理、端口转发）的共用入口，须在启动数据就绪后调用：
 * 无节点列表权限（未启用中控模式或成员无权）时回到概览，而不是展示一个只会报错的页面，并返回 null；
 * 否则静默刷新一次并开始周期刷新，返回停止函数，页面卸载时调用。
 */
export const enterNodePage = (
    portal: { hasPerm(perm: string): boolean; nodeLoad(silent?: boolean): Promise<void>; nodePoll(): () => void },
    router: Router
): (() => void) | null => {
    if (!portal.hasPerm(NODE_LIST_PERM)) {
        router.replace('/overview')
        return null
    }
    portal.nodeLoad(true)
    return portal.nodePoll()
}

/** 当前页面所在的中控根路径（以 / 结尾）与所选节点 ID（本机为空字符串） */
export const nodeLocation = (): { root: string; id: string } => {
    const match = window.location.pathname.match(NODE_PATH)
    if (match) return { root: match[1], id: match[2] }
    return { root: window.location.pathname.replace(/index\.html$/, ''), id: '' }
}

/** 中控对外地址（不含节点前缀与末尾斜杠），受管机用它接入 */
export const centerUrl = (): string =>
    window.location.origin + nodeLocation().root.replace(/\/+$/, '')

/** 仅在中控视角可用的页面：节点管理、用户管理与登录认证配置只属于中控，受管机上不生效。
 *  SSH 随节点转发，个人账户设置仍可在节点视角访问 */
export const isCenterPage = (path: string): boolean =>
    /^\/(?:node|system\/config\/auth|account\/members)(?:\/|$)/.test(path)

/** 节点状态的展示：text 为待翻译的文案，cls 为样式类 */
export const nodeStatus = (node: { status: string; online: boolean }): { text: string; cls: string } => {
    if (node.status === 'pending') return { text: '待接入', cls: 'text-slate-500' }
    if (node.status === 'revoked') return { text: '已吊销', cls: 'text-slate-500' }
    if (!node.online) return { text: '离线', cls: 'text-red-500 font-medium' }
    return { text: '在线', cls: 'text-emerald-600 font-medium' }
}

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
