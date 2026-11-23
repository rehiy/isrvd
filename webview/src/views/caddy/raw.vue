<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'

import { caddyConfigReloadConfirm } from '@/helper/caddy'

import CodeEditor from '@/component/code-editor.vue'

@Component({
    components: { CodeEditor }
})
class CaddyRaw extends Vue {
    portal = usePortal()

    raw = ''
    loading = false
    saving = false
    loaded = false
    loadedRaw = ''

    get canSave() {
        return this.loaded && !this.loading && !this.saving &&
            this.portal.hasPerm('GET /api/caddy/config') &&
            this.portal.hasPerm('POST /api/caddy/config')
    }

    async loadConfig() {
        if (this.loading || this.saving) {
            return
        }
        if (this.loadedRaw && this.raw !== this.loadedRaw) {
            this.portal.showConfirm(caddyConfigReloadConfirm(this.$t, () => this.fetchConfig()))
            return
        }
        await this.fetchConfig()
    }

    async fetchConfig() {
        if (this.loading || !this.portal.hasPerm('GET /api/caddy/config')) {
            return
        }
        this.loading = true
        this.loaded = false
        try {
            const cfg = (await api.caddyConfigInspect()).payload
            if (cfg !== null && (typeof cfg !== 'object' || Array.isArray(cfg))) {
                this.portal.showNotification('error', this.$t('加载失败，请刷新重试'))
                return
            }
            this.raw = JSON.stringify(cfg, null, 2)
            this.loadedRaw = this.raw
            this.loaded = true
        } finally {
            this.loading = false
        }
    }

    async saveConfig() {
        if (!this.canSave) {
            return
        }
        let parsed: unknown
        try {
            parsed = JSON.parse(this.raw)
        } catch {
            this.portal.showNotification('error', this.$t('JSON 格式有误，请检查编辑器中的错误提示'))
            return
        }
        if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
            this.portal.showNotification('error', this.$t('配置 JSON 必须是对象'))
            return
        }
        this.portal.showConfirm({
            title: this.$t('提交完整配置'),
            message: this.$t('将整体替换 Caddy 当前运行配置，请确认编辑内容无误。'),
            icon: 'fa-cloud-arrow-up',
            iconColor: 'slate',
            confirmText: this.$t('确认提交'),
            onConfirm: async () => {
                if (!this.canSave) {
                    return
                }
                this.saving = true
                try {
                    await api.caddyConfigLoad(parsed)
                    this.loadedRaw = this.raw
                    this.portal.showNotification('success', this.$t('配置已应用'))
                    await this.fetchConfig()
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
  <div class="page flex flex-col h-[calc(100dvh-4rem)]">
    <div class="page-toolbar">
      <!-- 桌面端 -->
      <div class="toolbar-desktop">
        <div class="title-group-static">
          <div class="page-icon bg-slate-500"><i class="fas fa-code text-white"></i></div>
          <div class="min-w-0"><h1 class="title-text">{{ $t('原始配置') }}</h1><p class="text-xs text-slate-500 truncate">{{ $t('直接查看和编辑 Caddy 的 JSON 运行配置') }}</p></div>
        </div>
        <div class="action-group">
          <button class="btn btn-secondary" :disabled="loading || saving || !portal.hasPerm('GET /api/caddy/config')" @click="loadConfig()"><i class="fas fa-rotate"></i>{{ $t('刷新') }}</button>
          <button v-if="portal.hasPerm('POST /api/caddy/config')" :disabled="!canSave" :aria-busy="saving" class="btn btn-secondary" @click="saveConfig()">
            <i v-if="saving" class="fas fa-spinner fa-spin"></i>
            <i v-else class="fas fa-cloud-arrow-up"></i>
            <span>{{ $t('提交配置') }}</span>
          </button>
        </div>
      </div>
      <!-- 移动端 -->
      <div class="toolbar-mobile">
        <div class="title-group">
          <div class="page-icon bg-slate-500"><i class="fas fa-code text-white"></i></div>
          <div class="min-w-0">
            <h1 class="title-text">{{ $t('原始配置') }}</h1>
            <p class="text-xs text-slate-500 truncate">{{ $t('查看和编辑 JSON 配置') }}</p>
          </div>
        </div>
        <div class="action-group-sm">
          <button class="btn btn-secondary btn-square" :disabled="loading || saving || !portal.hasPerm('GET /api/caddy/config')" :title="$t('刷新')" @click="loadConfig()"><i class="fas fa-rotate text-sm"></i></button>
          <button v-if="portal.hasPerm('POST /api/caddy/config')" :disabled="!canSave" :aria-busy="saving" class="btn btn-secondary btn-square" :title="$t('提交配置')" @click="saveConfig()"><i :class="saving ? 'fas fa-spinner fa-spin text-sm' : 'fas fa-cloud-arrow-up text-sm'"></i></button>
        </div>
      </div>
    </div>

    <div v-if="loading" class="card-body"><div class="empty-state"><div class="spinner-lg"></div><p class="text-slate-500">{{ $t('加载中...') }}</p></div></div>
    <div v-else-if="!loaded" class="card-body">
      <div class="empty-state">
        <p class="text-slate-500">{{ portal.hasPerm('GET /api/caddy/config') ? $t('加载失败，请刷新重试') : $t('没有权限') }}</p>
      </div>
    </div>
    <div v-else class="card-body flex-1 min-h-0 flex flex-col gap-3">
      <CodeEditor v-model="raw" language="json" :label="$t('原始配置')" height="100%" class="flex-1 min-h-[16rem]" :disabled="saving" />
      <p class="text-xs text-slate-400 flex items-start gap-1 flex-shrink-0">
        <i class="fas fa-circle-info mt-0.5 flex-shrink-0"></i>
        <span>{{ $t('提交将通过') }} <code class="px-1 bg-slate-100 rounded">POST /load</code> {{ $t('整体替换 Caddy 运行配置，操作前请确保已备份。') }}</span>
      </p>
    </div>
  </div>
</template>
