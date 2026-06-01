<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { SSHHostInfo, SSHHostUpsert, SSHCredentialInfo } from '@/service/types'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    components: { BaseModal },
    emits: ['success']
})
class HostEditModal extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    isOpen = false
    modalLoading = false
    editId = ''
    formData: SSHHostUpsert = { name: '', addr: '', user: '', password: '', privateKey: '', description: '', credentialId: '' }
    credentials: SSHCredentialInfo[] = []
    credentialsLoading = false

    // ─── 计算属性 ───
    get isEdit() {
        return this.editId !== ''
    }

    get title() {
        return this.isEdit ? '编辑 SSH 主机' : '添加 SSH 主机'
    }

    // ─── 方法 ───
    async loadCredentials() {
        if (this.credentials.length > 0) return // 已加载过
        this.credentialsLoading = true
        try {
            const res = await api.sshCredentialList()
            this.credentials = res.payload || []
        } catch {
            this.portal.showNotification('error', '加载凭据列表失败')
        } finally {
            this.credentialsLoading = false
        }
    }

    onCredentialChange() {
        const cred = this.credentials.find(c => c.id === this.formData.credentialId)
        if (cred) {
            this.formData.user = cred.user
            this.formData.password = ''
            this.formData.privateKey = ''
        } else {
            this.formData.user = ''
        }
    }

    show(host: SSHHostInfo | null = null) {
        this.editId = ''
        this.formData = { name: '', addr: '', user: '', password: '', privateKey: '', description: '', credentialId: '' }
        this.loadCredentials()

        if (host) {
            this.editId = host.id
            this.formData.name = host.name
            this.formData.addr = host.addr
            this.formData.description = host.description || ''
            this.formData.credentialId = host.credentialId || ''
            this.formData.user = host.user || ''
            this.formData.password = ''
            this.formData.privateKey = ''
        }
        
        this.isOpen = true
    }

    async handleConfirm() {
        if (!this.formData.name?.trim() || !this.formData.addr?.trim()) return
        if (!this.formData.credentialId && !this.formData.user?.trim()) return
        
        this.modalLoading = true
        try {
            const submitData: SSHHostUpsert = {
                name: this.formData.name,
                addr: this.formData.addr,
                description: this.formData.description,
            }
            
            if (this.formData.credentialId) {
                submitData.credentialId = this.formData.credentialId
            } else {
                submitData.user = this.formData.user
                submitData.password = this.formData.password
                submitData.privateKey = this.formData.privateKey
            }
            
            if (this.isEdit) {
                await api.sshHostUpdate(this.editId, submitData)
                this.portal.showNotification('success', '主机更新成功')
            } else {
                await api.sshHostCreate(submitData)
                this.portal.showNotification('success', '主机添加成功')
            }
            this.isOpen = false
            this.$emit('success')
        } catch {}
        this.modalLoading = false
    }
}

export default toNative(HostEditModal)
</script>

<template>
  <BaseModal v-model="isOpen" :title="title" :loading="modalLoading" confirm-class="btn-emerald" show-footer @confirm="handleConfirm">
    <form class="space-y-4" @submit.prevent="handleConfirm">
      <div>
        <label class="form-label">名称 <span class="text-red-500">*</span></label>
        <input v-model="formData.name" type="text" placeholder="请输入主机名称" required class="input" />
      </div>
      <div>
        <label class="form-label">地址 <span class="text-red-500">*</span></label>
        <input v-model="formData.addr" type="text" placeholder="请输入主机地址" required class="input" />
        <p class="text-xs text-slate-400 mt-1">格式：主机名或 IP [:端口]，如 192.168.1.100:2222，默认端口 22</p>
      </div>
      <div>
        <label class="form-label">描述 <span class="text-slate-400 font-normal">(可选)</span></label>
        <input v-model="formData.description" type="text" placeholder="请输入主机描述" class="input" />
      </div>

      <!-- 认证信息 -->
      <div class="pt-2 border-t border-slate-100 space-y-3">
        <div>
          <label class="form-label">选择凭据</label>
          <select v-model="formData.credentialId" class="input" :disabled="credentialsLoading" @change="onCredentialChange">
            <option value="">{{ credentialsLoading ? '加载中...' : '手动输入' }}</option>
            <option v-for="cred in credentials" :key="cred.id" :value="cred.id">
              {{ cred.name }} ({{ cred.user }})
            </option>
          </select>
        </div>
        <div v-if="formData.credentialId">
          <label class="form-label">用户名</label>
          <input :value="credentials.find(c => c.id === formData.credentialId)?.user || ''" type="text" class="input bg-slate-50" disabled />
          <p class="text-xs text-slate-400 mt-1">使用凭据中保存的用户名</p>
        </div>
      </div>

      <!-- 手动输入认证信息 -->
      <div v-if="!formData.credentialId" class="space-y-3">
        <div>
          <label class="form-label">用户名 <span class="text-red-500">*</span></label>
          <input v-model="formData.user" type="text" placeholder="请输入用户名" required class="input" autocomplete="off" />
        </div>
        <div>
          <label class="form-label">密码 <span class="text-slate-400 font-normal">(可选)</span></label>
          <input v-model="formData.password" type="password" :placeholder="isEdit ? '留空则保持不变' : '请输入登录密码'" class="input" autocomplete="new-password" />
        </div>
        <div>
          <label class="form-label">SSH 私钥 <span class="text-slate-400 font-normal">(可选，优先于密码)</span></label>
          <textarea
            v-model="formData.privateKey"
            rows="5"
            :placeholder="isEdit ? '留空则保持不变' : '请输入 SSH 私钥'"
            class="input font-mono text-xs"
          />
          <p class="text-xs text-slate-400 mt-1">PEM 格式私钥，以 "-----BEGIN" 开头，设置后优先使用私钥认证</p>
        </div>
      </div>
    </form>

    <template #confirm-text>{{ isEdit ? '保存修改' : '确认添加' }}</template>
  </BaseModal>
</template>
