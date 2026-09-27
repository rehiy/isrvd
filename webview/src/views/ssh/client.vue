<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import { wsUrl } from '@/service/client.ts'
import type { SSHHostInfo } from '@/service/types'

import { SplitPane } from '@/helper/split-pane'

import { TerminalPanel, WsTerminal } from '@/component/terminal'
import type { TerminalAdapter } from '@/component/terminal'

import SftpPanel from './widget/sftp-panel.vue'

@Component({ components: { TerminalPanel, SftpPanel } })
class SSHClientPage extends Vue {
    portal = usePortal()

    @Ref readonly containerRef!: HTMLDivElement

    host: SSHHostInfo | null = null
    adapter: TerminalAdapter | null = null

    filePane = new SplitPane()

    get hostId() { return this.$route.params.id as string }
    get connected() { return this.adapter?.connected ?? false }

    async mounted() {
        await this.loadHost()
        this.adapter = new WsTerminal(wsUrl(`ssh/to/${encodeURIComponent(this.hostId)}?token=${this.portal.token || ''}`))
        this.$nextTick(() => this.filePane.initialize(this.containerRef?.clientHeight ?? 600))
    }

    unmounted() {
        this.filePane.stop()
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
}

export default toNative(SSHClientPage)
</script>

<template>
  <div class="h-[calc(100vh-4rem)]">
    <div ref="containerRef" class="h-full page flex flex-col overflow-hidden">
      <!-- Toolbar -->
      <div class="page-toolbar page-toolbar-static">
        <div class="flex items-center justify-between">
          <div class="title-group-static">
            <div class="page-icon bg-teal-500 flex-shrink-0">
              <i class="fas fa-terminal text-white text-sm"></i>
            </div>
            <div class="min-w-0">
              <h1 class="title-text">{{ host?.name || 'SSH 终端' }}</h1>
              <p class="text-xs text-slate-500 truncate">{{ host ? `${host.user} @ ${host.addr}` : '正在加载主机信息...' }}</p>
            </div>
          </div>
          <div class="action-group">
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
          :class="{ 'bg-slate-200': filePane.dragging }"
          @mousedown.prevent="filePane.start($event, containerRef)"
        >
          <div class="w-8 h-0.5 rounded-full bg-slate-300 group-hover:bg-slate-400 transition-colors" :class="{ 'bg-slate-400': filePane.dragging }"></div>
        </div>

        <!-- 文件管理面板 -->
        <div class="flex-shrink-0 min-h-0 overflow-auto" :style="{ height: filePane.height + 'px' }">
          <SftpPanel :host-id="hostId" />
        </div>
      </div>
    </div>
  </div>
</template>
