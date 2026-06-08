<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { ApisixConsumer, ApisixRoute } from '@/service/types'

import PageSearch from '@/component/page-search.vue'
import BaseModal from '@/component/modal.vue'
import Combobox from '@/component/combobox.vue'

import WhitelistEditModal from './widget/whitelist-edit-modal.vue'

// ─── 添加用户 Modal 状态 ───
interface AddUserState {
    open: boolean
    loading: boolean
    route: ApisixRoute | null
    // 选择模式：existing = 已有 Consumer，new = 新建 Consumer
    mode: 'existing' | 'new'
    // 已有用户多选
    selectedUsernames: string[]
    // 新建用户
    newUsername: string
    newKey: string
}

@Component({
    components: { PageSearch, WhitelistEditModal, BaseModal, Combobox }
})
class Whitelist extends Vue {
    portal = usePortal()

    @Ref readonly editModalRef!: InstanceType<typeof WhitelistEditModal>

    // ─── 数据属性 ───
    whitelist: ApisixRoute[] = []
    consumers: ApisixConsumer[] = []
    loading = false
    searchText = ''

    addUser: AddUserState = {
        open: false,
        loading: false,
        route: null,
        mode: 'existing',
        selectedUsernames: [],
        newUsername: '',
        newKey: '',
    }

    // ─── 计算属性 ───
    get filteredWhitelist() {
        const keyword = this.searchText.trim().toLowerCase()
        if (!keyword) return this.whitelist
        return this.whitelist.filter((r: ApisixRoute) =>
            (r.name || '').toLowerCase().includes(keyword) ||
            (r.id || '').toLowerCase().includes(keyword) ||
            (r.consumers || []).some((c: string) => c.toLowerCase().includes(keyword))
        )
    }

    /** 已有 key-auth 的 Consumer */
    get keyAuthConsumers() {
        return this.consumers.filter(c => c.plugins?.['key-auth'])
    }

    /** 当前路由已有的白名单用户名集合 */
    get currentRouteConsumers() {
        return new Set(this.addUser.route?.consumers || [])
    }

    /** 可选的已有 Consumer（排除已在白名单中的 + 已选中的） */
    filteredSelectableConsumers(query: string) {
        const q = query.toLowerCase()
        return this.keyAuthConsumers
            .filter(c => !this.currentRouteConsumers.has(c.username))
            .filter(c => !this.addUser.selectedUsernames.includes(c.username))
            .filter(c => {
                if (!q) return true
                return c.username.toLowerCase().includes(q) || (c.desc || '').toLowerCase().includes(q)
            })
            .slice(0, 8)
    }

    // ─── 方法 ───
    async loadWhitelist() {
        this.loading = true
        try {
            this.whitelist = (await api.apisixWhitelist()).payload || []
        } catch {
            this.portal.showNotification('error', '加载白名单失败')
        } finally {
            this.loading = false
        }
    }

    async loadConsumers() {
        try {
            this.consumers = (await api.apisixConsumerList()).payload || []
        } catch {
            // 静默失败，不影响主流程
        }
    }

    getRouteUri(r: ApisixRoute) {
        return r.uris?.length ? r.uris.join(', ') : (r.uri || '-')
    }

    getRouteHost(r: ApisixRoute) {
        return r.hosts?.length ? r.hosts.join(', ') : (r.host || '*')
    }

    openCreateModal() {
        this.editModalRef?.show()
    }

    openAddUserModal(route: ApisixRoute) {
        Object.assign(this.addUser, {
            open: true,
            loading: false,
            route,
            mode: 'existing',
            selectedUsernames: [],
            newUsername: '',
            newKey: '',
        })
    }

    updateSelectedUsernames(value: string | string[]) {
        this.addUser.selectedUsernames = Array.isArray(value) ? value : []
    }

    consumerTagClass() {
        return 'bg-amber-50 text-amber-800 border border-amber-200'
    }

    switchAddUserMode(mode: 'existing' | 'new') {
        this.addUser.mode = mode
        if (mode === 'existing') {
            this.addUser.newUsername = ''
            this.addUser.newKey = ''
        } else {
            this.addUser.selectedUsernames = []
        }
    }

