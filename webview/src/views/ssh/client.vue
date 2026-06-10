<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { SSHHostInfo } from '@/service/types'

import * as SSHClient from '@/helper/ssh'

import SftpPanel from './widget/sftp-panel.vue'

@Component({ components: { SftpPanel } })
class SSHClientPage extends Vue {
    portal = usePortal()

    @Ref readonly xtermRef!: HTMLDivElement
    @Ref readonly containerRef!: HTMLDivElement

    // ─── 数据属性 ───
    host: SSHHostInfo | null = null
    connected = false
    viewMode: 'all' | 'terminal' | 'sftp' = 'all'

    // ─── 拖拽分隔条 ───
    sftpHeight = 280
    isDragging = false
    dragStartY = 0
    dragStartHeight = 0

    // ─── 计算属性 ───
    get hostId() {
        return this.$route.params.id as string
    }

    // ─── 生命周期 ───
    async mounted() {
        await this.loadHost()
        this.handleConnect()
        this.$nextTick(() => {
            this.initAllViewHeight()
        })
    }

    unmounted() {
        SSHClient.destroy()
        this.connected = false
        this.cleanupDrag()
    }

    // ─── 数据加载 ───
    async loadHost() {
        try {
            const res = await api.sshHostInspect(this.hostId)
            this.host = res.payload || null
        } catch {}
    }

    // ─── 终端连接管理 ───
    handleConnect() {
        if (this.connected) return
        this.connected = true
        SSHClient.create(this.xtermRef, this.portal.token || '', this.hostId)
    }

    handleDisconnect() {
        SSHClient.destroy()
        if (this.xtermRef) {
            this.xtermRef.innerHTML = ''
        }
        this.connected = false
    }

    // ─── 视图模式切换 ───
    switchViewMode(mode: 'all' | 'terminal' | 'sftp') {
        this.viewMode = mode
        this.$nextTick(() => {
            if (mode === 'all') {
                this.initAllViewHeight()
            }
            this.fitTerminalDelayed()
        })
    }

    // ─── 高度管理 ───
    initAllViewHeight() {
        if (this.viewMode !== 'all') return
        const containerH = this.containerRef?.clientHeight ?? 600
        this.sftpHeight = Math.floor(containerH * 0.3)
    }

    fitTerminalDelayed() {
        setTimeout(() => {
            SSHClient.fitTerminal()
        }, 100)
    }

    // ─── 拖拽逻辑 ───
    onDragStart(e: MouseEvent) {
        if (this.viewMode !== 'all') return
        this.isDragging = true
        this.dragStartY = e.clientY
        this.dragStartHeight = this.sftpHeight
        document.addEventListener('mousemove', this.onDragMove)
        document.addEventListener('mouseup', this.onDragEnd)
        document.body.style.cursor = 'row-resize'
        document.body.style.userSelect = 'none'
    }

    onDragMove(e: MouseEvent) {
        if (!this.isDragging || this.viewMode !== 'all') return
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
        this.cleanupDrag()
        this.$nextTick(() => {
            SSHClient.fitTerminal()
        })
    }

    cleanupDrag() {
        document.removeEventListener('mousemove', this.onDragMove)
        document.removeEventListener('mouseup', this.onDragEnd)
        document.body.style.cursor = ''
        document.body.style.userSelect = ''
    }
}

export default toNative(SSHClientPage)
</script>

