<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { FilerFileInfo } from '@/service/types'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    emits: ['success'],
    components: { BaseModal }
})
class MoveModal extends Vue {
    // ─── 数据属性 ───
    isOpen = false
    loading = false
    targetPathInput = ''
    files: FilerFileInfo[] = []

    // ─── 计算属性 ───
    get targetPathText() {
        return this.targetPathInput.trim().replace(/\/+$/, '')
    }

    get previewText() {
        return this.files.map(file => file.name).join('、')
    }

    get basePath() {
        const path = this.files[0]?.path || ''
        const index = path.lastIndexOf('/')
        return index <= 0 ? '/' : path.slice(0, index)
    }

    get targetPath() {
        if (!this.targetPathText) return ''
        if (this.targetPathText.startsWith('/')) return this.targetPathText
        return this.resolvePath(this.targetPathText) || '由后端校验'
    }

    get isValidTargetPath() {
        return !!this.targetPathText && !this.targetPathText.includes('\\')
    }

    // ─── 方法 ───
    show(files: FilerFileInfo[]) {
        this.files = files
        this.targetPathInput = ''
        this.isOpen = true
    }

    resolvePath(path: string) {
        const parts = this.basePath.split('/').filter(Boolean)
        for (const part of path.split('/')) {
            if (!part || part === '.') continue
            if (part === '..') {
                if (parts.length === 0) return ''
                parts.pop()
                continue
            }
            parts.push(part)
        }
        return '/' + parts.join('/')
    }

    async handleConfirm() {
        if (this.files.length === 0 || !this.isValidTargetPath) return
        this.loading = true
        try {
            for (let index = 0; index < this.files.length; index += 5) {
                const batch = this.files.slice(index, index + 5)
                await Promise.all(batch.map(file => api.filerRename(file.path, `${this.targetPathText}/${file.name}`)))
            }
            this.$emit('success')
            this.isOpen = false
        } finally {
            this.loading = false
        }
    }
}

export default toNative(MoveModal)
</script>

<template>
  <BaseModal ref="modalRef" v-model="isOpen" title="移动到新目录" :loading="loading" confirm-class="btn-blue" :confirm-disabled="!isValidTargetPath || files.length === 0" @confirm="handleConfirm">
    <div class="text-center py-6">
      <div class="empty-state-icon bg-blue-100 mx-auto">
        <i class="fas fa-folder-plus text-3xl text-blue-500"></i>
      </div>
      <p class="text-lg text-slate-700 mb-2">
        将选中的 <strong class="text-slate-900">{{ files.length }}</strong> 项移动到新目录
      </p>
      <p class="text-sm text-slate-500">目标路径由后端按用户目录校验，越界移动会被拒绝</p>
    </div>

    <form class="max-w-3xl space-y-4" @submit.prevent="handleConfirm">
      <div>
        <label for="moveTargetPath" class="form-label">
          新目录路径
        </label>
        <div class="relative">
          <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
            <i class="fas fa-folder text-slate-400"></i>
          </div>
          <input id="moveTargetPath" v-model="targetPathInput" type="text" :disabled="loading" required class="input pl-11" placeholder="请输入路径，如：output 或 tmp/output">
        </div>
        <p class="text-xs text-slate-400 mt-1">支持相对路径或绝对路径，最终目标：{{ targetPath || '无效路径' }}</p>
      </div>
    </form>

    <p v-if="files.length > 0" class="detail-value-mono mt-4">
      {{ previewText }}
    </p>

    <template #confirm-text>
      {{ loading ? '移动中...' : '确认移动' }}
    </template>
  </BaseModal>
</template>
