<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import { RouteAccessPerm } from '@/service/types'
import type { MemberInfo, MemberUpsert, Route } from '@/service/types'

import BaseModal from '@/component/modal.vue'

// 方法颜色映射
const METHOD_COLOR: Record<string, string> = {
    GET:    'bg-blue-50 text-blue-600',
    POST:   'bg-green-50 text-green-700',
    PUT:    'bg-yellow-50 text-yellow-700',
    PATCH:  'bg-orange-50 text-orange-700',
    DELETE: 'bg-red-50 text-red-600',
    ANY:    'bg-purple-50 text-purple-700',
}

// 模块元信息（图标 + 显示名）
const MODULE_META: Record<string, { icon: string; label: string }> = {
    overview: { icon: 'fas fa-gauge-high',  label: '系统概览' },
    system:   { icon: 'fas fa-gear',        label: '系统管理' },
    account:  { icon: 'fas fa-users',       label: '用户管理' },
    shell:    { icon: 'fas fa-terminal',    label: 'Shell 终端' },
    filer:    { icon: 'fas fa-folder',      label: '文件管理' },
    agent:    { icon: 'fas fa-robot',       label: 'AI Agent' },
    apisix:   { icon: 'fas fa-route',       label: 'APISIX' },
    caddy:    { icon: 'fas fa-shield',      label: 'Caddy 网关' },
    docker:   { icon: 'fab fa-docker',      label: 'Docker' },
    swarm:    { icon: 'fas fa-layer-group', label: 'Swarm' },
    compose:  { icon: 'fas fa-cubes',       label: 'Compose' },
    cron:     { icon: 'fas fa-clock',       label: '计划任务' },
}

interface RouteItem {
    key: string
    label: string
    access: number
}

interface RouteGroup {
    module: string
    label: string
    icon: string
    routes: RouteItem[]
}