    async handleAddUser() {
        const route = this.addUser.route
        if (!route?.id) return

        // 获取路由当前的 key-auth 配置（从 plugins 中读取）
        const routeKeyAuth = (route.plugins?.['key-auth'] as Record<string, unknown>) || {}
        const keyAuthConfig = {
            header: (routeKeyAuth['header'] as string) || 'token',
            query: (routeKeyAuth['query'] as string) || undefined,
            hide_credentials: (routeKeyAuth['hide_credentials'] as boolean) || false,
        }

        const currentConsumers = route.consumers || []

        if (this.addUser.mode === 'existing') {
            const usernames = [...new Set(this.addUser.selectedUsernames)]
                .filter(username => !currentConsumers.includes(username))
                .filter(username => this.keyAuthConsumers.some(c => c.username === username))
            if (usernames.length === 0) return this.portal.showNotification('error', '请选择要添加的用户')

            this.addUser.loading = true
            try {
                await api.apisixWhitelistCreate({
                    route_id: route.id,
                    consumers: [...currentConsumers, ...usernames],
                    key_auth: keyAuthConfig,
                })
                this.portal.showNotification('success', `已将 ${usernames.length} 个用户加入白名单`)
                this.addUser.open = false
                this.loadWhitelist()
            } catch (e: unknown) {
                this.portal.showNotification('error', (e instanceof Error ? e.message : '') || '操作失败')
            } finally {
                this.addUser.loading = false
            }
        } else {
            const username = this.addUser.newUsername.trim()
            const key = this.addUser.newKey.trim()
            if (!username) return this.portal.showNotification('error', '请输入用户名')
            if (!key) return this.portal.showNotification('error', '请输入 key-auth key')
            if (currentConsumers.includes(username)) {
                return this.portal.showNotification('error', `用户 "${username}" 已在白名单中`)
            }

            this.addUser.loading = true
            try {
                // 先创建 Consumer
                const created = await api.apisixConsumerCreate({
                    username,
                    plugins: { 'key-auth': { key } },
                })
                if (created.payload) this.consumers.push(created.payload)
                this.portal.showNotification('success', `Consumer "${username}" 创建成功`)

                // 再加入白名单
                await api.apisixWhitelistCreate({
                    route_id: route.id,
                    consumers: [...currentConsumers, username],
                    key_auth: keyAuthConfig,
                })
                this.portal.showNotification('success', `用户 "${username}" 已加入白名单`)
                this.addUser.open = false
                this.loadWhitelist()
            } catch (e: unknown) {
                this.portal.showNotification('error', (e instanceof Error ? e.message : '') || '操作失败')
            } finally {
                this.addUser.loading = false
            }
        }
    }

    revokeConsumer(route: ApisixRoute, consumer: string) {
        const routeId = route.id
        if (!routeId) return
        this.portal.showConfirm({
            title: '撤销白名单',
            message: `确定要将用户 <strong class="text-slate-900">${consumer}</strong> 从路由 <strong class="text-slate-900">${route.name || routeId}</strong> 的白名单中移除吗？`,
            icon: 'fa-user-minus',
            iconColor: 'red',
            confirmText: '确认撤销',
            danger: true,
            onConfirm: async () => {
                await api.apisixWhitelistRevoke({ route_id: routeId, consumer_name: consumer })
                this.portal.showNotification('success', '撤销成功')
                this.loadWhitelist()
            }
        })
    }

    // ─── 生命周期 ───
    mounted() {
        this.loadWhitelist()
        this.loadConsumers()
    }
}

export default toNative(Whitelist)
</script>

