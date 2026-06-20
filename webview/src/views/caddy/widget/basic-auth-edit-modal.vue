<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { CaddyBasicAuthRoute, CaddyRoute } from '@/service/types'

import BaseModal from '@/component/modal.vue'
import Combobox from '@/component/combobox.vue'
import ToggleCard from '@/component/toggle-card.vue'

type Mode = 'setup' | 'edit' | 'addUser'

const defaultForm = () => ({
    routeIndex: -1,
    username: '',
    password: '',
    realm: '',
    forwardUser: false,
    forwardHeader: 'X-Remote-User',
    users: [] as string[],
})

@Component({
    expose: ['showSetup', 'showEdit', 'showAddUser'],
    components: { BaseModal, Combobox, ToggleCard },
    emits: ['success']
})
class BasicAuthEditModal extends Vue {
    portal = usePortal()

    open = false
    loading = false
    mode: Mode = 'setup'
    routes: CaddyRoute[] = []
    authRoutes: CaddyBasicAuthRoute[] = []
    form = defaultForm()

    get title() {
        if (this.mode === 'edit') return '编辑认证配置'
        if (this.mode === 'addUser') return '添加认证账号'
        return '添加认证配置'
    }

    get confirmText() {
        if (this.mode === 'edit') return '保存配置'
        if (this.mode === 'addUser') return '确认添加'
        return '确认配置'
    }

    // edit 模式后端不做 realm 兜底，留空即清空；setup 模式留空表示不设置
    get realmPlaceholder() {
        return this.mode === 'edit' ? '留空将清空 Realm' : '留空则不设置 Realm'
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
        this.resetForm()
        this.mode = 'setup'
        this.open = true
        this.loading = true
        try {
            const [authRes, routeRes] = await Promise.all([api.caddyBasicAuthList(), api.caddyRouteList()])
            this.authRoutes = authRes.payload || []
            this.routes = routeRes.payload || []
        } catch {
            this.portal.showNotification('error', '加载路由失败')
        } finally {
            this.loading = false
        }
    }

    showEdit(route: CaddyBasicAuthRoute) {
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
        const name = route['@id'] || `路由 #${route.index}`
        const hosts = route.match?.flatMap(m => m.host || []).join(', ')
        const paths = route.match?.flatMap(m => m.path || []).join(', ')
        const suffix = [hosts, paths].filter(Boolean).join(' ')
        return suffix ? `${name}  (${suffix})` : name
    }

    getAuthRouteName(route: CaddyBasicAuthRoute) {
        return route.name || `路由 #${route.index}`
    }

    userTagClass() {
        return 'bg-cyan-50 text-cyan-800 border border-cyan-200'
    }

    filteredEditingUsers(query: string) {
        const q = query.toLowerCase()
        return (this.editingRoute?.users || [])
            .filter(user => !this.form.users.includes(user.username))
            .filter(user => !q || user.username.toLowerCase().includes(q))
    }

    updateEditingUsers(value: string | string[]) {
        const route = this.editingRoute
        if (!route || !Array.isArray(value)) return
        const allowed = new Set(route.users.map(user => user.username))
        this.form.users = value.filter(username => allowed.has(username))
    }

    async handleConfirm() {
        if (this.mode === 'setup') return this.handleSetup()
        if (this.mode === 'addUser') return this.handleAddUser()
        return this.handleEdit()
    }

    async handleSetup() {
        const { routeIndex, username, password, realm } = this.form
        if (routeIndex < 0) return this.portal.showNotification('error', '请选择目标路由')
        if (!username.trim()) return this.portal.showNotification('error', '请输入用户名')
        if (!password) return this.portal.showNotification('error', '请输入密码')

        this.loading = true
        try {
            await api.caddyBasicAuthUserCreate(routeIndex, {
                username: username.trim(),
                password,
                realm: realm || undefined,
                forwardHeader: this.forwardHeaderValue,
            })
            this.portal.showNotification('success', '认证配置成功')
            this.open = false
            this.$emit('success')
        } catch (e: unknown) {
            this.portal.showNotification('error', (e instanceof Error ? e.message : '') || '操作失败')
        } finally {
            this.loading = false
        }
    }

    async handleAddUser() {
        const { routeIndex, username, password, realm } = this.form
        if (!username.trim()) return this.portal.showNotification('error', '请输入用户名')
        if (!password) return this.portal.showNotification('error', '请输入密码')

        this.loading = true
        try {
            await api.caddyBasicAuthUserCreate(routeIndex, {
                username: username.trim(),
                password,
                realm: realm || undefined,
                forwardHeader: this.forwardHeaderValue,
            })
            this.portal.showNotification('success', `账号 "${username.trim()}" 添加成功`)
            this.open = false
            this.$emit('success')
        } catch (e: unknown) {
            this.portal.showNotification('error', (e instanceof Error ? e.message : '') || '操作失败')
        } finally {
            this.loading = false
        }
    }

