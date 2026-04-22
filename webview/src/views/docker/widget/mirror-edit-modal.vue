<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    components: { BaseModal },
    emits: ['success']
})
class MirrorEditModal extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    isOpen = false
    modalLoading = false
    mirrors: string[] = []
    newMirror = ''

    show(currentMirrors: string[]) {
        this.mirrors = [...currentMirrors]
        this.newMirror = ''
        this.isOpen = true
    }

    addMirror() {
        let url = this.newMirror.trim()
        if (!url) return
        if (!url.startsWith('http://') && !url.startsWith('https://')) {
            url = 'https://' + url
        }
        if (!url.endsWith('/')) {
            url += '/'
        }
        if (this.mirrors.includes(url)) {
            this.actions.showNotification('warning', '该加速地址已存在')
            return
        }
        this.mirrors.push(url)
        this.newMirror = ''
    }

    removeMirror(index: number) {
        this.mirrors.splice(index, 1)
    }

    async handleConfirm() {
        this.modalLoading = true
        try {
            const res = await api.getSettings()
            const docker = res.payload?.docker || { host: '', containerRoot: '', mirrors: [] }
            docker.mirrors = this.mirrors
            await api.updateAllSettings({ docker })
            this.actions.showNotification('success', '镜像加速器更新成功')
            this.isOpen = false
            this.$emit('success')
        } catch (e) {
            this.actions.showNotification('error', '保存失败')
        }
        this.modalLoading = false
    }
}

export default toNative(MirrorEditModal)
</script>

<template>
  <BaseModal
    v-model="isOpen"
    title="管理镜像加速器"
    :loading="modalLoading"
    show-footer
    @confirm="handleConfirm"
  >
    <div class="space-y-4">
      <p class="text-sm text-slate-500">配置 Docker Hub 镜像加速地址，加速国内镜像拉取。</p>

      <!-- 添加输入框 -->
      <div class="flex gap-2">
        <input
          type="text"
          v-model="newMirror"
          placeholder="例如: https://mirror.example.com"
          class="input flex-1"
          @keydown.enter.prevent="addMirror"
        />
        <button
          type="button"
          class="px-4 py-2 rounded-lg bg-purple-500 hover:bg-purple-600 text-white text-sm font-medium transition-colors flex-shrink-0"
          @click="addMirror"
        >
          添加
        </button>
      </div>

      <!-- 已配置的加速地址列表 -->
      <div v-if="mirrors.length > 0" class="space-y-2">
        <div
          v-for="(mirror, index) in mirrors"
          :key="index"
          class="flex items-center justify-between gap-2 px-3 py-2.5 bg-slate-50 rounded-lg border border-slate-200"
        >
          <div class="flex items-center gap-2 min-w-0">
            <i class="fas fa-bolt text-sky-400 text-sm flex-shrink-0"></i>
            <code class="text-sm text-slate-700 truncate">{{ mirror }}</code>
          </div>
          <button
            type="button"
            class="w-7 h-7 flex items-center justify-center rounded-md text-slate-400 hover:text-red-500 hover:bg-red-50 transition-colors flex-shrink-0"
            @click="removeMirror(index)"
            title="移除"
          >
            <i class="fas fa-times text-xs"></i>
          </button>
        </div>
      </div>

      <div v-else class="text-center py-6 text-slate-400 text-sm">
        暂未配置加速地址
      </div>
    </div>
  </BaseModal>
</template>
