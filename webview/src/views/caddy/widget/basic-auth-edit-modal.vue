<script lang="ts">
import { Component, Prop, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { CaddyBasicAuthRoute, CaddyRoute } from '@/service/types'

import Combobox from '@/component/combobox.vue'
import BaseModal from '@/component/modal.vue'
import ToggleCard from '@/component/toggle-card.vue'

type Mode = 'setup' | 'edit' | 'addUser'

const defaultForm = () => ({
    routeIndex: -1,
    username: '',
    password: '',
    realm: '',
    forwardUser: true,
    forwardHeader: 'X-Remote-User',
    users: [] as string[],
})

@Component({
    expose: ['showSetup', 'showEdit', 'showAddUser'],
    components: { BaseModal, Combobox, ToggleCard },
    emits: ['success', 'refresh']
})
class BasicAuthEditModal extends Vue {
    portal = usePortal()

    @Prop({ type: String, default: 'srv0' }) readonly server!: string

    open = false
    loading = false
    mode: Mode = 'setup'
    targetServer = 'srv0'
    routes: CaddyRoute[] = []
    authRoutes: CaddyBasicAuthRoute[] = []
    form = defaultForm()
    revision = ''

    get canCreate() {
        return this.portal.hasPerm('POST /api/caddy/basic-auth/:index/users')
    }

    get canUpdate() {
        return this.portal.hasPerm('PUT /api/caddy/basic-auth/:index/config')
    }

    get canDelete() {
        return this.portal.hasPerm('DELETE /api/caddy/basic-auth/:index/users/:username')
    }

    get confirmDisabled() {
        return !this.revision || (this.mode === 'edit' ? !this.canUpdate : !this.canCreate)
    }

    invalidateDraft() {
        this.resetForm()
        this.authRoutes = []
        this.routes = []
        this.revision = ''
        this.open = false
        this.$emit('refresh')
    }

    get title() {
        if (this.mode === 'edit') {
            return this.$t('编辑认证配置')
        }
        if (this.mode === 'addUser') {
            return this.$t('添加认证账号')
        }
        return this.$t('添加认证配置')
    }

    get confirmText() {
        return this.mode === 'edit' ? this.$t('保存') : this.$t('添加')
    }

    // edit 模式后端不做 realm 兜底，留空即清空；setup 模式留空表示不设置
    get realmPlaceholder() {
        return this.mode === 'edit' ? this.$t('留空将清空 Realm') : this.$t('留空则不设置 Realm')
    }

    get authRouteIndexSet() {
        return new Set(this.authRoutes.map(route => route.index))
    }

    get unauthRoutes() {
        return this.routes.filter(route => !this.authRouteIndexSet.has(route.index))
    }

    get editingRoute() {
        return this.authRoutes.find(route => route.index === this.form.routeIndex) || null
    }

    // 透传请求头的归一化值：关闭时为空串，开启但未填写时回退默认值
    get forwardHeaderValue() {
        const { forwardUser, forwardHeader } = this.form
        return forwardUser ? (forwardHeader.trim() || 'X-Remote-User') : ''
    }

    resetForm() {
        Object.assign(this.form, defaultForm())
    }

    async showSetup() {
        if (this.loading || !this.canCreate) {
            return
        }
        this.targetServer = this.server
        this.resetForm()
        this.authRoutes = []
        this.routes = []
        this.revision = ''
        this.mode = 'setup'
        this.open = true
        this.loading = true
        try {
            const [authRes, routeRes] = await Promise.all([
                api.caddyBasicAuthSnapshot(this.targetServer),
                api.caddyRouteSnapshot(this.targetServer),
            ])
            const authSnapshot = authRes.payload
            const routeSnapshot = routeRes.payload
            if (!authSnapshot?.revision || authSnapshot.revision !== routeSnapshot?.revision) {
                this.portal.showNotification('error', this.$t('路由列表已变化，请刷新后重试'))
                this.invalidateDraft()
                return
            }
            this.authRoutes = authSnapshot.routes || []
            this.routes = routeSnapshot.routes || []
            this.revision = authSnapshot.revision
        } catch (error) {
            this.invalidateDraft()
            throw error
        } finally {
            this.loading = false
        }
    }

    showEdit(route: CaddyBasicAuthRoute) {
        if (this.loading || !this.canUpdate) {
            return
        }
        this.revision = route.revision
        this.targetServer = this.server
        this.resetForm()
        this.mode = 'edit'
        this.authRoutes = [route]
        Object.assign(this.form, {
            routeIndex: route.index,
            realm: route.realm || '',
            forwardUser: !!route.forwardHeader,
            forwardHeader: route.forwardHeader || 'X-Remote-User',
            users: route.users.map(user => user.username),
        })
        this.open = true
    }

    // 为已配置认证的路由追加账号：沿用其 realm 与透传头，仅录入用户名/密码
    showAddUser(route: CaddyBasicAuthRoute) {
        if (this.loading || !this.canCreate) {
            return
        }
        this.revision = route.revision
        this.targetServer = this.server
        this.resetForm()
        this.mode = 'addUser'
        this.authRoutes = [route]
        Object.assign(this.form, {
            routeIndex: route.index,
            realm: route.realm || '',
            forwardUser: !!route.forwardHeader,
            forwardHeader: route.forwardHeader || 'X-Remote-User',
        })
        this.open = true
    }

    getRouteLabel(route: CaddyRoute) {
        const name = route['@id'] || this.$t('路由 #{0}', route.index)
        const hosts = route.match?.flatMap(m => m.host || []).join(', ')
        const paths = route.match?.flatMap(m => m.path || []).join(', ')
        const suffix = [hosts, paths].filter(Boolean).join(' ')
        return suffix ? `${name}  (${suffix})` : name
    }

    getAuthRouteName(route: CaddyBasicAuthRoute) {
        return route.name || this.$t('路由 #{0}', route.index)
    }

    userTagClass() {
        return 'bg-cyan-50 text-cyan-700 border border-cyan-200'
    }

    userTagDisabled() {
        return !this.canDelete || this.form.users.length <= 1
    }

    filteredEditingUsers(query: string) {
        const q = query.toLowerCase()
        return (this.editingRoute?.users || [])
            .filter(user => !this.form.users.includes(user.username))
            .filter(user => !q || user.username.toLowerCase().includes(q))
    }

    updateEditingUsers(value: string | string[]) {
        const route = this.editingRoute
        if (!route || !Array.isArray(value) || this.loading || !this.canDelete) {
            return
        }
        const allowed = new Set(route.users.map(user => user.username))
        const users = [...new Set(value.filter(username => allowed.has(username)))]
        if (users.length === 0) {
            this.portal.showNotification('error', this.$t('请至少保留一个账号；解除认证请使用账号删除操作'))
            return
        }
        this.form.users = users
    }

    async handleConfirm() {
        if (this.loading || this.confirmDisabled || !this.open) {
            return
        }
        if (this.mode === 'setup') {
            return this.handleSetup()
        }
        if (this.mode === 'addUser') {
            return this.handleAddUser()
        }
        return this.handleEdit()
    }

    async handleSetup() {
        const { routeIndex, username, password, realm } = this.form
        if (routeIndex < 0) {
            return this.portal.showNotification('error', this.$t('请选择目标路由'))
        }
        if (!username.trim()) {
            return this.portal.showNotification('error', this.$t('请输入用户名'))
        }
        if (!password) {
            return this.portal.showNotification('error', this.$t('请输入密码'))
        }

        this.loading = true
        try {
            await api.caddyBasicAuthUserCreate(routeIndex, {
                username: username.trim(),
                password,
                revision: this.revision,
                realm: realm || undefined,
                forwardHeader: this.forwardHeaderValue,
            }, this.targetServer)
            this.portal.showNotification('success', this.$t('认证配置成功'))
            this.open = false
            this.$emit('success')
        } catch (error) {
            this.invalidateDraft()
            throw error
        } finally {
            this.loading = false
        }
    }

    async handleAddUser() {
        const { routeIndex, username, password, realm } = this.form
        if (!username.trim()) {
            return this.portal.showNotification('error', this.$t('请输入用户名'))
        }
        if (!password) {
            return this.portal.showNotification('error', this.$t('请输入密码'))
        }

        this.loading = true
        try {
            await api.caddyBasicAuthUserCreate(routeIndex, {
                username: username.trim(),
                password,
                revision: this.revision,
                realm: realm || undefined,
                forwardHeader: this.forwardHeaderValue,
            }, this.targetServer)
            this.portal.showNotification('success', this.$t('账号 "{0}" 添加成功', username.trim()))
            this.open = false
            this.$emit('success')
        } catch (error) {
            this.invalidateDraft()
            throw error
        } finally {
            this.loading = false
        }
    }

    async handleEdit() {
        const route = this.editingRoute
        if (!route) {
            return this.portal.showNotification('error', this.$t('路由不存在'))
        }
        const { routeIndex, realm, users } = this.form
        const removedUsers = route.users.map(user => user.username).filter(name => !users.includes(name))
        if (removedUsers.length > 0 && !this.canDelete) {
            return this.portal.showNotification('error', this.$t('没有删除账号的权限'))
        }
        if (users.length === 0) {
            return this.portal.showNotification('error', this.$t('请至少保留一个账号；解除认证请使用账号删除操作'))
        }

        this.loading = true
        try {
            await api.caddyBasicAuthConfigUpdate(routeIndex, {
                realm,
                forwardHeader: this.forwardHeaderValue,
                revision: this.revision,
                removeUsers: removedUsers,
            }, this.targetServer)
            this.portal.showNotification('success', this.$t('配置更新成功'))
            this.open = false
            this.$emit('success')
        } catch (error) {
            this.invalidateDraft()
            throw error
        } finally {
            this.loading = false
        }
    }
}

export default toNative(BasicAuthEditModal)
</script>

<template>
  <BaseModal v-model="open" :title="title" :loading="loading" :confirm-disabled="confirmDisabled" :confirm-class="mode === 'addUser' ? 'btn-emerald' : 'btn-cyan'" @confirm="handleConfirm">
    <div class="space-y-4">
      <div v-if="mode === 'setup'">
        <label class="form-label">{{ $t('目标路由') }} <span class="text-red-500">*</span></label>
        <select v-model="form.routeIndex" class="input">
          <option :value="-1" disabled>{{ $t('请选择路由') }}</option>
          <option v-for="route in unauthRoutes" :key="route.index" :value="route.index">
            {{ getRouteLabel(route) }}
          </option>
        </select>
        <p v-if="unauthRoutes.length === 0" class="form-help">{{ $t('所有路由均已配置认证') }}</p>
      </div>
      <div v-else-if="editingRoute">
        <label class="form-label">{{ $t('路由') }}</label>
        <div class="detail-value text-sm text-slate-700">
          {{ getAuthRouteName(editingRoute) }}
        </div>
      </div>

      <!-- Realm 与透传配置：仅配置认证 / 编辑配置时显示 -->
      <template v-if="mode !== 'addUser'">
        <div>
          <label class="form-label">Realm</label>
          <input v-model="form.realm" type="text" class="input" :placeholder="realmPlaceholder" autocomplete="off" />
          <p class="form-help">{{ $t('HTTP Basic 认证弹窗中显示的描述文字') }}</p>
        </div>
        <ToggleCard v-model="form.forwardUser" :label="$t('将认证用户名传给后端')" :desc="$t('认证通过后，将登录用户名注入到指定请求头中转发给上游服务')">
          <div>
            <label class="form-label">{{ $t('请求头名称') }}</label>
            <input v-model="form.forwardHeader" type="text" class="input" placeholder="X-Remote-User" autocomplete="off" />
            <p class="form-help">{{ $t('Caddy 变量') }} <code class="bg-slate-100 px-1 rounded-lg">&#123;http.auth.user.id&#125;</code> {{ $t('将被注入到此 Header') }}</p>
          </div>
        </ToggleCard>
      </template>

      <!-- 编辑模式：管理已有账号（标记移除） -->
      <div v-if="mode === 'edit' && editingRoute">
        <label class="form-label">{{ $t('现有账号') }}</label>
        <Combobox
          :model-value="form.users"
          multiple
          :placeholder="$t('暂无账号')"
          :search-placeholder="$t('搜索账号')"
          max-height="240px"
          :tag-class="userTagClass"
          :tag-disabled="userTagDisabled"
          :disabled="loading || !canDelete"
          :allow-custom="false"
          @update:model-value="updateEditingUsers"
        >
          <template #hint-extra="{ query }">
            <span class="text-xs text-slate-400">{{ filteredEditingUsers(query).length }} {{ $t('个可恢复') }}</span>
          </template>

          <template #default="{ query, select }">
            <div class="select-list p-2">
              <button
                v-for="user in filteredEditingUsers(query)"
                :key="user.username"
                type="button"
                class="flex w-full items-center gap-2.5 rounded-lg border border-transparent px-2.5 py-2 text-left transition-all duration-150 hover:bg-slate-50"
                @click="select(user.username)"
              >
                <span class="row-icon bg-cyan-50 text-cyan-600">
                  <i class="fas fa-user text-xs"></i>
                </span>
                <span class="block truncate text-sm font-medium text-slate-700">{{ user.username }}</span>
              </button>
            </div>
          </template>

          <template #empty>
            <div class="py-8 text-center text-sm text-slate-400">{{ $t('已选择全部账号') }}</div>
          </template>
        </Combobox>
        <p v-if="canDelete" class="form-help">{{ $t('点击账号标签上的 × 标记移除，保存后生效') }}</p>
        <p v-else class="form-help">{{ $t('没有删除账号的权限') }}</p>
        <p class="text-xs text-amber-600 mt-1">{{ $t('最后一个账号不能在编辑配置中移除，请在列表中删除并确认解除认证') }}</p>
      </div>

      <!-- 账号录入：setup 创建首个账号 / addUser 追加账号 -->
      <div v-if="mode !== 'edit'" class="space-y-4">
        <div>
          <label class="form-label">{{ $t('用户名') }} <span class="text-red-500">*</span></label>
          <input v-model="form.username" type="text" class="input" :placeholder="$t('请输入用户名')" autocomplete="off" />
        </div>
        <div>
          <label class="form-label">{{ $t('密码') }} <span class="text-red-500">*</span></label>
          <input v-model="form.password" type="password" class="input" :placeholder="$t('请输入密码')" autocomplete="new-password" />
          <p class="form-help">{{ $t('密码将以 bcrypt 形式存储在 Caddy 配置中') }}</p>
        </div>
      </div>
    </div>
    <template #confirm-text>{{ confirmText }}</template>
  </BaseModal>
</template>
