import { defineStore } from 'pinia'
import { ref } from 'vue'

import { POLL_INTERVAL } from '@/helper/format'
import api from '@/service/api'
import type { NodeInfo } from '@/service/types'

/**
 * 节点 Store（仅中控模式使用）
 *
 * 持有节点列表，供头部节点切换器、节点管理与端口转发页共用。
 * 当前节点由页面所在的路径决定（/n/<节点ID>/，见 helper/node.ts），切换节点是一次整页跳转，不放进 store。
 */
export const useNodeStore = defineStore('node', () => {
    // ─── 状态定义 ───

    const nodes = ref<NodeInfo[]>([])
    const loading = ref(true) // 首次加载或手动刷新期间为 true；静默刷新不改变它

    // ─── 操作定义 ───

    // 并发调用共用同一次请求，避免轮询与手动刷新叠加
    let inflight: Promise<void> | null = null

    function load(silent = false) {
        if (!silent) loading.value = true
        inflight ??= api.nodeList()
            .then(res => { nodes.value = res.payload || [] })
            .catch(() => {
                // 错误提示由请求拦截器统一处理，保留现有列表
            })
            .finally(() => { inflight = null })
        return inflight.finally(() => { loading.value = false })
    }

    // 开始周期刷新，返回停止函数；同一时刻只有一个页面轮询
    function poll() {
        const timer = setInterval(() => load(true), POLL_INTERVAL)
        return () => clearInterval(timer)
    }

    function reset() {
        nodes.value = []
        loading.value = true
    }

    return {
        // 状态
        nodes,
        loading,
        // 操作
        load,
        poll,
        reset,
    }
})
