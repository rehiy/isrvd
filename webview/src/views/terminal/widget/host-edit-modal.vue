<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { SSHHostInfo, SSHHostUpsert } from '@/service/types'

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
    formData: SSHHostUpsert = { name: '', addr: '', user: '', password: '', privateKey: '', description: '' }

    // ─── 计算属性 ───
    get isEdit() {
        return this.editId !== ''
    }

    get title() {
        return this.isEdit ? '编辑 SSH 主机' : '添加 SSH 主机'
    }

    // ─── 方法 ───
    show(host: SSHHostInfo | null = null) {
        if (host) {
            this.editId = host.id
            this.formData = {
                name: host.name,
                addr: host.addr,
                user: host.user,
                password: '',
                privateKey: host.privateKey || '',
                description: host.description
            }
        } else {
            this.editId = ''
            this.formData = { name: '', addr: '', user: '', password: '', privateKey: '', description: '' }
        }
        this.isOpen = true
    }

    async handleConfirm() {
        if (!this.formData.name?.trim() || !this.formData.addr?.trim() || !this.formData.user?.trim()) return
        this.modalLoading = true
        try {
            if (this.isEdit) {
                await api.sshHostUpdate(this.editId, this.formData)
                this.portal.showNotification('success', '主机更新成功')
            } else {
                await api.sshHostCreate(this.formData)
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
        <input v-model="formData.name" type="text" placeholder="例如: 生产服务器" required class="input" />
      </div>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
        <div>
          <label class="form-label">地址 <span class="text-red-500">*</span></label>
          <input v-model="formData.addr" type="text" placeholder="host 或 host:port" required class="input" />
          <p class="text-xs text-slate-400 mt-1">默认端口 22</p>
        </div>
        <div>
          <label class="form-label">用户名 <span class="text-red-500">*</span></label>
          <input v-model="formData.user" type="text" placeholder="例如: root" required class="input" autocomplete="off" />
        </div>
      </div>
      <div>
        <label class="form-label">描述 <span class="text-slate-400 font-normal">(可选)</span></label>
        <input v-model="formData.description" type="text" placeholder="主机用途说明" class="input" />
      </div>
      <div>
        <label class="form-label">密码 <span class="text-slate-400 font-normal">(可选)</span></label>
        <input v-model="formData.password" type="password" :placeholder="isEdit ? '留空保持不变' : '登录密码'" class="input" autocomplete="new-password" />
      </div>
      <div>
        <label class="form-label">SSH 私钥 <span class="text-slate-400 font-normal">(可选，优先于密码)</span></label>
        <textarea
          v-model="formData.privateKey"
          rows="5"
          :placeholder="isEdit ? '留空保持不变' : '-----BEGIN OPENSSH PRIVATE KEY-----\n...\n-----END OPENSSH PRIVATE KEY-----'"
          class="input font-mono text-xs"
        />
        <p class="text-xs text-slate-400 mt-1">PEM 格式私钥，设置后优先使用私钥认证</p>
      </div>
    </form>

    <template #confirm-text>{{ isEdit ? '保存修改' : '确认添加' }}</template>
  </BaseModal>
</template>
