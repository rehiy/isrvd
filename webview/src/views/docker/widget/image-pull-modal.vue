<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { DockerImageSearchResult, DockerRegistryInfo } from '@/service/types'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    components: { BaseModal },
    emits: ['success']
})
class ImagePullModal extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    isOpen = false
    modalLoading = false
    // source 为 '' 时表示 Docker Hub/默认源；否则为私有仓库 URL
    formData = { source: '', image: '' }
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

    get pullPreview() {
        const image = this.formData.image.trim() || 'image:latest'
        if (!this.isRegistryMode) return image
        return `${this.formData.source}/${image}`
    }

    // ─── 方法 ───
    async loadDaemonInfo() {
        try {
            const res = await api.dockerInfo()
            const info = res.payload
            this.daemonMirrors = info?.registryMirrors || []
            this.indexServerAddress = info?.indexServerAddress || ''
        } catch {}
    }

    async loadRegistries() {
        try {
            const res = await api.dockerRegistryList()
            this.registries = res.payload || []
        } catch {}
    }

    show(source = '') {
        this.formData = { source, image: '' }
        this.searchResults = []
        this.searchKeyword = ''
        this.loadDaemonInfo()
        this.loadRegistries()
        this.isOpen = true
    }

    async handleConfirm() {
        const imageRef = this.formData.image.trim()
        if (!imageRef) return
        this.modalLoading = true
        try {
            await api.dockerImagePull(imageRef, this.formData.source, '')
            this.portal.showNotification('success', '镜像拉取成功')
            this.isOpen = false
            this.$emit('success')
        } catch {
        } finally {
            this.modalLoading = false
        }
    }

    async handleSearchImage() {
        const keyword = this.searchKeyword.trim()
        if (!keyword) {
            this.searchResults = []
            return
        }
        this.searchLoading = true
        try {
            const res = await api.dockerImageSearch(keyword)
            this.searchResults = res.payload || []
        } catch {
            this.portal.showNotification('error', '搜索镜像失败')
        } finally {
            this.searchLoading = false
        }
    }

    selectSearchResult(item: DockerImageSearchResult) {
        this.formData.image = item.name
        this.searchResults = []
        this.searchKeyword = ''
    }
}

export default toNative(ImagePullModal)
</script>

<template>
  <BaseModal ref="modalRef" v-model="isOpen" title="拉取镜像" :loading="modalLoading" confirm-class="btn-blue" show-footer @confirm="handleConfirm">
    <form class="max-w-3xl space-y-4" @submit.prevent="handleConfirm">
      <section>
        <div class="space-y-3">
          <div>
            <label class="form-label">镜像源</label>
            <select v-model="formData.source" class="input">
              <option value="">Docker Hub（默认）</option>
              <option v-for="reg in registries" :key="reg.url" :value="reg.url">
                {{ reg.name }} ({{ reg.url }}){{ reg.description ? ' - ' + reg.description : '' }}
              </option>
            </select>
          </div>

          <div v-if="!isRegistryMode" class="text-xs text-slate-500">
            <div v-if="daemonMirrors.length > 0" class="flex flex-wrap gap-1.5">
              <code
                v-for="mirror in daemonMirrors"
                :key="mirror"
                class="inline-flex items-center gap-1 px-2 py-1 bg-slate-50 border border-slate-200 rounded-lg text-xs font-mono text-slate-600"
              >
                <i class="fas fa-bolt text-slate-400 text-xs"></i>{{ mirror }}
              </code>
            </div>
            <div v-else class="flex items-center gap-1.5 text-slate-500">
              {{ indexServerAddress || 'https://index.docker.io/v1/' }}
              <span class="text-slate-400">（未配置加速器）</span>
            </div>
          </div>
        </div>
      </section>

      <section v-if="!isRegistryMode">
        <div class="space-y-3">
          <div class="flex gap-2">
            <input v-model="searchKeyword" type="text" placeholder="请输入要搜索的镜像名称" class="input flex-1" @keydown.enter.prevent="handleSearchImage" />
            <button type="button" :disabled="searchLoading" class="btn h-[46px] btn-secondary" @click="handleSearchImage">
              <i :class="['fas', searchLoading ? 'fa-spinner fa-spin' : 'fa-search']"></i>
              {{ searchLoading ? '搜索中' : '搜索' }}
            </button>
          </div>

          <div v-if="searchResults.length > 0" class="max-h-52 overflow-y-auto rounded-xl border border-slate-200 bg-white p-1 shadow-sm">
            <button
              v-for="item in searchResults"
              :key="item.name"
              type="button"
              class="w-full rounded-lg px-3 py-2.5 text-left hover:bg-slate-50 transition-colors"
              @click="selectSearchResult(item)"
            >
              <div class="flex items-center justify-between gap-3">
                <div class="min-w-0 flex items-center gap-2">
                  <i v-if="item.isOfficial" class="fas fa-certificate text-slate-400 text-xs flex-shrink-0" title="官方镜像"></i>
                  <span class="text-sm font-medium text-slate-800 truncate">{{ item.name }}</span>
                </div>
                <div class="flex items-center gap-2 flex-shrink-0">
                  <span v-if="item.isOfficial" class="inline-flex items-center px-1.5 py-0.5 rounded-lg text-xs font-medium bg-slate-100 text-slate-600">官方</span>
                  <span class="text-xs text-slate-400"><i class="fas fa-star text-slate-400 mr-0.5"></i>{{ item.starCount }}</span>
                </div>
              </div>
              <p v-if="item.description" class="text-xs text-slate-500 mt-1 truncate">{{ item.description }}</p>
            </button>
          </div>
        </div>
      </section>

      <section>
        <div class="space-y-4">
          <div>
            <label class="form-label">镜像引用</label>
            <input v-model="formData.image" type="text" placeholder="请输入镜像引用" required class="input font-mono" />
            <p class="text-xs text-slate-400 mt-1">
              未指定 tag 时默认拉取 latest；选择私有仓库时不需要重复填写仓库地址。
            </p>
          </div>

          <div class="border-l-2 border-slate-300 bg-slate-50 px-3 py-2.5">
            <p class="text-xs font-semibold text-slate-600 mb-1">最终将拉取</p>
            <code class="block text-sm text-slate-700 font-mono break-all">{{ pullPreview }}</code>
          </div>
        </div>
      </section>
    </form>

    <template #confirm-text>开始拉取</template>
  </BaseModal>
</template>
