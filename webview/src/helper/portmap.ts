import api from '@/service/api'
import type { NodePortMap } from '@/service/types'

/**
 * 端口转发规则按节点整体替换，增删改都要基于节点最新的规则列表，避免覆盖他人刚做的修改。
 * fn 拿到该节点最新的规则副本，返回修改后的完整列表；返回 null 表示目标规则已不存在，不提交。
 * 返回 false 表示节点或规则已不存在（未提交），调用方负责提示。
 */
export const portmapPatch = async (nodeId: string, fn: (list: NodePortMap[]) => NodePortMap[] | null): Promise<boolean> => {
    const res = await api.nodeList()
    const node = (res.payload || []).find(n => n.id === nodeId)
    const next = node ? fn([...(node.portmaps || [])]) : null
    if (!next) return false
    await api.nodePortMapUpdate(nodeId, { portmaps: next })
    return true
}

/** 规则在节点内的唯一标识：同一协议下监听端口全局唯一 */
export const portmapIs = (a: NodePortMap, b: Pick<NodePortMap, 'proto' | 'listen'>): boolean =>
    a.proto === b.proto && a.listen === b.listen
