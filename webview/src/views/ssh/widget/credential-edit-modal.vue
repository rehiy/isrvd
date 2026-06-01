<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { SSHCredentialInfo, SSHCredentialUpsert } from '@/service/types'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    components: { BaseModal },
    emits: ['success']
})
class CredentialEditModal extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    isOpen = false
    modalLoading = false
    editId = ''
    formData: SSHCredentialUpsert = { name: '', description: '', user: '', password: '', privateKey: '' }

    // ─── 计算属性 ───
    get isEdit() {
        return this.editId !== ''
    }

    get title() {
        return this.isEdit ? '编辑 SSH 凭据' : '添加 SSH 凭据'
    }

    // ─── 方法 ───
    async show(cred: SSHCredentialInfo | null = null) {
        if (cred) {
            this.editId = cred.id
            this.formData = { name: cred.name, description: cred.description, user: cred.user, password: '', privateKey: '' }
            this.isOpen = true
            this.modalLoading = true
            try {
                const res = await api.sshCredential(cred.id)
                const detail = res.payload
                if (!detail) throw new Error('凭据详情为空')
                this.formData = {
                    name: detail.name,
                    description: detail.description,
                    user: detail.user,
                    password: detail.password || '',
                    privateKey: detail.privateKey || '',
                }
            } catch {
                this.portal.showNotification('error', '加载凭据详情失败')
            } finally {
                this.modalLoading = false
            }
            return
        }

        this.editId = ''
        this.formData = { name: '', description: '', user: '', password: '', privateKey: '' }
        this.isOpen = true
    }

    async handleConfirm() {
        if (!this.formData.name?.trim() || !this.formData.user?.trim()) return
        this.modalLoading = true
        try {
            if (this.isEdit) {
                await api.sshCredentialUpdate(this.editId, this.formData)
                this.portal.showNotification('success', '凭据更新成功')
            } else {
                await api.sshCredentialCreate(this.formData)
                this.portal.showNotification('success', '凭据添加成功')
            }
            this.isOpen = false
            this.$emit('success')
        } catch {}
        this.modalLoading = false
    }
}

export default toNative(CredentialEditModal)
</script>

<template>
  <BaseModal v-model="isOpen" :title="title" :loading="modalLoading" confirm-class="btn-purple" show-footer @confirm="handleConfirm">
    <form class="space-y-4" @submit.prevent="handleConfirm">
      <div>
        <label class="form-label">凭据名称 <span class="text-red-500">*</span></label>
        <input v-model="formData.name" type="text" placeholder="请输入凭据名称" required class="input" />
        <p class="text-xs text-slate-400 mt-1">便于识别的名称，如 "生产环境密钥"</p>
      </div>
      <div>
        <label class="form-label">描述 <span class="text-slate-400 font-normal">(可选)</span></label>
        <input v-model="formData.description" type="text" placeholder="请输入凭据描述" class="input" />
      </div>
      <div>
        <label class="form-label">用户名 <span class="text-red-500">*</span></label>
        <input v-model="formData.user" type="text" placeholder="请输入用户名" required class="input" autocomplete="off" />
        <p class="text-xs text-slate-400 mt-1">SSH 登录用户名</p>
      </div>
      <div>
        <label class="form-label">密码 <span class="text-slate-400 font-normal">(与私钥二选一)</span></label>
        <input v-model="formData.password" type="password" :placeholder="isEdit ? '留空则保持不变' : '请输入登录密码'" class="input" autocomplete="new-password" />
      </div>
      <div>
        <label class="form-label">SSH 私钥 <span class="text-slate-400 font-normal">(优先于密码)</span></label>
        <textarea
          v-model="formData.privateKey"
          rows="5"
          :placeholder="isEdit ? '留空则保持不变' : '请输入 SSH 私钥'"
          class="input font-mono text-xs"
        />
        <p class="text-xs text-slate-400 mt-1">PEM 格式私钥，设置后优先使用私钥认证</p>
      </div>
    </form>

    <template #confirm-text>{{ isEdit ? '保存修改' : '确认添加' }}</template>
  </BaseModal>
</template>
