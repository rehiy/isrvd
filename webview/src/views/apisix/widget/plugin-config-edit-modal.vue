<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

import api from '@/service/api'
import type {
    ApisixCreatePluginConfigRequest,
    ApisixPluginConfig,
    ApisixRoute,
    ApisixUpdatePluginConfigRequest
} from '@/service/types'

import BaseModal from '@/component/modal.vue'
import PluginConfigPanel from './plugin-config-panel.vue'

const defaultFormData = () => ({
    id: '',
    desc: '',
    plugins: {} as Record<string, unknown>
})

@Component({
    expose: ['show'],
    components: { BaseModal, PluginConfigPanel },
    emits: ['success']
})
class PluginConfigEditModal extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    isOpen = false
    modalLoading = false
    isEditMode = false
    formData = defaultFormData()
    availablePlugins: Record<string, { schema: Record<string, unknown> }> = {}
    routes: ApisixRoute[] = []

    resetForm() {
        this.formData = defaultFormData()
    }

    async show(config: ApisixPluginConfig | null = null) {
        await this.loadResources()
        if (config) {
            this.isEditMode = true
            this.formData = {
                id: config.id || '',
                desc: config.desc || '',
                plugins: { ...(config.plugins || {}) }
            }
        } else {
            this.isEditMode = false
            this.resetForm()
        }
        this.isOpen = true
    }

    async loadResources() {
        try {
            const [pluginsRes, routesRes] = await Promise.all([
                api.apisixListPlugins(),
                api.apisixListRoutes()
            ])
            this.availablePlugins = pluginsRes.payload || {}
            this.routes = routesRes.payload || []
        } catch {
            this.availablePlugins = {}
            this.routes = []
        }
    }

    updatePlugins(plugins: Record<string, unknown>) {
        this.formData.plugins = plugins
    }

    buildPayload(): ApisixCreatePluginConfigRequest | ApisixUpdatePluginConfigRequest {
        const payload: ApisixCreatePluginConfigRequest = {
            desc: this.formData.desc.trim(),
            plugins: this.formData.plugins,
        }
        // 只在创建时且用户手动填写了 ID 才传入
        if (!this.isEditMode && this.formData.id.trim()) {
            payload.id = this.formData.id.trim()
        }
        return payload
    }

    async handleConfirm() {
        if (Object.keys(this.formData.plugins || {}).length === 0) {
            this.actions.showNotification('error', '至少需要配置一个插件')
            return
        }

        this.modalLoading = true
        try {
            const payload = this.buildPayload()
            if (this.isEditMode) {
                await api.apisixUpdatePluginConfig(this.formData.id, payload)
            } else {
                await api.apisixCreatePluginConfig(payload)
            }
            this.isOpen = false
            this.$emit('success')
        } catch (e: unknown) {
            this.actions.showNotification('error', (e instanceof Error ? e.message : '') || '操作失败')
        }
        this.modalLoading = false
    }
}

export default toNative(PluginConfigEditModal)
</script>

<template>
  <BaseModal
    v-model="isOpen"
    :title="isEditMode ? '编辑插件配置' : '创建插件配置'"
    :loading="modalLoading"
  >
    <div class="max-w-3xl space-y-4 p-1">
      <div>
        <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">配置 ID</label>
        <input v-model="formData.id" type="text" class="input" :disabled="isEditMode" placeholder="留空由 APISIX 自动生成" />
        <p class="text-xs text-slate-400 mt-1">创建时可选；编辑时不能修改 ID。</p>
      </div>
      <div>
        <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">描述</label>
        <textarea v-model="formData.desc" rows="2" class="input resize-none" placeholder="插件配置描述"></textarea>
      </div>

      <PluginConfigPanel
        :plugins="formData.plugins"
        :available-plugins="availablePlugins"
        :show-import="true"
        :routes="routes"
        @update:plugins="updatePlugins"
      />
    </div>

    <template #footer>
      <div class="flex justify-end gap-2">
        <button @click="isOpen = false" class="px-4 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-200 rounded-lg hover:bg-slate-50">取消</button>
        <button @click="handleConfirm()" :disabled="modalLoading" class="px-4 py-2 text-sm font-medium text-white bg-rose-500 rounded-lg hover:bg-rose-600 disabled:opacity-50 shadow-sm">
          <i v-if="modalLoading" class="fas fa-spinner fa-spin mr-1"></i>{{ isEditMode ? '保存' : '创建' }}
        </button>
      </div>
    </template>
  </BaseModal>
</template>