@Component({
    expose: ['show'],
    components: { BaseModal },
    emits: ['success']
})
class MemberEditModal extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    isOpen = false
    modalLoading = false
    routesLoading = false
    routeGroups: RouteGroup[] = []
    // 非具体权限路由 key 集合（匿名/登录即可），默认勾选且不可取消
    autoPerms: Set<string> = new Set()
    // formData.permissions 的 Set 镜像，用于 O(1) 查找
    permSet: Set<string> = new Set()
    methodColor = METHOD_COLOR
    // 编辑时保存原始用户名，用于后端定位
    originalUsername = ''
    formData: MemberUpsert = {
        username: '',
        password: '',
        homeDirectory: '',
        description: '',
        permissions: []
    }

    // ─── 计算属性 ───
    get isEdit() {
        return this.originalUsername !== ''
    }

    get title() {
        return this.isEdit ? '编辑成员' : '新建成员'
    }

    // ─── 方法 ───
    async loadRoutes() {
        if (this.routeGroups.length > 0) return
        this.routesLoading = true
        try {
            const res = await api.accountRouteList()
            const items: Route[] = res.payload || []
            // 收集非具体权限路由（匿名/登录即可），自动勾选且不可取消
            const autoPerms = new Set<string>()
            for (const item of items) {
                if (item.access !== RouteAccessPerm) autoPerms.add(item.key)
            }
            this.autoPerms = autoPerms
            // 按 module 分组，保持模块顺序
            const moduleOrder = Object.keys(MODULE_META)
            const groupMap = new Map<string, RouteItem[]>()
            for (const item of items) {
                if (!groupMap.has(item.module)) groupMap.set(item.module, [])
                groupMap.get(item.module)?.push({ key: item.key, label: item.label || item.key, access: item.access })
            }
            // 按预定义顺序排列模块，未知模块追加到末尾
            const ordered: RouteGroup[] = []
            for (const mod of moduleOrder) {
                if (groupMap.has(mod)) {
                    const meta = MODULE_META[mod] || { icon: 'fas fa-circle', label: mod }
                    ordered.push({ module: mod, label: meta.label, icon: meta.icon, routes: groupMap.get(mod) ?? [] })
                    groupMap.delete(mod)
                }
            }
            for (const [mod, routes] of groupMap) {
                const meta = MODULE_META[mod] || { icon: 'fas fa-circle', label: mod }
                ordered.push({ module: mod, label: meta.label, icon: meta.icon, routes })
            }
            this.routeGroups = ordered
        } finally {
            this.routesLoading = false
        }
    }

    async show(member: MemberInfo | null = null) {
        if (member) {
            this.originalUsername = member.username
            const perms = [...(member.permissions || [])]
            this.formData = {
                username: member.username,
                password: '',
                homeDirectory: member.homeDirectory,
                description: member.description,
                permissions: perms
            }
            this.permSet = new Set(perms)
        } else {
            this.originalUsername = ''
            this.formData = {
                username: '', password: '', homeDirectory: '', description: '',
                permissions: []
            }
            this.permSet = new Set()
        }
        this.isOpen = true
        await this.loadRoutes()
    }

    // 路由是否被勾选（自动权限路由始终返回 true）
    isChecked(key: string): boolean {
        return this.autoPerms.has(key) || this.permSet.has(key)
    }

    // 路由是否为自动勾选（不可手动取消）
    isAuto(key: string): boolean {
        return this.autoPerms.has(key)
    }

    togglePerm(key: string) {
        if (this.autoPerms.has(key)) return
        if (this.permSet.has(key)) {
            this.permSet.delete(key)
            this.formData.permissions = this.formData.permissions.filter(k => k !== key)
        } else {
            this.permSet.add(key)
            this.formData.permissions = [...this.formData.permissions, key]
        }
    }

    manualRouteKeys(): string[] {
        return this.routeGroups
            .flatMap(group => group.routes)
            .filter(route => !this.autoPerms.has(route.key))
            .map(route => route.key)
    }

    applyRouteShortcut(keys: string[]) {
        const manualKeySet = new Set(this.manualRouteKeys())
        const nextKeySet = new Set(keys)
        const preserved = this.formData.permissions.filter(key => !manualKeySet.has(key))
        this.formData.permissions = [...preserved, ...keys]
        this.permSet = new Set([...preserved, ...nextKeySet])
    }

    selectAllRoutes() {
        this.applyRouteShortcut(this.manualRouteKeys())
    }

    selectReadOnlyRoutes() {
        const keys = this.routeGroups
            .flatMap(group => group.routes)
            .filter(route => !this.autoPerms.has(route.key) && this.methodOf(route.key) === 'GET')
            .map(route => route.key)
        this.applyRouteShortcut(keys)
    }

    clearRoutes() {
        this.applyRouteShortcut([])
    }

    isGroupAllChecked(routes: RouteItem[]): boolean {
        return routes.every(r => this.isChecked(r.key))
    }

    isGroupPartialChecked(routes: RouteItem[]): boolean {
        const checked = routes.filter(r => this.isChecked(r.key)).length
        return checked > 0 && checked < routes.length
    }

    toggleGroup(routes: RouteItem[]) {
        // 只操作非自动权限的路由
        const manualRoutes = routes.filter(r => !this.autoPerms.has(r.key)).map(r => r.key)
        if (manualRoutes.length === 0) return
        const allChecked = manualRoutes.every(k => this.permSet.has(k))
        if (allChecked) {
            const removeSet = new Set(manualRoutes)
            manualRoutes.forEach(k => this.permSet.delete(k))
            this.formData.permissions = this.formData.permissions.filter(k => !removeSet.has(k))
        } else {
            const toAdd = manualRoutes.filter(k => !this.permSet.has(k))
            toAdd.forEach(k => this.permSet.add(k))
            this.formData.permissions = [...this.formData.permissions, ...toAdd]
        }
    }

    methodOf(key: string): string {
        return key.split(' ')[0]
    }

    pathOf(key: string): string {
        return key.split(' ')[1]
    }

    async handleConfirm() {
        if (!this.formData.username?.trim()) return
        if (!this.isEdit && !this.formData.password?.trim()) {
            this.portal.showNotification('warning', '请填写登录密码')
            return
        }
        this.modalLoading = true
        try {
            if (this.isEdit) {
                await api.accountMemberUpdate(this.originalUsername, this.formData)
                this.portal.showNotification('success', '成员更新成功')
            } else {
                await api.accountMemberCreate(this.formData)
                this.portal.showNotification('success', '成员添加成功')
            }
            this.isOpen = false
            this.$emit('success')
        } catch {
        } finally {
            this.modalLoading = false
        }
    }
}

export default toNative(MemberEditModal)
</script>

