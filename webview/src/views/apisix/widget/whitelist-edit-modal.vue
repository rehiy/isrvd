<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { ApisixConsumer, ApisixKeyAuthConfig, ApisixRoute } from '@/service/types'

import BaseModal from '@/component/modal.vue'
import ToggleCard from '@/component/toggle-card.vue'
import Combobox from '@/component/combobox.vue'

const defaultFormData = () => ({
    routeId: '',
    keyAuthHeader: 'token',
    keyAuthQuery: 'token',
    hideCredentials: false,
})

@Component({
    expose: ['show'],
    components: { BaseModal, ToggleCard, Combobox },
    emits: ['success']
})
class WhitelistEditModal extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    isOpen = false
    modalLoading = false
    routes: ApisixRoute[] = []
    consumers: ApisixConsumer[] = []
    whitelistConsumers: string[] = []
    formData = defaultFormData()

    // ─── 计算属性 ───
    /** 只展示已有 key-auth 插件的 Consumer */
    get keyAuthConsumers() {
        return this.consumers.filter(c => c.plugins?.['key-auth'])
    }

    get selectableRoutes() {
        return this.routes.filter(route => {
            if (!route.id) return false
            const plugins = route.plugins || {}
            return !plugins['consumer-restriction']
        })
    }

    get selectedRoute() {
        return this.selectableRoutes.find(route => route.id === this.formData.routeId)
    }

    get keyAuthConfig(): ApisixKeyAuthConfig {
        const config: ApisixKeyAuthConfig = {
            header: this.formData.keyAuthHeader.trim(),
            hide_credentials: this.formData.hideCredentials,
        }
        const query = this.formData.keyAuthQuery.trim()
        if (query) config.query = query
        return config
    }

    // ─── 方法 ───
    resetForm() {
        Object.assign(this.formData, defaultFormData())
        this.whitelistConsumers = []
    }

    async show() {
        this.resetForm()
        this.isOpen = true
        this.modalLoading = true
        try {
            const [routesRes, consumersRes] = await Promise.all([api.apisixRouteList(), api.apisixConsumerList()])
            this.routes = routesRes.payload || []
            this.consumers = consumersRes.payload || []
        } catch {
            this.portal.showNotification('error', '加载路由或消费者列表失败')
        } finally {
            this.modalLoading = false
        }
    }

    filteredConsumers(query: string) {
        return this.keyAuthConsumers
            .filter(consumer => !this.whitelistConsumers.includes(consumer.username))
            .filter(consumer => {
                if (!query) return true
                return consumer.username.toLowerCase().includes(query) || (consumer.desc || '').toLowerCase().includes(query)
            })
            .slice(0, 8)
    }

    consumerTagClass() {
        return 'bg-amber-50 text-amber-800 border border-amber-200'
    }

    updateWhitelistConsumers(value: string | string[]) {
        this.whitelistConsumers = Array.isArray(value) ? value : []
    }

    async handleConfirm() {
        if (!this.formData.routeId) return this.portal.showNotification('error', '请选择要配置白名单的路由')
        if (this.whitelistConsumers.length === 0) return this.portal.showNotification('error', '白名单用户不能为空')
        if (!this.formData.keyAuthHeader.trim()) return this.portal.showNotification('error', 'key-auth 请求头名称不能为空')

        this.modalLoading = true
        try {
            await api.apisixWhitelistCreate({
                route_id: this.formData.routeId,
                consumers: this.whitelistConsumers,
                key_auth: this.keyAuthConfig,
            })
            this.portal.showNotification('success', '白名单配置成功')
            this.isOpen = false
            this.$emit('success')
        } catch (e: unknown) {
            this.portal.showNotification('error', (e instanceof Error ? e.message : '') || '操作失败')
        } finally {
            this.modalLoading = false
        }
    }
}

export default toNative(WhitelistEditModal)
</script>

