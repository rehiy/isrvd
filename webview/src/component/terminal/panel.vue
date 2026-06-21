<script lang="ts">
/**
 * 通用终端面板组件
 *
 * Props：
 *   adapter   终端适配器（必填），实现 connect / disconnect / fit
 *
 * Expose：
 *   connect()     手动连接
 *   disconnect()  手动断开
 *
 * 用法：
 *   <TerminalPanel :adapter="adapter" />
 */
import { Component, Prop, Ref, Vue, Watch, toNative } from 'vue-facing-decorator'

import type { TerminalAdapter } from './types'

@Component({ expose: ['connect', 'disconnect'] })
class TerminalPanel extends Vue {
    @Prop({ required: true }) adapter!: TerminalAdapter

    @Ref readonly xtermRef!: HTMLDivElement

    private resizeObserver: ResizeObserver | null = null

    mounted() {
        this.connect()
        // ResizeObserver 监听容器变化，触发 fit（处理分隔条拖拽等场景）
        this.resizeObserver = new ResizeObserver(() => this.adapter.fit())
        this.resizeObserver.observe(this.xtermRef)
    }

    unmounted() {
        this.resizeObserver?.disconnect()
        this.resizeObserver = null
        this.disconnect()
    }

    @Watch('adapter')
    onAdapterChange(next: TerminalAdapter, prev: TerminalAdapter) {
        if (next === prev) return
        prev.disconnect()
        this.$nextTick(() => this.connect())
    }

    connect() {
        this.adapter.connect(this.xtermRef)
    }

    disconnect() {
        this.adapter.disconnect()
    }
}

export default toNative(TerminalPanel)
</script>

<template>
  <div ref="xtermRef" class="h-full w-full rounded-lg overflow-hidden"></div>
</template>