<template>
  <div class="h-[calc(100vh-100px)]">
    <div ref="containerRef" class="h-full card flex flex-col overflow-hidden">
      <!-- Toolbar -->
      <div class="card-toolbar">
        <!-- 桌面端 -->
        <div class="hidden md:flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="page-icon bg-teal-500">
              <i class="fas fa-terminal text-white text-sm"></i>
            </div>
            <div class="min-w-0">
              <h1 class="text-lg font-semibold text-slate-800 truncate">{{ host?.name || 'SSH 终端' }}</h1>
              <p class="text-xs text-slate-500 truncate">{{ host ? `${host.user} @ ${host.addr}` : '正在加载主机信息...' }}</p>
            </div>
          </div>
          <div class="flex items-center gap-2 flex-shrink-0">
            <!-- 视图切换 -->
            <div class="tab-group">
              <button type="button" :class="['tab-btn', viewMode === 'all' ? 'tab-btn-active text-teal-600' : 'tab-btn-inactive']" @click="switchViewMode('all')">
                <i class="fas fa-border-all"></i>全部
              </button>
              <button type="button" :class="['tab-btn', viewMode === 'terminal' ? 'tab-btn-active text-teal-600' : 'tab-btn-inactive']" @click="switchViewMode('terminal')">
                <i class="fas fa-terminal"></i>终端
              </button>
              <button type="button" :class="['tab-btn', viewMode === 'sftp' ? 'tab-btn-active text-teal-600' : 'tab-btn-inactive']" @click="switchViewMode('sftp')">
                <i class="fas fa-folder-open"></i>文件
              </button>
            </div>
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
            <div class="page-icon bg-teal-500 flex-shrink-0">
              <i class="fas fa-terminal text-white text-sm"></i>
            </div>
            <div class="min-w-0">
              <h1 class="text-lg font-semibold text-slate-800 truncate">{{ host?.name || 'SSH 终端' }}</h1>
              <p class="text-xs text-slate-500 truncate">{{ host ? `${host.user} @ ${host.addr}` : '正在加载主机信息...' }}</p>
            </div>
          </div>
          <div class="flex items-center gap-2 flex-shrink-0">
            <div class="tab-group">
              <button type="button" class="tab-btn !h-8 !px-2" :class="viewMode === 'all' ? 'tab-btn-active text-teal-600' : 'tab-btn-inactive'" title="全部" @click="switchViewMode('all')">
                <i class="fas fa-border-all text-xs"></i>
              </button>
              <button type="button" class="tab-btn !h-8 !px-2" :class="viewMode === 'terminal' ? 'tab-btn-active text-teal-600' : 'tab-btn-inactive'" title="终端" @click="switchViewMode('terminal')">
                <i class="fas fa-terminal text-xs"></i>
              </button>
              <button type="button" class="tab-btn !h-8 !px-2" :class="viewMode === 'sftp' ? 'tab-btn-active text-teal-600' : 'tab-btn-inactive'" title="文件" @click="switchViewMode('sftp')">
                <i class="fas fa-folder-open text-xs"></i>
              </button>
            </div>
            <button v-if="!connected" class="btn btn-emerald w-9 h-9 !px-0" title="连接终端" @click="handleConnect()">
              <i class="fas fa-plug text-sm"></i>
            </button>
            <button v-else class="btn btn-secondary w-9 h-9 !px-0" title="断开连接" @click="handleDisconnect()">
              <i class="fas fa-plug-circle-xmark text-sm"></i>
            </button>
          </div>
        </div>
      </div>

      <!-- 主内容区域 -->
      <div class="flex-1 flex flex-col min-h-0 overflow-hidden">
        <!-- 终端区域：使用 v-show 保持 DOM 存在 -->
        <div
          v-show="viewMode !== 'sftp'"
          class="min-h-0"
          :class="{
            'flex-1': viewMode === 'terminal',
            'flex-[7_0_0%]': viewMode === 'all'
          }"
        >
          <div class="terminal-pane h-full">
            <div ref="xtermRef" class="h-full rounded-lg overflow-hidden"></div>
          </div>
        </div>

        <!-- 全部视图：拖拽条 + SFTP -->
        <template v-if="viewMode === 'all'">
          <!-- 拖拽分隔条 -->
          <div
            class="flex-shrink-0 h-1.5 bg-slate-100 hover:bg-slate-200 cursor-row-resize transition-colors flex items-center justify-center group"
            :class="{ 'bg-slate-200': isDragging }"
            @mousedown.prevent="onDragStart"
          >
            <div class="w-8 h-0.5 rounded-full bg-slate-300 group-hover:bg-slate-400 transition-colors" :class="{ 'bg-slate-400': isDragging }"></div>
          </div>

          <!-- SFTP 文件管理面板 -->
          <div class="flex-shrink-0 min-h-0 overflow-auto" :style="{ height: sftpHeight + 'px' }">
            <SftpPanel :host-id="hostId" :height="sftpHeight + 'px'" />
          </div>
        </template>

        <!-- 文件独立视图 -->
        <div v-show="viewMode === 'sftp'" class="flex-1 min-h-0 overflow-auto">
          <SftpPanel :host-id="hostId" :height="'100%'" />
        </div>
      </div>
    </div>
  </div>
</template>
