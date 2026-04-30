<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

import api from '@/service/api'
import type { SystemMemberInfo, SystemMemberUpsertRequest } from '@/service/types'

import BaseModal from '@/component/modal.vue'

// 可配置的模块列表
const MODULES = [
    { key: 'filer',   label: '文件管理', icon: 'fas fa-folder' },
    { key: 'docker',  label: 'Docker',   icon: 'fab fa-docker' },
    { key: 'swarm',   label: 'Swarm',    icon: 'fas fa-layer-group' },
    { key: 'compose', label: 'Compose',  icon: 'fas fa-cubes' },
    { key: 'apisix',  label: 'APISIX',   icon: 'fas fa-route' },
    { key: 'agent',   label: 'AI Agent', icon: 'fas fa-robot' },
    { key: 'system',  label: '系统管理', icon: 'fas fa-gear' },
    { key: 'shell',   label: 'Shell终端', icon: 'fas fa-terminal' },
]

function emptyPermissions(): Record<string, string> {
    const p: Record<string, string> = {}
    for (const m of MODULES) p[m.key] = ''
    return p
}

@Component({
    expose: ['show'],
    components: { BaseModal },
    emits: ['success']
})
class MemberEditModal extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    // ─── 数据属性 ───
    isOpen = false
    modalLoading = false
    modules = MODULES
    // 编辑时保存原始用户名，用于后端定位
    originalUsername = ''
    passwordSet = false
    formData: SystemMemberUpsertRequest = {
        username: '', password: '', homeDirectory: '',
        permissions: emptyPermissions()
    }

    // ─── 计算属性 ───
    get isEdit() {
        return this.originalUsername !== ''
    }

    get title() {
        return this.isEdit ? '编辑成员' : '添加成员'
    }

    get passwordPlaceholder() {
        if (!this.isEdit) return '登录密码'
        return this.passwordSet ? '留空则保持不变' : '尚未设置'
    }

    // ─── 方法 ───
    show(member: SystemMemberInfo | null = null) {
        if (member) {
            this.originalUsername = member.username
            this.passwordSet = member.passwordSet
            const perms = emptyPermissions()
            if (member.permissions) {
                for (const k of Object.keys(perms)) {
                    perms[k] = member.permissions[k] || ''
                }
            }
            this.formData = {
                username: member.username,
                password: '',
                homeDirectory: member.homeDirectory,
                permissions: perms
            }
        } else {
            this.originalUsername = ''
            this.passwordSet = false
            this.formData = {
                username: '', password: '', homeDirectory: '',
                permissions: emptyPermissions()
            }
        }
        this.isOpen = true
    }

    setPermission(moduleKey: string, value: string) {
        this.formData.permissions = { ...this.formData.permissions, [moduleKey]: value }
    }

    async handleConfirm() {
        if (!this.formData.username?.trim()) return
        if (!this.isEdit && !this.formData.password?.trim()) {
            this.actions.showNotification('warning', '请填写登录密码')
            return
        }
        this.modalLoading = true
        try {
            if (this.isEdit) {
                await api.updateMember(this.originalUsername, this.formData)
                this.actions.showNotification('success', '成员更新成功')
            } else {
                await api.createMember(this.formData)
                this.actions.showNotification('success', '成员添加成功')
            }
            this.isOpen = false
            this.$emit('success')
        } catch (e) {}
        this.modalLoading = false
    }
}

export default toNative(MemberEditModal)
</script>

<template>
  <BaseModal
    v-model="isOpen"
    :title="title"
    :loading="modalLoading"
    show-footer
    @confirm="handleConfirm"
  >
    <form @submit.prevent="handleConfirm" class="space-y-4">
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">用户名 <span class="text-red-500">*</span></label>
        <input type="text" v-model="formData.username" placeholder="登录用户名" required :disabled="isEdit" class="input disabled:bg-slate-50 disabled:text-slate-500" autocomplete="off" />
        <p v-if="isEdit" class="mt-1 text-xs text-slate-400">用户名不可修改</p>
      </div>
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">
          密码
          <span v-if="!isEdit" class="text-red-500">*</span>
          <span v-else class="text-slate-400 font-normal">(留空则保持不变)</span>
        </label>
        <input type="password" v-model="formData.password" :placeholder="passwordPlaceholder" class="input" autocomplete="new-password" />
      </div>
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">Home 目录 <span class="text-slate-400 font-normal">(可选)</span></label>
        <input type="text" v-model="formData.homeDirectory" placeholder="留空则使用 基础目录/用户名" class="input" />
        <p class="mt-1 text-xs text-slate-400">相对路径基于"基础目录"，留空则自动创建为 基础目录/用户名</p>
      </div>
      <!-- 模块权限 -->
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">模块权限</label>
        <div class="rounded-lg border border-slate-200 overflow-hidden">
          <table class="w-full text-sm">
            <thead>
              <tr class="bg-slate-50 border-b border-slate-200">
                <th class="px-3 py-2 text-left text-xs font-semibold text-slate-600">模块</th>
                <th class="px-3 py-2 text-center text-xs font-semibold text-slate-600 w-20">无权限</th>
                <th class="px-3 py-2 text-center text-xs font-semibold text-slate-600 w-20">只读</th>
                <th class="px-3 py-2 text-center text-xs font-semibold text-slate-600 w-20">读写</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-100">
              <tr v-for="mod in modules" :key="mod.key" class="hover:bg-slate-50">
                <td class="px-3 py-2">
                  <div class="flex items-center gap-2">
                    <i :class="[mod.icon, 'text-slate-400 w-4 text-center']"></i>
                    <span class="text-slate-700">{{ mod.label }}</span>
                  </div>
                </td>
                <td class="px-3 py-2 text-center">
                  <input type="radio" :name="'perm-' + mod.key" value="" :checked="!formData.permissions[mod.key]" @change="setPermission(mod.key, '')" class="w-4 h-4 accent-slate-400" />
                </td>
                <td class="px-3 py-2 text-center">
                  <input type="radio" :name="'perm-' + mod.key" value="r" :checked="formData.permissions[mod.key] === 'r'" @change="setPermission(mod.key, 'r')" class="w-4 h-4 accent-blue-500" />
                </td>
                <td class="px-3 py-2 text-center">
                  <input type="radio" :name="'perm-' + mod.key" value="rw" :checked="formData.permissions[mod.key] === 'rw'" @change="setPermission(mod.key, 'rw')" class="w-4 h-4 accent-green-500" />
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </form>
  </BaseModal>
</template>
