<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { ApisixConsumer, ApisixConsumerCreate, ApisixConsumerUpdate } from '@/service/types'

import BaseModal from '@/component/modal.vue'

import PluginConfigPanel from './plugin-config-panel.vue'

const defaultFormData = () => ({
    username: '',
    desc: '',
    plugins: {} as Record<string, unknown>,
})

@Component({
    expose: ['show'],
    components: { BaseModal, PluginConfigPanel },
    emits: ['success']
})
class ConsumerEditModal extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    isOpen = false
    modalLoading = false
    isEditMode = false
    availablePlugins: Record<string, { schema: Record<string, unknown> }> = {}
    declare $refs: { pluginPanel: InstanceType<typeof PluginConfigPanel> }

    formData = defaultFormData()

    // ─── 方法 ───
    resetForm() {
        Object.assign(this.formData, defaultFormData())
    }

    async loadPlugins() {
        try {
            const pl = await api.apisixPluginList()
            this.availablePlugins = pl.payload || {}
        } catch {}
    }

    async show(consumer: ApisixConsumer | null = null) {
        await this.loadPlugins()
        if (consumer) {
            this.isEditMode = true
            this.formData = {
                username: consumer.username,
                desc: consumer.desc || '',
                plugins: consumer.plugins || {},
            }
        } else {
            this.isEditMode = false
            this.resetForm()
        }
        this.isOpen = true
    }

    onPluginsUpdate(plugins: Record<string, unknown>) {
        this.formData.plugins = plugins
    }

    async handleConfirm() {
        if (!this.formData.username) {
            this.portal.showNotification('error', '消费者名称不能为空')
            return
        }
        if (this.$refs.pluginPanel?.pluginsJsonError) {
            this.portal.showNotification('error', '请修正 Plugin JSON 格式错误')
            return
        }
        this.modalLoading = true
        try {
            const payload: ApisixConsumerUpdate = {
                desc: this.formData.desc,
                plugins: this.formData.plugins,
            }
            if (this.isEditMode) {
                await api.apisixConsumerUpdate(this.formData.username, payload)
            } else {
await api.apisixConsumerCreate({ username: this.formData.username, desc: payload.desc, plugins: payload.plugins } as ApisixConsumerCreate)
            }
            this.isOpen = false
            this.$emit('success')
        } catch (e: unknown) {
            this.portal.showNotification('error', (e instanceof Error ? e.message : '') || '操作失败')
        } finally {
            this.modalLoading = false
        }
    }
}

export default toNative(ConsumerEditModal)
</script>

<template>
  <BaseModal v-model="isOpen" :title="isEditMode ? '编辑消费者' : '新建消费者'" :loading="modalLoading" confirm-class="btn-violet" @confirm="handleConfirm">
    <div class="max-w-3xl space-y-4 p-1">
      <div class="space-y-3">
        <div>
          <label class="form-label">名称 <span class="text-red-500">*</span></label>
          <input v-model="formData.username" type="text" :disabled="isEditMode" class="input" :class="{ 'disabled:bg-slate-50 disabled:text-slate-500': isEditMode }" placeholder="请输入消费者名称" />
        </div>
        <div>
          <label class="form-label">描述</label>
          <textarea v-model="formData.desc" rows="3" class="input" placeholder="请输入消费者描述（可选）"></textarea>
        </div>
      </div>

      <PluginConfigPanel ref="pluginPanel" :plugins="formData.plugins" :available-plugins="availablePlugins" @update:plugins="onPluginsUpdate" />
    </div>

    <template #confirm-text>
      确认{{ isEditMode ? '更新' : '新建' }}
    </template>
  </BaseModal>
</template>
