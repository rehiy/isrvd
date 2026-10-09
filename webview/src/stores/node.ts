import { defineStore } from 'pinia'
import { ref } from 'vue'

import api from '@/service/api'
import type { NodeInfo } from '@/service/types'

/**
 * 节点 Store（仅中控模式使用）
 *
 * 持有节点列表，供头部节点切换器使用。
 * 当前节点由页面所在的路径决定（/n/<节点ID>/，见 helper/node.ts），切换节点是一次整页跳转，不放进 store。
 */
export const useNodeStore = defineStore('node', () => {
    // ─── 状态定义 ───

    const nodes = ref<NodeInfo[]>([])

    // ─── 操作定义 ───

    async function load() {
        try {
            const res = await api.nodeList()
            nodes.value = res.payload || []
        } catch {
            // 错误提示由请求拦截器统一处理，保留现有列表
        }
    }

    function reset() {
        nodes.value = []
    }

    return {
        // 状态
        nodes,
        // 操作
        load,
        reset,
    }
})
