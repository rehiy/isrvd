<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

import api from '@/service/api'
import type { ApisixConsumer, ApisixCreateConsumerRequest, ApisixUpdateConsumerRequest } from '@/service/types'

import BaseModal from '@/component/modal.vue'
import PluginConfigPanel from '@/views/apisix/widget/plugin-config-panel.vue'

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
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

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
            const pl = await api.apisixListPlugins()
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
            this.actions.showNotification('error', '用户名不能为空')
            return
        }
        if (this.$refs.pluginPanel?.pluginsJsonError) {
            this.actions.showNotification('error', '请修正 Plugin JSON 格式错误')
            return
        }
        this.modalLoading = true
        try {
            const payload: ApisixUpdateConsumerRequest = {
                desc: this.formData.desc,
                plugins: this.formData.plugins,
            }
            if (this.isEditMode) {
                await api.apisixUpdateConsumer(this.formData.username, payload)
            } else {
                await api.apisixCreateConsumer({ username: this.formData.username, desc: payload.desc, plugins: payload.plugins } as ApisixCreateConsumerRequest)
            }
            this.isOpen = false
            this.$emit('success')
        } catch (e: unknown) {
            this.actions.showNotification('error', (e instanceof Error ? e.message : '') || '操作失败')
        }
        this.modalLoading = false
    }
}

export default toNative(ConsumerEditModal)
</script>

<template>
  <BaseModal
    v-model="isOpen"
    :title="isEditMode ? '编辑用户' : '创建用户'"
    :loading="modalLoading"
  >
    <div class="max-w-3xl space-y-4 p-1">
      <div class="space-y-3">
        <div>
          <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">用户名 <span class="text-red-500">*</span></label>
          <input v-model="formData.username" type="text" :disabled="isEditMode" class="input" :class="{ 'disabled:bg-slate-50 disabled:text-slate-500': isEditMode }" placeholder="输入用户名" />
        </div>
        <div>
          <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">描述</label>
          <textarea v-model="formData.desc" rows="3" class="input" placeholder="用户描述"></textarea>
        </div>
      </div>

      <PluginConfigPanel
        :plugins="formData.plugins"
        :available-plugins="availablePlugins"
        @update:plugins="onPluginsUpdate"
        ref="pluginPanel"
      />
    </div>

    <template #footer>
      <div class="flex justify-end gap-2">
        <button @click="isOpen = false" class="px-4 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-200 rounded-lg hover:bg-slate-50">取消</button>
        <button @click="handleConfirm()" :disabled="modalLoading" class="px-4 py-2 text-sm font-medium text-white bg-violet-500 rounded-lg hover:bg-violet-600 disabled:opacity-50 shadow-sm">
          <i v-if="modalLoading" class="fas fa-spinner fa-spin mr-1"></i>{{ isEditMode ? '保存' : '创建' }}
        </button>
      </div>
    </template>
  </BaseModal>
</template>