<template>
  <BaseModal v-model="isOpen" :title="title" :loading="modalLoading" confirm-class="btn-blue" show-footer @confirm="handleConfirm">
    <form class="space-y-4" @submit.prevent="handleConfirm">
      <div>
        <label class="form-label">用户名 <span class="text-red-500">*</span></label>
        <input v-model="formData.username" type="text" placeholder="请输入用户名" required :disabled="isEdit" class="input disabled:bg-slate-50 disabled:text-slate-500" autocomplete="off" />
        <p v-if="isEdit" class="mt-1 text-xs text-slate-400">用户名不可修改</p>
      </div>
      <div>
        <label class="form-label">
          密码
          <span v-if="!isEdit" class="text-red-500">*</span>
          <span v-else class="text-slate-400 font-normal">(留空则保持不变)</span>
        </label>
        <input v-model="formData.password" type="password" :placeholder="isEdit ? '留空则保持不变' : '请输入登录密码'" class="input" autocomplete="new-password" />
      </div>
      <div>
        <label class="form-label">家目录 <span class="text-slate-400 font-normal">(可选)</span></label>
        <input v-model="formData.homeDirectory" type="text" placeholder="请输入家目录（可选）" class="input" />
        <p class="mt-1 text-xs text-slate-400">相对路径基于"基础目录"，留空则自动创建为 基础目录/用户名</p>
      </div>
      <div>
        <label class="form-label">描述 <span class="text-slate-400 font-normal">(可选)</span></label>
        <input v-model="formData.description" type="text" placeholder="请输入成员描述（可选）" class="input" maxlength="64" />
        <p class="mt-1 text-xs text-slate-400">用于标识成员用途，最长 64 字符</p>
      </div>
      <!-- 路由权限 -->
      <div>
        <div class="flex items-center gap-2 mb-2">
          <label class="form-label">路由权限</label>
          <div class="ml-auto flex items-center gap-1">
            <button type="button" class="tab-btn tab-btn-text bg-blue-50 text-blue-600 hover:bg-blue-100" @click="selectAllRoutes">全选</button>
            <button type="button" class="tab-btn tab-btn-text bg-slate-50 text-slate-600 hover:bg-slate-100" @click="selectReadOnlyRoutes">只读</button>
            <button type="button" class="tab-btn tab-btn-text bg-red-50 text-red-600 hover:bg-red-100" @click="clearRoutes">清空</button>
          </div>
        </div>
        <div class="space-y-2">
          <div v-for="group in routeGroups" :key="group.module" class="rounded-lg border border-slate-200 overflow-hidden">
            <!-- 模块标题行（全选/取消） -->
            <div
              class="flex items-center gap-2 px-3 py-2 bg-slate-100 border-b border-slate-200 cursor-pointer select-none hover:bg-slate-200 transition-colors"
              @click="toggleGroup(group.routes)"
            >
              <input
                type="checkbox"
                :checked="isGroupAllChecked(group.routes)"
                :indeterminate="isGroupPartialChecked(group.routes)"
                class="w-4 h-4 accent-blue-500 pointer-events-none"
                readonly
              />
              <i :class="[group.icon, 'text-slate-400 w-4 text-center text-xs']"></i>
              <span class="text-xs font-semibold text-slate-700">{{ group.label }}</span>
              <span class="ml-auto text-xs text-slate-400">
                {{ group.routes.filter(r => isChecked(r.key)).length }} / {{ group.routes.length }}
              </span>
            </div>
            <!-- 路由列表 -->
            <div class="divide-y divide-slate-100">
              <label
                v-for="item in group.routes"
                :key="item.key"
                :class="['flex items-center gap-2 px-3 py-1.5', isAuto(item.key) ? 'opacity-60 cursor-not-allowed' : 'hover:bg-slate-50 cursor-pointer']"
              >
                <input type="checkbox" :checked="isChecked(item.key)" :disabled="isAuto(item.key)" class="w-4 h-4 accent-blue-500 flex-shrink-0" @change="togglePerm(item.key)" />
                <span :class="['inline-block w-14 text-center text-xs font-mono font-semibold rounded px-1 py-0.5 flex-shrink-0', methodColor[methodOf(item.key)] || 'bg-slate-100 text-slate-600']">
                  {{ methodOf(item.key) }}
                </span>
                <span class="min-w-0 flex-1 flex items-center gap-2">
                  <span class="text-xs text-slate-700 truncate">{{ item.label }}</span>
                  <code class="hidden sm:inline text-xs text-slate-400 font-mono truncate">{{ pathOf(item.key) }}</code>
                </span>
                <span v-if="isAuto(item.key)" class="ml-auto text-xs text-slate-400 flex-shrink-0">自动</span>
              </label>
            </div>
          </div>
        </div>
      </div>
    </form>
  </BaseModal>
</template>