<template>
  <div class="card">
    <!-- Toolbar -->
    <div class="card-toolbar">
      <!-- 桌面端 -->
      <div class="hidden md:flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="page-icon bg-amber-500">
            <i class="fas fa-shield-halved text-white"></i>
          </div>
          <div>
            <h1 class="text-lg font-semibold text-slate-800 truncate">白名单管理</h1>
            <p class="text-xs text-slate-500">配置路由级别的 Consumer 访问白名单</p>
          </div>
        </div>
        <div class="flex items-center gap-2 flex-shrink-0">
          <PageSearch v-model="searchText" search-key="apisix-whitelist" placeholder="搜索路由或用户..." focus-color="amber" type-to-search />
          <button class="btn btn-secondary" @click="loadWhitelist()">
            <i class="fas fa-rotate"></i>刷新
          </button>
          <button v-if="portal.hasPerm('POST /api/apisix/whitelist')" class="btn btn-amber" @click="openCreateModal()">
            <i class="fas fa-plus"></i>配置白名单
          </button>
        </div>
      </div>
      <!-- 移动端 -->
      <div class="flex md:hidden items-center justify-between">
        <div class="flex items-center gap-3 min-w-0 flex-1">
          <div class="page-icon bg-amber-500">
            <i class="fas fa-shield-halved text-white"></i>
          </div>
          <div class="min-w-0">
            <h1 class="text-lg font-semibold text-slate-800 truncate">白名单管理</h1>
            <p class="text-xs text-slate-500 truncate">路由级 Consumer 白名单</p>
          </div>
        </div>
        <div class="flex items-center gap-1 flex-shrink-0">
          <button class="btn btn-secondary w-9 h-9 !px-0" title="刷新" @click="loadWhitelist()">
            <i class="fas fa-rotate text-sm"></i>
          </button>
          <button v-if="portal.hasPerm('POST /api/apisix/whitelist')" class="btn btn-amber w-9 h-9 !px-0" title="配置白名单" @click="openCreateModal()">
            <i class="fas fa-plus text-sm"></i>
          </button>
        </div>
      </div>
    </div>

    <!-- 移动端搜索 -->
    <div class="mobile-search">
      <PageSearch v-model="searchText" search-key="apisix-whitelist" placeholder="搜索路由或用户..." width-class="w-full" focus-color="amber" />
    </div>

    <!-- Loading -->
    <div v-if="loading" class="card-body">
      <div class="empty-state">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>
    </div>

    <!-- Empty -->
    <div v-else-if="filteredWhitelist.length === 0" class="card-body">
      <div class="empty-state">
        <div class="empty-state-icon">
          <i class="fas fa-shield-halved text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">{{ whitelist.length === 0 ? '暂无白名单数据' : '未找到匹配白名单' }}</p>
        <p class="text-sm text-slate-400">{{ whitelist.length === 0 ? '配置路由的 Consumer 白名单后将在此显示' : '尝试更换关键词或清空搜索条件' }}</p>
      </div>
    </div>

    <!-- 列表 -->
    <template v-else>
      <!-- 桌面端表格 -->
      <div class="card-table hidden md:block">
        <table class="w-full border-collapse">
          <thead>
            <tr class="bg-slate-50 border-b border-slate-200">
              <th class="th">路由</th>
              <th class="th">描述</th>
              <th class="th">白名单用户</th>
              <th class="w-24 th-right">用户数</th>
              <th class="w-24 th-right">操作</th>
            </tr>
          </thead>
          <tbody class="bg-white divide-y divide-slate-100">
            <tr v-for="route in filteredWhitelist" :key="route.id" class="hover:bg-slate-50 transition-colors">
              <td class="px-4 py-3 max-w-[280px]">
                <div class="flex items-center gap-2 min-w-0">
                  <div class="row-icon bg-amber-400">
                    <i class="fas fa-shield-halved text-white text-sm"></i>
                  </div>
                  <div class="min-w-0">
                    <span class="font-medium text-slate-800 truncate block">{{ getRouteUri(route) }}</span>
                    <span class="text-xs text-slate-400 truncate block mt-0.5">{{ getRouteHost(route) }}</span>
                  </div>
                </div>
              </td>
              <td class="px-4 py-3 text-xs text-slate-500">{{ route.name || route.id }}</td>
              <td class="px-4 py-3">
                <div class="flex flex-wrap gap-1.5">
                  <span v-for="consumer in (route.consumers || [])" :key="consumer" class="inline-flex items-center gap-1.5 px-2 py-0.5 bg-amber-50 text-amber-800 rounded-lg text-xs group">
                    <i class="fas fa-user text-amber-500 text-[10px]"></i>
                    <span class="break-all">{{ consumer }}</span>
                    <button v-if="portal.hasPerm('POST /api/apisix/whitelist/revoke')" class="opacity-0 group-hover:opacity-100 hover:text-red-500 transition-all" title="撤销" @click="revokeConsumer(route, consumer)"><i class="fas fa-xmark text-[10px]"></i></button>
                  </span>
                </div>
              </td>
              <td class="px-4 py-3 text-right text-xs text-slate-500">{{ (route.consumers || []).length }}</td>
              <td class="px-4 py-3 text-right">
                <button
                  v-if="portal.hasPerm('POST /api/apisix/whitelist')"
                  class="btn-icon btn-icon-amber"
                  title="添加用户"
                  @click="openAddUserModal(route)"
                >
                  <i class="fas fa-user-plus text-xs"></i>
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 移动端卡片 -->
      <div class="card-body md:hidden space-y-3">
        <div v-for="route in filteredWhitelist" :key="route.id" class="card-interactive">
          <!-- 顶部：路由标识 -->
          <div class="flex items-center justify-between mb-3">
            <div class="flex items-center gap-3 min-w-0 flex-1">
              <div class="list-icon bg-amber-400">
                <i class="fas fa-shield-halved text-white text-base"></i>
              </div>
              <div class="min-w-0">
                <div class="font-medium text-slate-800 text-sm truncate">{{ getRouteUri(route) }}</div>
                <div class="text-xs text-slate-400 truncate mt-0.5">{{ getRouteHost(route) }}</div>
              </div>
            </div>
          </div>

          <!-- 描述 -->
          <div class="flex items-center gap-2 mb-3">
            <span class="text-xs text-slate-400 flex-shrink-0">描述</span>
            <span class="text-xs text-slate-500">{{ route.name || route.id }}</span>
          </div>

          <!-- 用户列表 -->
          <div class="flex items-start gap-2 mb-3">
            <span class="text-xs text-slate-400 flex-shrink-0 mt-0.5">用户</span>
            <div class="flex flex-wrap gap-1.5">
              <span v-for="consumer in (route.consumers || [])" :key="consumer" class="inline-flex items-center gap-1.5 px-2.5 py-1 bg-amber-50 text-amber-800 rounded-lg text-xs">
                <i class="fas fa-user text-amber-500 text-[10px]"></i>
                <span class="break-all">{{ consumer }}</span>
                <button v-if="portal.hasPerm('POST /api/apisix/whitelist/revoke')" class="hover:text-red-500 transition-colors" title="撤销" @click="revokeConsumer(route, consumer)"><i class="fas fa-xmark text-[10px]"></i></button>
              </span>
            </div>
          </div>

          <!-- 底部：操作按钮 -->
          <div class="card-actions">
            <button v-if="portal.hasPerm('POST /api/apisix/whitelist')" class="btn-icon btn-icon-amber" title="添加用户" @click="openAddUserModal(route)">
              <i class="fas fa-user-plus text-xs"></i><span class="text-xs ml-1">添加用户</span>
            </button>
          </div>
        </div>
      </div>
    </template>
  </div>

  <WhitelistEditModal ref="editModalRef" @success="loadWhitelist" />

  <!-- 添加用户 Modal -->
  <BaseModal v-model="addUser.open" title="添加白名单用户" :loading="addUser.loading" confirm-class="btn-amber" @confirm="handleAddUser">
    <div class="max-w-3xl space-y-4 p-1">
      <!-- 路由信息 -->
      <p v-if="addUser.route" class="text-xs text-slate-400">
        为路由 <span class="font-medium text-slate-600">{{ addUser.route.name || addUser.route.id }}</span>
        （<span class="text-slate-500">{{ getRouteUri(addUser.route) }}</span>）添加白名单用户
      </p>

      <!-- 模式切换 -->
      <div class="tab-group w-full">
        <button
          type="button"
          class="tab-btn flex-1 justify-center"
          :class="addUser.mode === 'existing' ? 'tab-btn-active text-amber-600' : 'tab-btn-inactive'"
          @click="switchAddUserMode('existing')"
        >
          <i class="fas fa-user-check"></i>选择已有用户
        </button>
        <button
          type="button"
          class="tab-btn flex-1 justify-center"
          :class="addUser.mode === 'new' ? 'tab-btn-active text-amber-600' : 'tab-btn-inactive'"
          @click="switchAddUserMode('new')"
        >
          <i class="fas fa-user-plus"></i>新建用户
        </button>
      </div>

      <!-- 选择已有用户 -->
      <div v-if="addUser.mode === 'existing'" class="space-y-2">
        <label class="form-label">Consumer <span class="text-red-500">*</span></label>
        <Combobox
          :model-value="addUser.selectedUsernames"
          multiple
          placeholder="搜索并选择 Consumer，可多选"
          search-placeholder="搜索 Consumer"
          max-height="280px"
          :tag-class="consumerTagClass"
          @update:model-value="updateSelectedUsernames"
        >
          <template #hint-extra="{ query }">
            <span class="text-xs text-slate-400">{{ filteredSelectableConsumers(query).length }} 个可选</span>
          </template>

          <template #default="{ query, select }">
            <div class="select-list p-2">
              <button
                v-for="consumer in filteredSelectableConsumers(query)"
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
            <div v-if="filteredSelectableConsumers('').length === 0" class="py-8 text-center">
              <i class="fas fa-search text-2xl text-slate-300 mb-2"></i>
              <p class="text-sm text-slate-400">{{ keyAuthConsumers.length === 0 ? '暂无已配置 key-auth 的 Consumer' : '全部已选或均已在白名单中' }}</p>
            </div>
          </template>
        </Combobox>
        <p class="text-xs text-slate-400">仅展示已配置 key-auth 插件且不在当前白名单中的 Consumer</p>
      </div>

      <!-- 新建用户 -->
      <div v-else class="space-y-3">
        <div>
          <label class="form-label">用户名 <span class="text-red-500">*</span></label>
          <input v-model="addUser.newUsername" type="text" class="input" placeholder="请输入 Consumer 用户名" />
        </div>
        <div>
          <label class="form-label">key-auth key <span class="text-red-500">*</span></label>
          <input v-model="addUser.newKey" type="text" class="input" placeholder="请输入 API Key" />
          <p class="mt-1 text-xs text-slate-400">提交后会先创建 Consumer 并配置 key-auth，再加入路由白名单</p>
        </div>
      </div>
    </div>

    <template #confirm-text>
      确认添加
    </template>
  </BaseModal>
</template>
