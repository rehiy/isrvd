import { defineStore } from 'pinia'
import { ref } from 'vue'

import api from '@/service/api'
import type { NodeInfo } from '@/service/types'

import { POLL_INTERVAL } from '@/helper/format'

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
    const loadFailed = ref(false) // 最近一次加载失败；用于区分「加载失败」与「暂无节点」

    // ─── 操作定义 ───

    // 并发调用共用同一次请求，避免轮询与手动刷新叠加
    let inflight: Promise<void> | null = null

    let generation = 0

    function load(silent = false) {
        if (!silent) {
            loading.value = true
        }
        const requestGeneration = generation
        inflight ??= api.nodeList()
            .then(res => {
                if (requestGeneration === generation) {
                    nodes.value = res.payload || []
                    loadFailed.value = false
                }
            })
            .catch(() => {
                if (requestGeneration === generation) {
                    loadFailed.value = true
                }
            })
            .finally(() => {
                if (requestGeneration === generation) {
                    inflight = null
                    loading.value = false
                }
            })
        return inflight
    }

    // 开始周期刷新，返回停止函数；同一时刻只有一个页面轮询
    function poll() {
        const timer = setInterval(() => load(true), POLL_INTERVAL)
        return () => clearInterval(timer)
    }

    function reset() {
        generation++
        inflight = null
        nodes.value = []
        loading.value = true
        loadFailed.value = false
    }

    return {
        // 状态
        nodes,
        loading,
        loadFailed,
        // 操作
        load,
        poll,
        reset,
    }
})
