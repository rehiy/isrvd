<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import { wsUrl } from '@/service/client.ts'
import type { SSHHostInfo } from '@/service/types'

import { TerminalPanel, WsTerminal } from '@/component/terminal'
import type { TerminalAdapter } from '@/component/terminal'

import SftpPanel from './widget/sftp-panel.vue'

@Component({ components: { TerminalPanel, SftpPanel } })
class SSHClientPage extends Vue {
    portal = usePortal()

    @Ref readonly containerRef!: HTMLDivElement

    host: SSHHostInfo | null = null
    adapter: TerminalAdapter | null = null

    // ─── 拖拽分隔条 ───
    sftpHeight = 0
    isDragging = false
    dragStartY = 0
    dragStartHeight = 0

    get hostId() { return this.$route.params.id as string }
    get connected() { return this.adapter?.connected ?? false }

    async mounted() {
        await this.loadHost()
        this.adapter = new WsTerminal(wsUrl(`ssh/to/${encodeURIComponent(this.hostId)}?token=${this.portal.token || ''}`))
        this.$nextTick(() => this.initSftpHeight())
    }

    unmounted() {
        this.cleanupDrag()
    }

    async loadHost() {
        try {
            const res = await api.sshHostInspect(this.hostId)
            this.host = res.payload || null
        } catch {}
    }

    handleDisconnect() {
        this.adapter?.disconnect()
        this.adapter = null
    }

    handleReconnect() {
        this.adapter = new WsTerminal(wsUrl(`ssh/to/${encodeURIComponent(this.hostId)}?token=${this.portal.token || ''}`))
    }

    initSftpHeight() {
        const containerH = this.containerRef?.clientHeight ?? 600
        this.sftpHeight = Math.floor(containerH * 0.4)
    }

    // ─── 拖拽逻辑 ───
    onDragStart(e: MouseEvent) {
        this.isDragging = true
        this.dragStartY = e.clientY
        this.dragStartHeight = this.sftpHeight
        document.addEventListener('mousemove', this.onDragMove)
        document.addEventListener('mouseup', this.onDragEnd)
        document.body.classList.add('drag-resizing')
    }

    onDragMove(e: MouseEvent) {
        if (!this.isDragging) return
        const containerH = this.containerRef?.clientHeight ?? 600
        const delta = this.dragStartY - e.clientY
        this.sftpHeight = Math.min(Math.max(this.dragStartHeight + delta, 120), containerH - 120)
    }

    onDragEnd() {
        this.isDragging = false
        this.cleanupDrag()
        // TerminalPanel 内部 ResizeObserver 会自动触发 fit，无需手动调用
    }

    cleanupDrag() {
        document.removeEventListener('mousemove', this.onDragMove)
        document.removeEventListener('mouseup', this.onDragEnd)
        document.body.classList.remove('drag-resizing')
    }
}

export default toNative(SSHClientPage)
</script>

<template>
  <div class="h-[calc(100vh-100px)]">
    <div ref="containerRef" class="h-full card flex flex-col overflow-hidden">
      <!-- Toolbar -->
      <div class="card-toolbar">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3 min-w-0">
            <div class="page-icon bg-teal-500 flex-shrink-0">
              <i class="fas fa-terminal text-white text-sm"></i>
            </div>
            <div class="min-w-0">
              <h1 class="text-lg font-semibold text-slate-800 truncate">{{ host?.name || 'SSH 终端' }}</h1>
              <p class="text-xs text-slate-500 truncate">{{ host ? `${host.user} @ ${host.addr}` : '正在加载主机信息...' }}</p>
            </div>
          </div>
          <div class="flex items-center gap-2 flex-shrink-0">
            <button v-if="!connected" class="btn btn-emerald" @click="handleReconnect()">
              <i class="fas fa-plug"></i><span class="hidden md:inline">连接终端</span>
            </button>
            <button v-else class="btn btn-secondary" @click="handleDisconnect()">
              <i class="fas fa-plug-circle-xmark"></i><span class="hidden md:inline">断开连接</span>
            </button>
          </div>
        </div>
      </div>

      <!-- 主内容区：终端 + 分隔条 + 文件管理 -->
      <div class="flex-1 flex flex-col min-h-0 overflow-hidden">
        <!-- 终端 -->
        <div class="terminal-pane">
          <TerminalPanel v-if="adapter" :adapter="adapter" />
        </div>

        <!-- 拖拽分隔条 -->
        <div
          class="flex-shrink-0 h-1.5 bg-slate-100 hover:bg-slate-200 cursor-row-resize transition-colors flex items-center justify-center group"
          :class="{ 'bg-slate-200': isDragging }"
          @mousedown.prevent="onDragStart"
        >
          <div class="w-8 h-0.5 rounded-full bg-slate-300 group-hover:bg-slate-400 transition-colors" :class="{ 'bg-slate-400': isDragging }"></div>
        </div>

        <!-- 文件管理面板 -->
        <div class="flex-shrink-0 min-h-0 overflow-auto" :style="{ height: sftpHeight + 'px' }">
          <SftpPanel :host-id="hostId" />
        </div>
      </div>
    </div>
  </div>
</template>
