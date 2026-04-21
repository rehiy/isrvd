<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { SystemMemberInfo, SystemMemberUpsertRequest } from '@/service/types'
import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

import BaseModal from '@/component/modal.vue'

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
    // 编辑时保存原始用户名，用于后端定位
    originalUsername = ''
    passwordSet = false
    formData: SystemMemberUpsertRequest = { username: '', password: '', homeDirectory: '', allowTerminal: false }

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
            this.formData = {
                username: member.username,
                password: '',
                homeDirectory: member.homeDirectory,
                allowTerminal: member.allowTerminal
            }
        } else {
            this.originalUsername = ''
            this.passwordSet = false
            this.formData = { username: '', password: '', homeDirectory: '', allowTerminal: false }
        }
        this.isOpen = true
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
      <div class="flex items-center gap-2">
        <input id="allowTerminalSwitch" type="checkbox" v-model="formData.allowTerminal" class="w-4 h-4" />
        <label for="allowTerminalSwitch" class="text-sm text-slate-700">允许访问 Shell 终端</label>
      </div>
    </form>
  </BaseModal>
</template>