<template>
  <BaseModal v-model="isOpen" title="配置路由白名单" :loading="modalLoading" confirm-class="btn-amber" @confirm="handleConfirm">
    <div class="max-w-3xl space-y-4 p-1">
      <div>
        <label class="form-label">路由 <span class="text-red-500">*</span></label>
        <select v-model="formData.routeId" class="input">
          <option value="">请选择未配置白名单的路由</option>
          <option v-for="route in selectableRoutes" :key="route.id" :value="route.id">
            {{ route.name || route.id }} - {{ (route.uris?.length ? route.uris.join(', ') : route.uri) || '-' }}
          </option>
        </select>
        <p v-if="selectedRoute" class="text-xs text-slate-400 mt-1">
          保存后会为所选路由配置 key-auth 和 consumer-restriction.whitelist 插件
        </p>
        <p v-else class="text-xs text-slate-400 mt-1">仅展示尚未配置 consumer-restriction 白名单的路由</p>
      </div>

      <div v-if="selectableRoutes.length === 0" class="rounded-lg border border-amber-100 bg-amber-50/50 px-3 py-2 text-xs text-amber-700">
        暂无可配置白名单的路由
      </div>

      <div class="space-y-3">
        <div>
          <label class="form-label">key-auth 插件配置</label>
          <p class="mt-1 text-xs text-slate-400">配置路由认证插件参数</p>
        </div>
        <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
          <div>
            <label class="form-label">请求头名称 <span class="text-red-500">*</span></label>
            <input v-model="formData.keyAuthHeader" type="text" class="input" placeholder="例如：token" />
            <p class="mt-1 text-xs text-slate-400">客户端通过该 Header 传递 Consumer API Key</p>
          </div>
          <div>
            <label class="form-label">查询参数名称</label>
            <input v-model="formData.keyAuthQuery" type="text" class="input" placeholder="例如：token（可选）" />
            <p class="mt-1 text-xs text-slate-400">留空表示不启用 Query 参数取 key</p>
          </div>
        </div>
        <ToggleCard v-model="formData.hideCredentials" label="隐藏认证凭据" desc="开启后转发到上游前移除请求中的认证凭据" />
      </div>

      <div class="space-y-3">
        <div>
          <label class="form-label">白名单用户 <span class="text-red-500">*</span></label>
          <Combobox
            :model-value="whitelistConsumers"
            multiple
            placeholder="搜索并选择 Consumer，可多选"
            search-placeholder="搜索 Consumer"
            max-height="320px"
            :tag-class="consumerTagClass"
            @update:model-value="updateWhitelistConsumers"
          >
            <template #hint-extra="{ query }">
              <span class="text-xs text-slate-400">{{ filteredConsumers(query.toLowerCase()).length }} 个可选</span>
            </template>

            <template #default="{ query, select }">
              <div class="select-list p-2">
                <button
                  v-for="consumer in filteredConsumers(query.toLowerCase())"
                  :key="consumer.username"
                  type="button"
                  class="flex w-full items-center gap-2.5 rounded-lg border border-transparent px-2.5 py-2 text-left transition-all duration-150 hover:bg-slate-50"
                  @click="select(consumer.username)"
                >
                  <span class="row-icon bg-violet-50 text-violet-600">
                    <i class="fas fa-user text-xs"></i>
                  </span>
                  <span class="min-w-0 flex-1">
                    <span class="block truncate text-sm font-medium text-slate-700">{{ consumer.username }}</span>
                    <span v-if="consumer.desc" class="mt-0.5 block truncate text-xs text-slate-400">{{ consumer.desc }}</span>
                  </span>
                  <span class="rounded-lg bg-emerald-50 px-1.5 py-0.5 text-xs text-emerald-700">key-auth</span>
                </button>
              </div>
            </template>

            <template #empty>
              <div v-if="filteredConsumers('').length === 0" class="py-8 text-center">
                <i class="fas fa-search text-2xl text-slate-300 mb-2"></i>
                <p class="text-sm text-slate-400">{{ keyAuthConsumers.length === 0 ? '暂无已配置 key-auth 的 Consumer' : '全部已选' }}</p>
              </div>
            </template>
          </Combobox>
          <p class="text-xs text-slate-400 mt-1">仅展示已配置 key-auth 插件的 Consumer；如需添加新用户，请先在 Consumer 管理中创建</p>
        </div>
      </div>
    </div>

    <template #confirm-text>
      确认配置
    </template>
  </BaseModal>
</template>
