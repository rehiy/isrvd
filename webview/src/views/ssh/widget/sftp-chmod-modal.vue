<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { SFTPFileInfo } from '@/service/types'

import { joinPath } from '@/helper/utils'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    components: { BaseModal }
})
class SftpChmodModal extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    isOpen = false
    loading = false
    hostId = ''
    formData = { path: '', mode: '' }

    // ─── 方法 ───
    show(hostId: string, file: SFTPFileInfo, basePath: string) {
        this.hostId = hostId
        this.formData.path = joinPath(basePath, file.name)
        this.formData.mode = this.extractPermissionFromMode(file.mode)
        this.isOpen = true
    }

    async handleConfirm() {
        if (!this.formData.mode.trim()) return
        
        // 验证权限格式
        if (!/^[0-7]{3,4}$/.test(this.formData.mode)) {
            this.portal.showNotification('error', '权限格式无效，请输入 3-4 位八进制数字（如 755、644）')
            return
        }

        this.loading = true
        try {
            await api.sftpFileChmod(this.hostId, { path: this.formData.path, mode: this.formData.mode })
            this.portal.showNotification('success', '权限修改成功')
            this.isOpen = false
            // 触发父组件刷新
            this.$emit('success')
        } catch (e: unknown) {
            this.portal.showNotification('error', '权限修改失败: ' + (e instanceof Error ? e.message : ''))
        } finally {
            this.loading = false
        }
    }

    // 从 mode 字符串提取权限数字（如从 "-rw-r--r--" 提取 "644"）
    extractPermissionFromMode(mode: string): string {
        if (mode.length < 10) return ''
        
        // 解析 rwx 权限为数字
        const parseRwx = (str: string): number => {
            let val = 0
            if (str[0] === 'r') val += 4
            if (str[1] === 'w') val += 2
            if (str[2] === 'x' || str[2] === 's' || str[2] === 't') val += 1
            return val
        }
        
        const owner = parseRwx(mode.substr(1, 3))
        const group = parseRwx(mode.substr(4, 3))
        const other = parseRwx(mode.substr(7, 3))
        
        return `${owner}${group}${other}`
    }
}

export default toNative(SftpChmodModal)
</script>

<template>
  <BaseModal ref="modalRef" v-model="isOpen" title="修改权限" :loading="loading" :confirm-disabled="!formData.mode.trim()" @confirm="handleConfirm">
    <form @submit.prevent="handleConfirm">
      <div>
        <label for="fileMode" class="form-label">
          权限 (八进制)
        </label>
        <div class="relative">
          <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
            <i class="fas fa-unlock text-slate-400"></i>
          </div>
          <input id="fileMode" v-model="formData.mode" type="text" :disabled="loading" required placeholder="请输入文件权限" class="input pl-11">
        </div>
        <p class="text-xs text-slate-400 mt-1">三位八进制数，例如：755、644</p>
        <div class="mt-3 p-4 bg-slate-50 rounded-xl border border-slate-200">
          <p class="text-sm font-medium text-slate-700 mb-2">常用权限:</p>
          <div class="flex flex-wrap gap-2">
            <span class="badge-primary">755 - rwxr-xr-x</span>
            <span class="badge-primary">644 - rw-r--r--</span>
            <span class="badge-warning">777 - rwxrwxrwx</span>
          </div>
        </div>
      </div>
    </form>

    <template #confirm-text>
      {{ loading ? '修改中...' : '确认修改' }}
    </template>
  </BaseModal>
</template>
