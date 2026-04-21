<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { DockerImageSearchResult, DockerRegistryInfo } from '@/service/types'
import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    components: { BaseModal },
    emits: ['success']
})
class ImagePullModal extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    // ─── 数据属性 ───
    isOpen = false
    modalLoading = false
    // source 为 '' 时表示 Docker Hub/默认源；否则为私有仓库 URL
    formData = { source: '', image: '', tag: '', namespace: '' }
    registries: DockerRegistryInfo[] = []
    searchResults: DockerImageSearchResult[] = []
    searchLoading = false
    searchKeyword = ''
    daemonMirrors: string[] = []
    indexServerAddress = ''

    // ─── 计算属性 ───
    get isRegistryMode() {
        return this.formData.source !== ''
    }

    // ─── 方法 ───
    async loadDaemonInfo() {
        try {
            const res = await api.dockerInfo()
            const info = res.payload
            this.daemonMirrors = info?.registryMirrors || []
            this.indexServerAddress = info?.indexServerAddress || ''
        } catch (e) {}
    }

    async loadRegistries() {
        try {
            const res = await api.listRegistries()
            this.registries = res.payload || []
        } catch (e) {}
    }

    show(source = '') {
        this.formData = { source, image: '', tag: '', namespace: '' }
        this.searchResults = []
        this.searchKeyword = ''
        this.loadDaemonInfo()
        this.loadRegistries()
        this.isOpen = true
    }

    async handleConfirm() {
        if (!this.formData.image.trim()) return
        this.modalLoading = true
        try {
            if (this.isRegistryMode) {
                const imageRef = this.formData.tag.trim()
                    ? `${this.formData.image.trim()}:${this.formData.tag.trim()}`
                    : this.formData.image.trim()
                await api.pullFromRegistry(imageRef, this.formData.source, this.formData.namespace.trim())
            } else {
                await api.pullImage(this.formData.image, this.formData.tag)
            }
            this.actions.showNotification('success', '镜像拉取成功')
            this.isOpen = false
            this.$emit('success')
        } catch (e) {}
        this.modalLoading = false
    }

    async handleSearchImage() {
        if (!this.searchKeyword.trim()) return
        this.searchLoading = true
        try {
            const res = await api.imageSearch(this.searchKeyword.trim())
            this.searchResults = res.payload || []
        } catch (e) {
            this.actions.showNotification('error', '搜索镜像失败')
        }
        this.searchLoading = false
    }

    selectSearchResult(item: DockerImageSearchResult) {
        this.formData.image = item.name
        this.formData.tag = 'latest'
        this.searchResults = []
        this.searchKeyword = ''
    }
}

export default toNative(ImagePullModal)
</script>

<template>
  <BaseModal
    ref="modalRef"
    v-model="isOpen"
    title="拉取镜像"
    :loading="modalLoading"
    show-footer
    @confirm="handleConfirm"
  >
    <form @submit.prevent="handleConfirm" class="space-y-4">
      <!-- 源选择 -->
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">镜像源</label>
        <select v-model="formData.source" class="input">
          <option value="">Docker Hub（默认）</option>
          <option v-for="reg in registries" :key="reg.url" :value="reg.url">
            {{ reg.name }} ({{ reg.url }}){{ reg.description ? ' - ' + reg.description : '' }}
          </option>
        </select>
      </div>

      <!-- Docker Hub 模式：镜像搜索 -->
      <template v-if="!isRegistryMode">
        <div>
          <label class="block text-sm font-medium text-slate-700 mb-2">搜索镜像</label>
          <div class="flex gap-2">
            <input type="text" v-model="searchKeyword" placeholder="搜索 Docker Hub 镜像" class="input flex-1" @keyup.enter="handleSearchImage" />
            <button type="button" @click="handleSearchImage" :disabled="searchLoading" class="px-4 py-2 rounded-lg bg-slate-700 hover:bg-slate-800 text-white text-xs font-medium flex items-center gap-1.5 transition-colors disabled:opacity-50">
              <i :class="['fas', searchLoading ? 'fa-spinner fa-spin' : 'fa-search']"></i>
              {{ searchLoading ? '搜索中' : '搜索' }}
            </button>
          </div>
        </div>

        <div v-if="searchResults.length > 0" class="border border-slate-200 rounded-xl max-h-48 overflow-y-auto">
          <div 
            v-for="item in searchResults" 
            :key="item.name" 
            @click="selectSearchResult(item)" 
            class="px-4 py-2.5 hover:bg-blue-50 cursor-pointer border-b border-slate-100 last:border-b-0 transition-colors"
          >
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <i v-if="item.isOfficial" class="fas fa-certificate text-blue-500 text-xs" title="官方镜像"></i>
                <span class="text-sm font-medium text-slate-800">{{ item.name }}</span>
              </div>
              <div class="flex items-center gap-2">
                <span v-if="item.isOfficial" class="inline-flex items-center px-1.5 py-0.5 rounded-lg text-xs font-medium bg-blue-100 text-blue-700">官方</span>
                <span class="text-xs text-slate-400"><i class="fas fa-star text-amber-400 mr-0.5"></i>{{ item.starCount }}</span>
              </div>
            </div>
            <p v-if="item.description" class="text-xs text-slate-500 mt-0.5 truncate">{{ item.description }}</p>
          </div>
        </div>
      </template>

      <!-- 私有仓库模式：命名空间 -->
      <div v-if="isRegistryMode">
        <label class="block text-sm font-medium text-slate-700 mb-2">命名空间 <span class="text-slate-400 font-normal">(可选)</span></label>
        <input type="text" v-model="formData.namespace" placeholder="例如: myteam" class="input" />
      </div>

      <div class="border-t border-slate-200 pt-4">
        <div class="mb-4">
          <label class="block text-sm font-medium text-slate-700 mb-2">镜像名称</label>
          <input type="text" v-model="formData.image" placeholder="例如: nginx, redis:alpine, ubuntu:22.04" required class="input" />
        </div>
        <div>
          <label class="block text-sm font-medium text-slate-700 mb-2">Tag（可选）</label>
          <input type="text" v-model="formData.tag" placeholder="默认 latest" class="input" />
        </div>
        <p v-if="isRegistryMode" class="mt-2 text-xs text-slate-400">
          将拉取: {{ formData.source }}/{{ formData.namespace ? formData.namespace + '/' : '' }}{{ formData.image || 'image' }}{{ formData.tag ? ':' + formData.tag : '' }}
        </p>
      </div>

      <!-- 镜像源提示（仅 Docker Hub 模式） -->
      <div v-if="!isRegistryMode" class="border-t border-slate-200 pt-4">
        <p class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-2">当前镜像源</p>
        <div v-if="daemonMirrors.length > 0" class="flex flex-wrap gap-1.5">
          <code
            v-for="mirror in daemonMirrors"
            :key="mirror"
            class="inline-flex items-center gap-1 px-2 py-1 bg-sky-50 border border-sky-200 rounded-lg text-xs font-mono text-sky-700"
          >
            <i class="fas fa-bolt text-sky-400 text-xs"></i>{{ mirror }}
          </code>
        </div>
        <div v-else class="flex items-center gap-1.5 text-xs text-slate-400">
          <i class="fab fa-docker text-blue-400"></i>
          {{ indexServerAddress || 'https://index.docker.io/v1/' }}
          <span class="text-slate-300">（未配置加速器）</span>
        </div>
      </div>
    </form>
  </BaseModal>
</template>
