<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import BaseModal from '@/component/modal.vue'

import type { FileInfo, ExplorerAdapter } from '../types'

@Component({
    expose: ['show'],
    emits: ['success'],
    components: { BaseModal },
})
class RenameModal extends Vue {
    isOpen = false
    loading = false
    adapter: ExplorerAdapter | null = null
    files: FileInfo[] = []
    input = ''

    get trimmedInput() { return this.input.trim() }
    get inputIsDir() { return this.trimmedInput.endsWith('/') }

    // 计算单个文件的目标路径
    resolveTarget(file: FileInfo): string {
        const t = this.trimmedInput
        // 以 / 结尾 → 移入该目录并保持原名
        return this.inputIsDir ? `${t}${file.name}` : t
    }

    // 将相对/绝对 target 还原为逻辑绝对路径（仅用于预览）
    toAbs(target: string, file: FileInfo): string {
        if (!target) return ''
        if (target.startsWith('/')) return target
        const idx = file.path.lastIndexOf('/')
        const dir = idx < 1 ? '/' : file.path.slice(0, idx)
        return dir === '/' ? `/${target}` : `${dir}/${target}`
    }

    get isValidInput() {
        const v = this.trimmedInput
        if (!v || v.includes('\\')) return false
        if (this.files.length > 1 && !this.inputIsDir) return false
        if (this.files.length === 1) {
            const file = this.files[0]
            if (file) {
                const dst = this.toAbs(this.resolveTarget(file), file)
                if (dst && dst === file.path) return false
            }
        }
        return true
    }

    get title() {
        return this.files.length > 1
            ? `重命名 / 移动 ${this.files.length} 项`
            : '重命名 / 移动'
    }

    get previewTargets(): { src: string; abs: string }[] {
        return this.files.map(f => ({
            src: f.path,
            abs: this.isValidInput ? this.toAbs(this.resolveTarget(f), f) : '',
        }))
    }

    show(adapter: ExplorerAdapter, target: FileInfo | FileInfo[]) {
        this.adapter = adapter
        this.files = Array.isArray(target) ? target : [target]
        this.input = this.files.length === 1 ? (this.files[0]?.name ?? '') : ''
        this.isOpen = true
    }

    async handleConfirm() {
        if (!this.isValidInput || this.files.length === 0 || !this.adapter) return
        this.loading = true
        try {
            for (let i = 0; i < this.files.length; i += 5) {
                const batch = this.files.slice(i, i + 5)
                await Promise.all(
                    batch.map(f => {
                        const newPath = this.toAbs(this.resolveTarget(f), f)
                        return this.adapter?.rename(f.path, newPath)
                    })
                )
            }
            this.$emit('success')
            this.isOpen = false
        } finally {
            this.loading = false
        }
    }
}

export default toNative(RenameModal)
</script>

<template>
  <BaseModal v-model="isOpen" :title="title" :loading="loading" :confirm-disabled="!isValidInput" @confirm="handleConfirm">
    <form class="space-y-4" @submit.prevent="handleConfirm">
      <div>
        <label for="fmRenameInput" class="form-label">目标路径</label>
        <div class="relative">
          <div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
            <i :class="inputIsDir ? 'fas fa-folder text-blue-400' : 'fas fa-file-export text-slate-400'"></i>
          </div>
          <input
            id="fmRenameInput"
            v-model="input"
            type="text"
            :disabled="loading"
            autofocus
            class="input pl-9"
            :placeholder="files.length > 1 ? '目标目录，如：/backup/' : '如：new.txt、/backup/new.txt、/backup/'"
          >
        </div>
        <p class="text-xs text-slate-400 mt-1.5">
          以 <code class="bg-slate-100 px-1 rounded">/</code> 结尾 → 移入该目录并保持原名；否则 → 完整目标路径（仅单文件）
        </p>
      </div>

      <div>
        <p class="text-xs font-medium text-slate-500 mb-1.5">操作预览（共 {{ files.length }} 项）</p>
        <div class="rounded-lg border border-slate-200 overflow-hidden">
          <div class="grid grid-cols-[1fr_auto_1fr] bg-slate-100 border-b border-slate-200 px-3 py-1.5 text-xs font-medium text-slate-500">
            <span>原路径</span><span></span><span>目标路径</span>
          </div>
          <div class="max-h-52 overflow-auto divide-y divide-slate-100">
            <div
              v-for="item in previewTargets"
              :key="item.src"
              class="grid grid-cols-[1fr_auto_1fr] items-center gap-2 px-3 py-2 text-xs hover:bg-slate-50"
            >
              <span class="font-mono truncate text-slate-600" :title="item.src">{{ item.src }}</span>
              <i class="fas fa-arrow-right text-slate-300 shrink-0"></i>
              <span
                class="font-mono truncate"
                :class="item.abs ? 'text-blue-600' : 'text-slate-300 italic'"
                :title="item.abs"
              >{{ item.abs || '请输入目标路径' }}</span>
            </div>
          </div>
        </div>
      </div>
    </form>
    <template #confirm-text>{{ loading ? '处理中...' : '确认' }}</template>
  </BaseModal>
</template>
