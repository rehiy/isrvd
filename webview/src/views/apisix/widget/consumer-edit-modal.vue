<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

import api from '@/service/api'
import type { ApisixConsumer, ApisixCreateConsumerRequest, ApisixUpdateConsumerRequest } from '@/service/types'

import BaseModal from '@/component/modal.vue'

const TYPE_DEFAULTS: Record<string, string | number | boolean | unknown[] | Record<string, unknown>> = { string: '', integer: 0, number: 0, boolean: false, array: [], object: {} }

const defaultFormData = () => ({
    username: '',
    desc: '',
    plugins: {} as Record<string, unknown>,
    pluginsJson: '{}',
    pluginsJsonError: '',
})

@Component({
    expose: ['show'],
    components: { BaseModal },
    emits: ['success']
})
class ConsumerEditModal extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    // ─── 数据属性 ───
    isOpen = false
    modalLoading = false
    isEditMode = false
    showPluginPanel = false
    pluginSearchKeyword = ''
    availablePlugins: Record<string, { schema: Record<string, unknown> }> = {}

    formData = defaultFormData()

    // ─── 计算属性 ───
    get currentPluginNames() { return Object.keys(this.formData.plugins || {}) }

    get filteredAvailablePlugins() {
        const kw = this.pluginSearchKeyword.toLowerCase()
        const all = Object.keys(this.availablePlugins)
        return kw ? all.filter(n => n.toLowerCase().includes(kw)) : all
    }

    // ─── 方法 ───
    resetForm() {
        Object.assign(this.formData, defaultFormData())
        this.showPluginPanel = false
        this.pluginSearchKeyword = ''
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
            const plugins = consumer.plugins || {}
            this.formData = {
                username: consumer.username,
                desc: consumer.desc || '',
                plugins,
                pluginsJson: JSON.stringify(plugins, null, 2),
                pluginsJsonError: '',
            }
        } else {
            this.isEditMode = false
            this.resetForm()
        }
        this.isOpen = true
    }

    syncPluginsFromJson() {
        try {
            this.formData.plugins = JSON.parse(this.formData.pluginsJson || '{}')
            this.formData.pluginsJsonError = ''
        } catch (e: unknown) {
            this.formData.pluginsJsonError = 'JSON 格式错误: ' + (e instanceof Error ? e.message : String(e))
        }
    }

    removePlugin(name: string) {
        const p = { ...this.formData.plugins }
        delete p[name]
        this.formData.plugins = p
        this.formData.pluginsJson = JSON.stringify(p, null, 2)
    }

    buildPluginDefault(schema: { properties?: Record<string, { type: string; default?: unknown }>; required?: string[] }) {
        if (!schema?.properties) return {}
        const required = new Set(schema.required || [])
        const result: Record<string, unknown> = {}
        for (const [key, def] of Object.entries(schema.properties)) {
            if (key === 'disable') continue
            if (required.has(key) || def.default !== undefined) {
                result[key] = def.default !== undefined ? def.default : (TYPE_DEFAULTS[def.type] ?? null)
            }
        }
        return result
    }

    addPresetPlugin(name: string) {
        if (this.formData.plugins[name] !== undefined) {
            return this.actions.showNotification('warning', `插件 ${name} 已存在`)
        }
        const p = { ...this.formData.plugins, [name]: this.buildPluginDefault(this.availablePlugins[name]?.schema) }
        this.formData.plugins = p
        this.formData.pluginsJson = JSON.stringify(p, null, 2)
        this.showPluginPanel = false
        this.pluginSearchKeyword = ''
    }

    async handleConfirm() {
        if (!this.formData.username) {
            this.actions.showNotification('error', '用户名不能为空')
            return
        }
        if (this.formData.pluginsJsonError) {
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

      <!-- 插件配置 -->
      <div class="flex items-center justify-between">
        <div>
          <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">插件配置</label>
          <p class="text-xs text-slate-400 mt-1">可直接编辑 JSON，或从插件列表添加</p>
        </div>
        <button @click="showPluginPanel = !showPluginPanel" :class="['px-3 py-1.5 text-xs rounded-lg border transition-colors', showPluginPanel ? 'border-violet-300 text-violet-600 bg-violet-50' : 'border-slate-200 text-slate-600 hover:border-slate-300 hover:bg-slate-50']"><i class="fas fa-puzzle-piece mr-1"></i>添加插件</button>
      </div>

      <!-- 插件选择面板 -->
      <div v-if="showPluginPanel" class="rounded-lg border border-slate-200 overflow-hidden mt-3">
        <div class="px-3 py-2 bg-slate-50 border-b border-slate-100">
          <div class="relative">
            <i class="fas fa-search absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-400 text-xs pointer-events-none"></i>
            <input v-model="pluginSearchKeyword" type="text" placeholder="搜索插件..." class="w-full pl-7 pr-3 py-1.5 text-xs bg-white border border-slate-200 rounded-md focus:outline-none focus:ring-1 focus:ring-violet-400 focus:border-violet-300" />
          </div>
        </div>
        <div class="max-h-44 overflow-y-auto p-2 grid grid-cols-2 md:grid-cols-3 gap-1">
          <button
            v-for="name in filteredAvailablePlugins"
            :key="name"
            :disabled="formData.plugins[name] !== undefined"
            :class="['px-2.5 py-1.5 text-xs rounded-lg text-left truncate transition-colors border', formData.plugins[name] !== undefined ? 'bg-violet-50 text-violet-500 border-violet-100 cursor-default' : 'bg-white hover:bg-slate-50 text-slate-700 border-slate-200 hover:border-slate-300']"
            @click="addPresetPlugin(name)"
          ><i v-if="formData.plugins[name] !== undefined" class="fas fa-check text-[9px] mr-1"></i>{{ name }}</button>
        </div>
      </div>

      <!-- 已添加插件 tags -->
      <div v-if="currentPluginNames.length > 0" class="flex flex-wrap gap-1 mt-3">
        <span v-for="name in currentPluginNames" :key="name" class="inline-flex items-center gap-1 px-2 py-0.5 bg-violet-50 text-violet-700 rounded text-xs">{{ name }}<button @click="removePlugin(name)" class="hover:text-red-500 transition-colors"><i class="fas fa-xmark text-[10px]"></i></button></span>
      </div>

      <!-- JSON 编辑器 -->
      <textarea v-model="formData.pluginsJson" @blur="syncPluginsFromJson" rows="8" :class="['input font-mono text-sm mt-3', formData.pluginsJsonError ? 'border-red-300 bg-red-50' : '']" placeholder='{"key-auth": {"key": "your-api-key"}}'></textarea>
      <p v-if="formData.pluginsJsonError" class="text-xs text-red-500 mt-1">{{ formData.pluginsJsonError }}</p>
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