    async handleEdit() {
        const route = this.editingRoute
        if (!route) return this.portal.showNotification('error', '路由不存在')
        const { routeIndex, realm, users } = this.form
        const removedUsers = route.users.map(user => user.username).filter(name => !users.includes(name))

        this.loading = true
        try {
            await api.caddyBasicAuthConfigUpdate(routeIndex, {
                realm,
                forwardHeader: this.forwardHeaderValue,
            })
            for (const name of removedUsers) {
                await api.caddyBasicAuthUserDelete(routeIndex, name)
            }
            this.portal.showNotification('success', '配置更新成功')
            this.open = false
            this.$emit('success')
        } catch (e: unknown) {
            this.portal.showNotification('error', (e instanceof Error ? e.message : '') || '操作失败')
        } finally {
            this.loading = false
        }
    }
}

export default toNative(BasicAuthEditModal)
</script>

<template>
  <BaseModal v-model="open" :title="title" :loading="loading" confirm-class="btn-cyan" @confirm="handleConfirm">
    <div class="max-w-3xl space-y-4 p-1">
      <div v-if="mode === 'setup'">
        <label class="form-label">目标路由 <span class="text-red-500">*</span></label>
        <select v-model="form.routeIndex" class="input">
          <option :value="-1" disabled>请选择路由</option>
          <option v-for="route in unauthRoutes" :key="route.index" :value="route.index">
            {{ getRouteLabel(route) }}
          </option>
        </select>
        <p v-if="unauthRoutes.length === 0" class="text-xs text-slate-400 mt-1">所有路由均已配置认证</p>
      </div>
      <div v-else-if="editingRoute">
        <label class="form-label">路由</label>
        <div class="detail-value text-sm text-slate-700">
          {{ getAuthRouteName(editingRoute) }}
        </div>
      </div>

      <!-- Realm 与透传配置：仅配置认证 / 编辑配置时显示 -->
      <template v-if="mode !== 'addUser'">
        <div>
          <label class="form-label">Realm</label>
          <input v-model="form.realm" type="text" class="input" :placeholder="realmPlaceholder" autocomplete="off" />
          <p class="text-xs text-slate-400 mt-1">HTTP Basic 认证弹窗中显示的描述文字</p>
        </div>
        <ToggleCard v-model="form.forwardUser" label="将认证用户名传给后端" desc="认证通过后，将登录用户名注入到指定请求头中转发给上游服务">
          <div>
            <label class="form-label">请求头名称</label>
            <input v-model="form.forwardHeader" type="text" class="input" placeholder="X-Remote-User" autocomplete="off" />
            <p class="text-xs text-slate-400 mt-1">Caddy 变量 <code class="bg-slate-100 px-1 rounded-lg">&#123;http.auth.user.id&#125;</code> 将被注入到此 Header</p>
          </div>
        </ToggleCard>
      </template>

      <!-- 编辑模式：管理已有账号（标记移除） -->
      <div v-if="mode === 'edit' && editingRoute">
        <label class="form-label">现有账号</label>
        <Combobox
          :model-value="form.users"
          multiple
          placeholder="暂无账号"
          search-placeholder="搜索账号"
          max-height="240px"
          :tag-class="userTagClass"
          @update:model-value="updateEditingUsers"
        >
          <template #hint-extra="{ query }">
            <span class="text-xs text-slate-400">{{ filteredEditingUsers(query).length }} 个可恢复</span>
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
            <div class="py-8 text-center text-sm text-slate-400">已选择全部账号</div>
          </template>
        </Combobox>
        <p class="text-xs text-slate-400 mt-1">点击账号标签上的 × 标记移除，保存后生效</p>
      </div>

      <!-- 账号录入：setup 创建首个账号 / addUser 追加账号 -->
      <div v-if="mode !== 'edit'" class="space-y-4">
        <div>
          <label class="form-label">用户名 <span class="text-red-500">*</span></label>
          <input v-model="form.username" type="text" class="input" placeholder="请输入用户名" autocomplete="off" />
        </div>
        <div>
          <label class="form-label">密码 <span class="text-red-500">*</span></label>
          <input v-model="form.password" type="password" class="input" placeholder="请输入密码" autocomplete="new-password" />
          <p class="text-xs text-slate-400 mt-1">密码将以 bcrypt 形式存储在 Caddy 配置中</p>
        </div>
      </div>
    </div>
    <template #confirm-text>{{ confirmText }}</template>
  </BaseModal>
</template>
