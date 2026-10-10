<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { useConfigStore, usePortal } from '@/stores'

@Component
class ConfigLayout extends Vue {
    portal = usePortal()
    config = useConfigStore()

    // 查看权限：GET 与 PUT 任一即可，保证只读成员也能查看配置
    get canView() {
        return this.portal.hasPerm('GET /api/system/config') || this.portal.hasPerm('PUT /api/system/config')
    }

    // 编辑权限：仅授权 PUT 的成员可保存
    get canUpdate() {
        return this.portal.hasPerm('PUT /api/system/config')
    }

    // 刷新/关闭页面时提醒未保存改动（浏览器原生弹窗）
    onBeforeUnload = (event: BeforeUnloadEvent) => {
        if (!this.canUpdate || !this.config.hasUnsaved()) return
        event.preventDefault()
        event.returnValue = ''
    }

    // ─── 方法 ───

    async reloadConfig() {
        try {
            await this.config.load(true)
            this.portal.showNotification('success', this.$t('配置已重载'))
        } catch {
            this.portal.showNotification('error', this.config.error || this.$t('配置重载失败'))
        }
    }

    async saveConfig() {
        if (this.config.saving) return
        try {
            await this.config.saveAll()
            await this.portal.refresh()
            this.portal.showNotification('success', this.$t('配置已保存，监听地址变更需重启生效'))
        } catch {
            this.portal.showNotification('error', this.config.error || this.$t('保存配置失败'))
        }
    }

    // ─── 生命周期 ───
    mounted() {
        // 草稿在 store 中常驻，仅在首次进入时加载，避免站内往返导致未保存改动被覆盖
        if (this.canView && !this.config.loaded) this.config.load()
        window.addEventListener('beforeunload', this.onBeforeUnload)
    }

    unmounted() {
        window.removeEventListener('beforeunload', this.onBeforeUnload)
    }
}

export default toNative(ConfigLayout)
</script>

<template>
  <div class="page">
    <!-- Toolbar -->
    <div class="page-toolbar">
      <!-- 桌面端 -->
      <div class="toolbar-desktop">
        <div class="flex items-center gap-3">
          <div class="page-icon bg-indigo-500">
            <i class="fas fa-gear text-white"></i>
          </div>
          <div>
            <h1 class="title-text">{{ $t('系统配置') }}</h1>
            <p class="text-xs text-slate-500">{{ $t('按分组管理服务器、认证、网关与容器参数') }}</p>
          </div>
        </div>
        <div class="action-group">
          <button type="button" class="btn btn-secondary" :disabled="config.loading || config.saving" @click="reloadConfig">
            <i :class="config.loading ? 'fas fa-spinner fa-spin' : 'fas fa-rotate'"></i>{{ $t('重载') }}
          </button>
          <button v-if="canUpdate" type="submit" form="config-form" class="btn btn-indigo rounded-xl whitespace-nowrap" :disabled="config.saving || config.loading">
            <i v-if="config.saving" class="fas fa-spinner fa-spin"></i>
            <i v-else class="fas fa-save"></i>
            <span>{{ config.saving ? $t('保存中...') : $t('保存配置') }}</span>
          </button>
        </div>
      </div>
      <!-- 移动端 -->
      <div class="toolbar-mobile">
        <div class="title-group">
          <div class="page-icon bg-indigo-500">
            <i class="fas fa-gear text-white"></i>
          </div>
          <div class="min-w-0">
            <h1 class="title-text">{{ $t('系统配置') }}</h1>
            <p class="text-xs text-slate-500 truncate">{{ $t('服务器、认证、网关与容器参数') }}</p>
          </div>
        </div>
        <div class="flex items-center gap-2">
          <button type="button" class="btn btn-secondary btn-square" :title="$t('重载')" :disabled="config.loading || config.saving" @click="reloadConfig">
            <i :class="config.loading ? 'fas fa-spinner fa-spin text-sm' : 'fas fa-rotate text-sm'"></i>
          </button>
          <button v-if="canUpdate" type="submit" form="config-form" class="btn btn-indigo btn-square" :title="$t('保存配置')" :disabled="config.saving || config.loading">
            <i v-if="config.saving" class="fas fa-spinner fa-spin text-sm"></i>
            <i v-else class="fas fa-save text-sm"></i>
          </button>
        </div>
      </div>
    </div>

    <!-- 无查看权限 -->
    <div v-if="!canView" class="card-body">
      <div class="empty-state">
        <div class="empty-state-icon">
          <i class="fas fa-lock text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">{{ $t('无权限查看系统配置') }}</p>
        <p class="text-sm text-slate-400">{{ $t('请联系管理员调整账号权限') }}</p>
      </div>
    </div>

    <!-- 加载中 -->
    <div v-else-if="config.loading && !config.loaded" class="card-body">
      <div class="empty-state">
        <div class="spinner-lg"></div>
        <p class="text-slate-500">{{ $t('加载中...') }}</p>
      </div>
    </div>

    <!-- 加载失败 -->
    <div v-else-if="config.error && !config.loaded" class="card-body">
      <div class="empty-state">
        <div class="empty-state-icon">
          <i class="fas fa-triangle-exclamation text-4xl text-red-400"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">{{ $t('加载系统配置失败') }}</p>
        <p class="text-sm text-slate-400 mb-4">{{ $t('请检查网络或权限后重试') }}</p>
        <button type="button" class="btn btn-secondary" @click="config.load()">
          <i class="fas fa-rotate"></i>{{ $t('重新加载') }}
        </button>
      </div>
    </div>

    <!-- 配置分组内容（分组切换在侧边栏子菜单） -->
    <template v-else>
      <!-- 只读提示：仅有查看权限时表单不可编辑 -->
      <div v-if="!canUpdate" class="card-body pb-0">
        <div class="flex items-center gap-2 px-3 py-2.5 rounded-xl bg-amber-50 border border-amber-200 text-amber-700 text-sm">
          <i class="fas fa-eye"></i>
          <span>{{ $t('当前账号仅有查看权限，配置项不可修改') }}</span>
        </div>
      </div>
      <form id="config-form" class="card-body" @submit.prevent="saveConfig">
        <fieldset :disabled="!canUpdate" class="space-y-6 min-w-0">
          <router-view />
        </fieldset>
      </form>
    </template>
  </div>
</template>
