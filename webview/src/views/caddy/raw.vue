<script lang="ts">
import { json } from '@codemirror/lang-json'
import { Codemirror } from 'vue-codemirror'
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'

@Component({
    components: { Codemirror }
})
class CaddyRaw extends Vue {
    portal = usePortal()

    raw = ''
    loading = false
    saving = false

    readonly extensions = [json()]

    async loadConfig() {
        this.loading = true
        try {
            const cfg = (await api.caddyConfig()).payload
            this.raw = JSON.stringify(cfg ?? null, null, 2)
        } catch {
            this.portal.showNotification('error', '加载 Caddy 配置失败')
        } finally {
            this.loading = false
        }
    }

    async saveConfig() {
        let parsed: unknown
        try {
            parsed = JSON.parse(this.raw)
        } catch {
            this.portal.showNotification('error', 'JSON 格式有误，请检查编辑器中的错误提示')
            return
        }
        this.portal.showConfirm({
            title: '提交完整配置',
            message: '将整体替换 Caddy 当前运行配置，请确认编辑内容无误。',
            icon: 'fa-cloud-arrow-up',
            iconColor: 'indigo',
            confirmText: '确认提交',
            onConfirm: async () => {
                this.saving = true
                try {
                    await api.caddyConfigLoad(parsed)
                    this.portal.showNotification('success', '配置已应用')
                    this.loadConfig()
                } catch (e: unknown) {
                    this.portal.showNotification('error', (e instanceof Error ? e.message : '') || '应用失败')
                } finally {
                    this.saving = false
                }
            }
        })
    }

    mounted() {
        this.loadConfig()
    }
}

export default toNative(CaddyRaw)
</script>

<template>
  <div>
    <div class="card mb-4">
      <div class="card-toolbar">
        <!-- 桌面端 -->
        <div class="hidden md:flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="page-icon bg-slate-700"><i class="fas fa-code text-white"></i></div>
            <div class="min-w-0"><h1 class="text-lg font-semibold text-slate-800 truncate">Caddy 原始配置</h1><p class="text-xs text-slate-500 truncate">直接查看和编辑 Caddy 的 JSON 运行配置</p></div>
          </div>
          <div class="flex items-center gap-2 flex-shrink-0">
            <button class="btn btn-secondary" @click="loadConfig()"><i class="fas fa-rotate"></i>刷新</button>
            <button v-if="portal.hasPerm('POST /api/caddy/config')" :disabled="saving" class="btn btn-indigo" @click="saveConfig()">
              <i v-if="saving" class="fas fa-spinner fa-spin"></i>
              <i v-else class="fas fa-cloud-arrow-up"></i>
              <span>{{ saving ? '提交中...' : '提交配置' }}</span>
            </button>
          </div>
        </div>
        <!-- 移动端 -->
        <div class="flex md:hidden items-center justify-between">
          <div class="flex items-center gap-3 min-w-0 flex-1">
            <div class="page-icon bg-slate-700"><i class="fas fa-code text-white"></i></div>
            <div class="min-w-0">
              <h1 class="text-lg font-semibold text-slate-800 truncate">Caddy 原始配置</h1>
              <p class="text-xs text-slate-500 truncate">查看和编辑 JSON 配置</p>
            </div>
          </div>
          <div class="flex items-center gap-1 flex-shrink-0">
            <button class="btn btn-secondary w-9 h-9 !px-0" title="刷新" @click="loadConfig()"><i class="fas fa-rotate text-sm"></i></button>
            <button v-if="portal.hasPerm('POST /api/caddy/config')" :disabled="saving" class="btn btn-indigo w-9 h-9 !px-0" title="提交配置" @click="saveConfig()"><i class="fas fa-cloud-arrow-up text-sm"></i></button>
          </div>
        </div>
      </div>

      <div v-if="loading" class="empty-state"><div class="w-12 h-12 spinner mb-3"></div><p class="text-slate-500">加载中...</p></div>
      <div v-else class="p-4 md:p-6 space-y-3">
        <div class="editor-container">
          <Codemirror v-model="raw" :style="{ height: '65vh' }" :extensions="extensions" />
        </div>
        <p class="text-xs text-slate-400 flex items-start gap-1">
          <i class="fas fa-circle-info mt-0.5 flex-shrink-0"></i>
          <span>提交将通过 <code class="px-1 bg-slate-100 rounded">POST /load</code> 整体替换 Caddy 运行配置，操作前请确保已备份。</span>
        </p>
      </div>
    </div>
  </div>
</template>
