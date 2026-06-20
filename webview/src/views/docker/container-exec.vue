<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import { wsUrl } from '@/service/client.ts'
import type { DockerContainerInfo } from '@/service/types'

import { TerminalPanel, WsTerminal } from '@/component/terminal'
import type { TerminalAdapter } from '@/component/terminal'

import ContainerFilePanel from './widget/container-file-panel.vue'

@Component({ components: { TerminalPanel, ContainerFilePanel } })
class ContainerTerminal extends Vue {
    portal = usePortal()

    @Ref readonly containerRef!: HTMLDivElement

    container: DockerContainerInfo | null = null
    adapter: TerminalAdapter | null = null
    shell = '/bin/sh'

    // ─── 拖拽分隔条 ───
    fileHeight = 0
    isDragging = false
    dragStartY = 0
    dragStartHeight = 0

    get containerId() { return this.$route.params.id as string }
    get connected() { return this.adapter?.connected ?? false }

    async mounted() {
        await this.loadContainer()
        this.$nextTick(() => this.initFileHeight())
    }

    unmounted() {
        this.adapter?.disconnect()
        this.adapter = null
        this.cleanupDrag()
    }

    async loadContainer() {
        try {
            const res = await api.dockerContainerList(true)
            this.container = (res.payload || []).find((c: DockerContainerInfo) => c.id === this.containerId) ?? null
            if (!this.container) {
                this.portal.showNotification('error', '容器不存在')
                this.$router.push('/docker/containers')
                return
            }
            this.handleConnect()
        } catch {}
    }

    handleConnect() {
        if (this.connected || !this.container) return
        this.adapter = new WsTerminal(wsUrl(`docker/container/${encodeURIComponent(this.containerId)}/exec?token=${this.portal.token ?? ''}&shell=${encodeURIComponent(this.shell)}`))
    }

    handleDisconnect() {
        this.adapter?.disconnect()
        this.adapter = null
    }

    handleShellChange() {
        if (this.connected) {
            this.handleDisconnect()
            this.$nextTick(() => this.handleConnect())
        }
    }

    initFileHeight() {
        const containerH = this.containerRef?.clientHeight ?? 600
        this.fileHeight = Math.floor(containerH * 0.4)
    }

    // ─── 拖拽逻辑 ───
    onDragStart(e: MouseEvent) {
        this.isDragging = true
        this.dragStartY = e.clientY
        this.dragStartHeight = this.fileHeight
        document.addEventListener('mousemove', this.onDragMove)
        document.addEventListener('mouseup', this.onDragEnd)
        document.body.style.cursor = 'row-resize'
        document.body.style.userSelect = 'none'
    }

    onDragMove(e: MouseEvent) {
        if (!this.isDragging) return
        const containerH = this.containerRef?.clientHeight ?? 600
        const delta = this.dragStartY - e.clientY
        this.fileHeight = Math.min(Math.max(this.dragStartHeight + delta, 120), containerH - 120)
    }

    onDragEnd() {
        this.isDragging = false
        this.cleanupDrag()
        // TerminalPanel 内部 ResizeObserver 会自动触发 fit，无需手动调用
    }

    cleanupDrag() {
        document.removeEventListener('mousemove', this.onDragMove)
        document.removeEventListener('mouseup', this.onDragEnd)
        document.body.style.cursor = ''
        document.body.style.userSelect = ''
    }
}

export default toNative(ContainerTerminal)
</script>

<template>
  <div class="h-[calc(100vh-100px)]">
    <div ref="containerRef" class="h-full card flex flex-col overflow-hidden">
      <!-- Toolbar -->
      <div class="card-toolbar">
        <div class="hidden md:flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div :class="['page-icon', container?.state === 'running' ? 'bg-emerald-400' : 'bg-slate-400']">
              <i class="fas fa-terminal text-white text-sm"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">容器终端</h1>
              <p class="text-xs text-slate-500 font-mono truncate max-w-xs">{{ container ? `${container.name || container.id} · ${container.image}` : '加载中...' }}</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <select v-model="shell" :disabled="connected" class="w-28 select-sm" @change="handleShellChange">
              <option value="/bin/sh">/bin/sh</option>
              <option value="/bin/bash">/bin/bash</option>
              <option value="/bin/ash">/bin/ash</option>
            </select>
            <button v-if="!connected" type="button" class="btn btn-emerald" @click="handleConnect()">
              <i class="fas fa-plug"></i>连接
            </button>
            <button v-else type="button" class="btn btn-secondary" @click="handleDisconnect()">
              <i class="fas fa-plug-circle-xmark"></i>断开
            </button>
          </div>
        </div>
        <div class="flex md:hidden items-center justify-between">
          <div class="flex items-center gap-3 min-w-0 flex-1">
            <div :class="['page-icon', container?.state === 'running' ? 'bg-emerald-400' : 'bg-slate-400']">
              <i class="fas fa-terminal text-white text-sm"></i>
            </div>
            <div class="min-w-0">
              <h1 class="text-lg font-semibold text-slate-800 truncate">容器终端</h1>
              <p class="text-xs text-slate-500 font-mono truncate">{{ container ? `${container.name || container.id} · ${container.image}` : '加载中...' }}</p>
            </div>
          </div>
          <div class="flex items-center gap-1 flex-shrink-0">
            <select v-model="shell" :disabled="connected" class="w-24 select-sm" @change="handleShellChange">
              <option value="/bin/sh">/bin/sh</option>
              <option value="/bin/bash">/bin/bash</option>
              <option value="/bin/ash">/bin/ash</option>
            </select>
            <button v-if="!connected" type="button" class="btn btn-emerald w-9 h-9 !px-0" title="连接" @click="handleConnect()">
              <i class="fas fa-plug text-sm"></i>
            </button>
            <button v-else type="button" class="btn btn-secondary w-9 h-9 !px-0" title="断开" @click="handleDisconnect()">
              <i class="fas fa-plug-circle-xmark text-sm"></i>
            </button>
          </div>
        </div>
      </div>

      <!-- 主内容区：终端 + 分隔条 + 文件管理 -->
      <div class="flex-1 flex flex-col min-h-0 overflow-hidden">
        <!-- 终端 -->
        <div class="flex-1 min-h-0 p-2">
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
        <div class="flex-shrink-0 min-h-0 overflow-auto" :style="{ height: fileHeight + 'px' }">
          <ContainerFilePanel :container-id="containerId" />
        </div>
      </div>
    </div>
  </div>
</template>
