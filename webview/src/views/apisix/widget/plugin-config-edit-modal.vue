<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type {
    ApisixPluginConfigCreate,
    ApisixPluginConfig,
    ApisixPluginConfigUpdate,
    ApisixRoute
} from '@/service/types'

import BaseModal from '@/component/modal.vue'

import { usePortal } from '@/stores'

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
    portal = usePortal()
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
                api.apisixPluginList(),
                api.apisixRouteList()
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

    buildPayload(): ApisixPluginConfigCreate | ApisixPluginConfigUpdate {
        const payload: ApisixPluginConfigCreate = {
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
            this.portal.showNotification('error', '至少需要配置一个插件')
            return
        }

        this.modalLoading = true
        try {
            const payload = this.buildPayload()
            if (this.isEditMode) {
                await api.apisixPluginConfigUpdate(this.formData.id, payload)
            } else {
                await api.apisixPluginConfigCreate(payload)
            }
            this.isOpen = false
            this.$emit('success')
        } catch (e: unknown) {
            this.portal.showNotification('error', (e instanceof Error ? e.message : '') || '操作失败')
        }
        this.modalLoading = false
    }
}

export default toNative(PluginConfigEditModal)
</script>

<template>
  <BaseModal
    v-model="isOpen"
    :title="isEditMode ? '编辑插件配置' : '新建插件配置'"
    :loading="modalLoading"
    confirm-class="btn-rose"
    @confirm="handleConfirm"
  >
    <div class="max-w-3xl space-y-4 p-1">
      <div>
        <label class="form-label">配置 ID</label>
        <input v-model="formData.id" type="text" class="input" :disabled="isEditMode" placeholder="留空由 APISIX 自动生成" />
        <p class="text-xs text-slate-400 mt-1">创建时可选；编辑时不能修改 ID。</p>
      </div>
      <div>
        <label class="form-label">描述</label>
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

    <template #confirm-text>
      确认{{ isEditMode ? '更新' : '新建' }}
    </template>
  </BaseModal>
</template>
