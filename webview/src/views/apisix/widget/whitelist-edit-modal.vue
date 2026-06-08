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
    consumerName: '',
    keyAuthHeader: 'apikey',
    keyAuthQuery: 'apikey',
    hideCredentials: false,
})

interface WhitelistConsumerEntry {
    username: string
    key: string
}

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
    whitelistEntries: WhitelistConsumerEntry[] = []
    formData = defaultFormData()

    // ─── 计算属性 ───
    get whitelistConsumers() {
        return this.whitelistEntries.map(entry => entry.username)
    }

    get existingConsumerNames() {
        return new Set(this.consumers.map(consumer => consumer.username))
    }

    get missingWhitelistEntries() {
        return this.whitelistEntries.filter(entry => !this.existingConsumerNames.has(entry.username))
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
    routeLabel(route: ApisixRoute) {
        return route.name || route.id || '未命名路由'
    }

    resetForm() {
        Object.assign(this.formData, defaultFormData())
        this.whitelistEntries = []
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

    isExistingConsumer(username: string) {
        return this.existingConsumerNames.has(username)
    }

    addWhitelistConsumer() {
        const names = this.formData.consumerName
            .split(/[,\n]/)
            .map(name => name.trim())
            .filter(Boolean)
        if (names.length === 0) return this.portal.showNotification('error', '请输入 Consumer 用户名')

        let added = 0
        names.forEach(username => {
            if (this.whitelistEntries.some(entry => entry.username === username)) return
            this.whitelistEntries.push({ username, key: '' })
            added += 1
        })
        if (added === 0) return this.portal.showNotification('error', '用户已在白名单列表中')
        this.formData.consumerName = ''
    }

    updateConsumerName(value: string | string[]) {
        this.formData.consumerName = Array.isArray(value) ? value.join(',') : value
    }

    filteredConsumers(query: string) {
        return this.consumers
            .filter(consumer => !this.whitelistEntries.some(entry => entry.username === consumer.username))
            .filter(consumer => {
                if (!query) return true
                return consumer.username.toLowerCase().includes(query) || (consumer.desc || '').toLowerCase().includes(query)
            })
            .slice(0, 8)
    }

    useConsumerSuggestion(username: string, select: (value: string) => void) {
        select(username)
        this.formData.consumerName = username
        this.addWhitelistConsumer()
    }

    removeWhitelistConsumer(username: string) {
        this.whitelistEntries = this.whitelistEntries.filter(entry => entry.username !== username)
    }

    async createMissingConsumers() {
        const entries = this.missingWhitelistEntries
        for (let index = 0; index < entries.length; index += 1) {
            const entry = entries[index]
            const username = entry.username.trim()
            const key = entry.key.trim()
            try {
                const created = await api.apisixConsumerCreate({
                    username,
                    plugins: {
                        'key-auth': { key },
                    },
                })
                if (created.payload) this.consumers.push(created.payload)
                this.portal.showNotification('success', `Consumer ${username} 创建成功（${index + 1}/${entries.length}）`)
            } catch (e: unknown) {
                const message = e instanceof Error ? e.message : '创建失败'
                const err = new Error(`Consumer ${username} 创建失败：${message}`)
                ;(err as Error & { cause: unknown }).cause = e
                throw err
            }
        }
    }

    async handleConfirm() {
        if (!this.formData.routeId) return this.portal.showNotification('error', '请选择要配置白名单的路由')
        if (this.whitelistConsumers.length === 0) return this.portal.showNotification('error', '白名单用户不能为空')
        const missingKeyEntry = this.missingWhitelistEntries.find(entry => !entry.key.trim())
        if (missingKeyEntry) return this.portal.showNotification('error', `Consumer ${missingKeyEntry.username} 不存在，请填写 key-auth key`)
        if (!this.formData.keyAuthHeader.trim()) return this.portal.showNotification('error', 'key-auth 请求头名称不能为空')

        this.modalLoading = true
        try {
            await this.createMissingConsumers()
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
          <p class="mt-1 text-xs text-slate-400">配置路由认证插件参数；新增 Consumer 会使用下方输入的 key-auth.key 创建</p>
        </div>
        <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
          <div>
            <label class="form-label">请求头名称 <span class="text-red-500">*</span></label>
            <input v-model="formData.keyAuthHeader" type="text" class="input" placeholder="例如：apikey" />
            <p class="mt-1 text-xs text-slate-400">客户端通过该 Header 传递 Consumer API Key</p>
          </div>
          <div>
            <label class="form-label">查询参数名称</label>
            <input v-model="formData.keyAuthQuery" type="text" class="input" placeholder="例如：apikey（可选）" />
            <p class="mt-1 text-xs text-slate-400">留空表示不启用 Query 参数取 key</p>
          </div>
        </div>
        <ToggleCard v-model="formData.hideCredentials" label="隐藏认证凭据" desc="开启后转发到上游前移除请求中的认证凭据" />
      </div>

      <div class="space-y-3">
        <div>
          <label class="form-label">白名单用户 <span class="text-red-500">*</span></label>
          <div class="flex gap-2">
            <div class="relative flex-1">
              <Combobox
                :model-value="formData.consumerName"
                placeholder="输入 Consumer 用户名，支持英文逗号分隔"
                search-placeholder="搜索 Consumer"
                max-height="320px"
                @update:model-value="updateConsumerName"
              >
                <template #hint-extra="{ query }">
                  <span class="text-xs text-slate-400">{{ filteredConsumers(query.toLowerCase()).length }} 个匹配</span>
                </template>

                <template #default="{ query, select }">
                  <div class="select-list p-2">
                    <button
                      v-for="consumer in filteredConsumers(query.toLowerCase())"
                      :key="consumer.username"
                      type="button"
                      class="flex w-full items-center gap-2.5 rounded-lg border border-transparent px-2.5 py-2 text-left transition-all duration-150 hover:bg-slate-50"
                      @click="useConsumerSuggestion(consumer.username, select)"
                    >
                      <span class="row-icon bg-violet-50 text-violet-600">
                        <i class="fas fa-user text-xs"></i>
                      </span>
                      <span class="min-w-0 flex-1">
                        <span class="block truncate text-sm font-medium text-slate-700">{{ consumer.username }}</span>
                        <span v-if="consumer.desc" class="mt-0.5 block truncate text-xs text-slate-400">{{ consumer.desc }}</span>
                      </span>
                      <span class="rounded-lg bg-emerald-50 px-1.5 py-0.5 text-xs text-emerald-700">已存在</span>
                    </button>
                  </div>
                </template>

                <template #empty="{ query }">
                  <div v-if="filteredConsumers(query.toLowerCase()).length === 0" class="py-8 text-center">
                    <i class="fas fa-search text-2xl text-slate-300 mb-2"></i>
                    <p class="text-sm text-slate-400">{{ consumers.length === 0 ? '暂无 Consumer' : '无匹配 Consumer，可直接输入新用户名' }}</p>
                  </div>
                </template>
              </Combobox>
            </div>
            <button type="button" class="btn btn-amber h-[46px] flex-shrink-0" @click="addWhitelistConsumer">
              <i class="fas fa-plus"></i>添加
            </button>
          </div>
          <p class="text-xs text-slate-400 mt-1">已存在的 Consumer 会直接加入白名单；不存在的 Consumer 需要填写 key-auth key，提交时会逐个创建并提示结果</p>
        </div>

        <div v-if="whitelistEntries.length > 0" class="overflow-hidden rounded-xl border border-slate-200 bg-white">
          <div
            v-for="entry in whitelistEntries"
            :key="entry.username"
            class="border-b border-slate-100 px-3 py-2.5 last:border-b-0"
          >
            <div class="flex items-center justify-between gap-3">
              <div class="min-w-0">
                <div class="flex items-center gap-2 min-w-0">
                  <span class="font-medium text-slate-700 truncate">{{ entry.username }}</span>
                  <span v-if="isExistingConsumer(entry.username)" class="inline-flex items-center rounded-lg bg-emerald-50 px-1.5 py-0.5 text-xs text-emerald-700">已存在</span>
                  <span v-else class="inline-flex items-center rounded-lg bg-amber-50 px-1.5 py-0.5 text-xs text-amber-700">待创建</span>
                </div>
                <p class="mt-0.5 text-xs text-slate-400">
                  {{ isExistingConsumer(entry.username) ? '保存后加入路由白名单' : '该 Consumer 不存在，提交时将使用下方 key 先创建' }}
                </p>
              </div>
              <button type="button" class="btn-icon btn-icon-red flex-shrink-0" title="移除" @click="removeWhitelistConsumer(entry.username)">
                <i class="fas fa-xmark text-xs"></i>
              </button>
            </div>
            <div v-if="!isExistingConsumer(entry.username)" class="mt-2">
              <label class="form-label">key-auth key <span class="text-red-500">*</span></label>
              <input v-model="entry.key" type="text" class="input" placeholder="请输入该 Consumer 的 API Key" />
            </div>
          </div>
        </div>

        <div v-else class="rounded-lg border border-dashed border-slate-200 bg-slate-50/70 px-3 py-3 text-center text-xs text-slate-400">
          暂未添加白名单用户
        </div>
      </div>
    </div>

    <template #confirm-text>
      确认配置
    </template>
  </BaseModal>
</template>
