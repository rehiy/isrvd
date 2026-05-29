<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { SSHHostInfo } from '@/service/types'

import * as SSHTerminal from '@/helper/ssh'
import SftpPanel from './widget/sftp-panel.vue'

@Component({ components: { SftpPanel } })
class SSHTerminalPage extends Vue {
    portal = usePortal()

    @Ref readonly xtermRef!: HTMLDivElement
    @Ref readonly containerRef!: HTMLDivElement

    // ─── 数据属性 ───
    host: SSHHostInfo | null = null
    connected = false

    // ─── 拖拽分隔条 ───
    sftpHeight = 280
    isDragging = false
    dragStartY = 0
    dragStartHeight = 0

    // ─── 计算属性 ───
    get hostId() {
        return this.$route.params.id as string
    }

    // ─── 方法 ───
    async loadHost() {
        try {
            const res = await api.sshHost(this.hostId)
            this.host = res.payload || null
        } catch {
            this.portal.showNotification('error', '加载主机信息失败')
        }
    }

    handleConnect() {
        if (this.connected) return
        this.connected = true
        SSHTerminal.create(this.xtermRef, this.portal.token || '', this.hostId)
    }

    handleDisconnect() {
        SSHTerminal.destroy()
        this.xtermRef.innerHTML = ''
        this.connected = false
    }

    // ─── 拖拽逻辑 ───
    onDragStart(e: MouseEvent) {
        this.isDragging = true
        this.dragStartY = e.clientY
        this.dragStartHeight = this.sftpHeight
        document.addEventListener('mousemove', this.onDragMove)
        document.addEventListener('mouseup', this.onDragEnd)
        document.body.style.cursor = 'row-resize'
        document.body.style.userSelect = 'none'
    }

    onDragMove(e: MouseEvent) {
        if (!this.isDragging) return
        const containerH = this.containerRef?.clientHeight ?? 600
        const delta = this.dragStartY - e.clientY
        const newHeight = Math.min(
            Math.max(this.dragStartHeight + delta, 120),
            containerH - 120
        )
        this.sftpHeight = newHeight
    }

    onDragEnd() {
        this.isDragging = false
        document.removeEventListener('mousemove', this.onDragMove)
        document.removeEventListener('mouseup', this.onDragEnd)
        document.body.style.cursor = ''
        document.body.style.userSelect = ''
    }

    // ─── 生命周期 ───
    async mounted() {
        await this.loadHost()
        this.handleConnect()
    }

    unmounted() {
        SSHTerminal.destroy()
        this.connected = false
        document.removeEventListener('mousemove', this.onDragMove)
        document.removeEventListener('mouseup', this.onDragEnd)
    }
}

export default toNative(SSHTerminalPage)
</script>

<template>
  <div class="terminal-page">
    <div ref="containerRef" class="h-full card flex flex-col overflow-hidden">
      <!-- Toolbar -->
      <div class="card-toolbar">
        <!-- 桌面端 -->
        <div class="hidden md:flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="page-icon bg-teal-600">
              <i class="fas fa-terminal text-white text-sm"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">{{ host?.name || 'SSH 终端' }}</h1>
              <p class="text-xs text-slate-500">{{ host ? `${host.user} @ ${host.addr}` : '正在加载主机信息...' }}</p>
            </div>
          </div>
          <div class="flex items-center gap-2 flex-shrink-0">
            <button v-if="!connected" class="btn btn-emerald" @click="handleConnect()">
              <i class="fas fa-plug"></i>连接终端
            </button>
            <button v-else class="btn btn-secondary" @click="handleDisconnect()">
              <i class="fas fa-plug-circle-xmark"></i>断开连接
            </button>
          </div>
        </div>
        <!-- 移动端 -->
        <div class="flex md:hidden items-center justify-between">
          <div class="flex items-center gap-3 min-w-0 flex-1">
            <div class="page-icon bg-teal-600 flex-shrink-0">
              <i class="fas fa-terminal text-white text-sm"></i>
            </div>
            <div class="min-w-0">
              <h1 class="text-lg font-semibold text-slate-800 truncate">{{ host?.name || 'SSH 终端' }}</h1>
              <p class="text-xs text-slate-500 truncate">{{ host ? `${host.user} @ ${host.addr}` : '正在加载主机信息...' }}</p>
            </div>
          </div>
          <div class="flex items-center gap-2 flex-shrink-0">
            <button v-if="!connected" class="btn btn-emerald w-9 h-9 !px-0" title="连接终端" @click="handleConnect()">
              <i class="fas fa-plug text-sm"></i>
            </button>
            <button v-else class="btn btn-secondary w-9 h-9 !px-0" title="断开连接" @click="handleDisconnect()">
              <i class="fas fa-plug-circle-xmark text-sm"></i>
            </button>
          </div>
        </div>
      </div>

      <!-- 终端区域 -->
      <div class="terminal-body flex-1 min-h-0">
        <div class="terminal-pane h-full">
          <div ref="xtermRef" class="h-full rounded-lg overflow-hidden"></div>
        </div>
      </div>

      <!-- 拖拽分隔条 -->
      <div
        class="flex-shrink-0 h-1.5 bg-slate-100 hover:bg-teal-200 cursor-row-resize transition-colors flex items-center justify-center group"
        :class="{ 'bg-teal-200': isDragging }"
        @mousedown.prevent="onDragStart"
      >
        <div class="w-8 h-0.5 rounded-full bg-slate-300 group-hover:bg-teal-400 transition-colors" :class="{ 'bg-teal-400': isDragging }"></div>
      </div>

      <!-- SFTP 文件管理面板 -->
      <SftpPanel :host-id="hostId" :height="sftpHeight" />
    </div>
  </div>
</template>
