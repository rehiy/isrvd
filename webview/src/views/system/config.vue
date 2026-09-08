<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { useConfigStore, usePortal } from '@/stores'
import type { ConfigGroup } from '@/stores'

@Component
class ConfigLayout extends Vue {
    portal = usePortal()
    config = useConfigStore()

    get canUpdate() {
        return this.portal.hasPerm('PUT /api/system/config')
    }

    get currentGroup(): ConfigGroup {
        return (this.$route.meta.group as ConfigGroup | undefined) || 'service'
    }

    // ─── 方法 ───

    async reloadConfig() {
        try {
            await this.config.load(true)
            this.portal.showNotification('success', '配置已重载')
        } catch {
            this.portal.showNotification('error', this.config.error || '配置重载失败')
        }
    }

    async saveCurrentGroup() {
        if (this.config.saving) return
        try {
            await this.config.saveGroup(this.currentGroup)
            await this.portal.refresh()
            this.portal.showNotification('success', '配置已保存，监听地址变更需重启生效')
        } catch {
            this.portal.showNotification('error', this.config.error || '保存配置失败')
        }
    }

    // ─── 生命周期 ───
    mounted() {
        if (this.canUpdate) this.config.load()
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
            <h1 class="title-text">系统配置</h1>
            <p class="text-xs text-slate-500">按分组管理服务器、认证、网关与容器参数</p>
          </div>
        </div>
        <div class="action-group">
          <button type="button" class="btn btn-secondary" :disabled="config.loading" @click="reloadConfig">
            <i :class="config.loading ? 'fas fa-spinner fa-spin' : 'fas fa-rotate'"></i>重载
          </button>
          <button v-if="canUpdate" type="submit" form="config-form" class="btn btn-indigo rounded-xl whitespace-nowrap" :disabled="config.saving || config.loading">
            <i v-if="config.saving" class="fas fa-spinner fa-spin"></i>
            <i v-else class="fas fa-save"></i>
            <span>{{ config.saving ? '保存中...' : '保存配置' }}</span>
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
            <h1 class="title-text">系统配置</h1>
            <p class="text-xs text-slate-500 truncate">服务器、认证、网关与容器参数</p>
          </div>
        </div>
        <div class="flex items-center gap-2">
          <button type="button" class="btn btn-secondary btn-square" title="重载" :disabled="config.loading" @click="reloadConfig">
            <i :class="config.loading ? 'fas fa-spinner fa-spin text-sm' : 'fas fa-rotate text-sm'"></i>
          </button>
          <button v-if="canUpdate" type="submit" form="config-form" class="btn btn-indigo btn-square" title="保存配置" :disabled="config.saving || config.loading">
            <i v-if="config.saving" class="fas fa-spinner fa-spin text-sm"></i>
            <i v-else class="fas fa-save text-sm"></i>
          </button>
        </div>
      </div>
    </div>

    <!-- 无权限 -->
    <div v-if="!canUpdate" class="card-body">
      <div class="empty-state">
        <div class="empty-state-icon">
          <i class="fas fa-lock text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">无权限修改系统配置</p>
        <p class="text-sm text-slate-400">请联系管理员调整账号权限</p>
      </div>
    </div>

    <!-- 加载中 -->
    <div v-else-if="config.loading && !config.loaded" class="card-body">
      <div class="empty-state">
        <div class="spinner-lg"></div>
        <p class="text-slate-500">加载中...</p>
      </div>
    </div>

    <!-- 加载失败 -->
    <div v-else-if="config.error && !config.loaded" class="card-body">
      <div class="empty-state">
        <div class="empty-state-icon">
          <i class="fas fa-triangle-exclamation text-4xl text-red-400"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">加载系统配置失败</p>
        <p class="text-sm text-slate-400 mb-4">请检查网络或权限后重试</p>
        <button type="button" class="btn btn-secondary" @click="config.load()">
          <i class="fas fa-rotate"></i>重新加载
        </button>
      </div>
    </div>

    <!-- 配置分组内容（分组切换在侧边栏子菜单） -->
    <form v-else id="config-form" class="card-body" @submit.prevent="saveCurrentGroup">
      <router-view />
    </form>
  </div>
</template>
