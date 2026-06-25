<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { DockerImageInfo, DockerRegistryInfo } from '@/service/types'

import { formatFileSize, formatTime } from '@/helper/utils'

import Modal from '@/component/modal.vue'
import PageSearch from '@/component/page-search.vue'
import ToggleCard from '@/component/toggle-card.vue'

import ImageBuildModal from './widget/image-build-modal.vue'
import ImagePullModal from './widget/image-pull-modal.vue'
import ImageTagModal from './widget/image-tag-modal.vue'
import RegistryPushModal from './widget/registry-push-modal.vue'

@Component({
    components: { PageSearch, Modal, ImagePullModal, ImageTagModal, ImageBuildModal, RegistryPushModal, ToggleCard }
})
class Images extends Vue {
    portal = usePortal()

    // ─── Refs ───
    @Ref readonly pullModalRef!: InstanceType<typeof ImagePullModal>
    @Ref readonly tagModalRef!: InstanceType<typeof ImageTagModal>
    @Ref readonly buildModalRef!: InstanceType<typeof ImageBuildModal>
    @Ref readonly registryPushModalRef!: InstanceType<typeof RegistryPushModal>

    // ─── 数据属性 ───
    images: DockerImageInfo[] = []
    registries: DockerRegistryInfo[] = []
    loading = false
    showAllImages = false
    searchText = ''
    pruneModalOpen = false
    pruneAll = false
    pruneLoading = false
    formatFileSize = formatFileSize
    formatTime = formatTime

    get filteredImages() {
        const keyword = this.searchText.trim().toLowerCase()
        if (!keyword) return this.images
        return this.images.filter((image: DockerImageInfo) => {
            const tags = (image.repoTags || []).join(' ')
            const digests = (image.repoDigests || []).join(' ')
            return (
                this.getImageName(image).toLowerCase().includes(keyword) ||
                image.id.toLowerCase().includes(keyword) ||
                image.shortId.toLowerCase().includes(keyword) ||
                tags.toLowerCase().includes(keyword) ||
                digests.toLowerCase().includes(keyword)
            )
        })
    }

    // ─── 方法 ───
    async loadImages() {
        this.loading = true
        try {
            const res = await api.dockerImageList(this.showAllImages)
            this.images = res.payload || []
        } finally {
            this.loading = false
        }
    }

    async loadRegistries() {
        try {
            const res = await api.dockerRegistryList()
            this.registries = res.payload || []
        } catch {}
    }

    openPush(image: DockerImageInfo) {
        const tag = image.repoTags.find(t => t !== '<none>:<none>') || ''
        this.registryPushModalRef?.show(this.registries, null, tag)
    }

    // 提取镜像名称（去掉 registry host 和 tag）
    // 例: "registry.com/nginx:latest" → "nginx", "localhost:5000/nginx:v1.0" → "nginx"
    extractImageName(ref: string): string {
        if (!ref || ref === '<none>:<none>') return ''
        // 去掉 digest 或 tag（: 后是 tag，@ 后是 digest；端口号的 : 通过检查 / 来区分）
        let repo = ref
        const atIdx = ref.indexOf('@')
        if (atIdx > 0) {
            repo = ref.substring(0, atIdx)
        } else {
            const colonIdx = ref.lastIndexOf(':')
            if (colonIdx > 0 && !ref.substring(colonIdx).includes('/')) {
                repo = ref.substring(0, colonIdx)
            }
        }
        // 去掉 registry host（第一段含 . 或 : 即为 host）
        const slashIdx = repo.indexOf('/')
        if (slashIdx === -1) return repo
        const first = repo.substring(0, slashIdx)
        return (first.includes('.') || first.includes(':')) ? repo.substring(slashIdx + 1) : repo
    }

    // 获取镜像显示名称：优先用 repoTags，无标签时从 repoDigests 提取
    getImageName(img: DockerImageInfo): string {
        if (img.repoTags && img.repoTags.length > 0) {
            const tag = img.repoTags.find(t => t && t !== '<none>:<none>')
            if (tag) return this.extractImageName(tag)
        }
        if (img.repoDigests && img.repoDigests.length > 0) {
            const digest = img.repoDigests[0]
            const atIndex = digest.indexOf('@')
            if (atIndex > 0) {
                return this.extractImageName(digest.substring(0, atIndex))
            }
        }
        return '<none>'
    }

    async pullImage(image: DockerImageInfo) {
        const tag = image.repoTags.find(t => t && t !== '<none>:<none>')
        if (!tag) {
            this.portal.showNotification('error', '镜像无标签，无法拉取')
            return
        }

        // 解析 registry host（用于匹配私有仓库携带认证信息）
        let tagHost = ''
        let imageName = tag
        const firstSlash = tag.indexOf('/')
        if (firstSlash > 0) {
            const firstPart = tag.substring(0, firstSlash)
            if (firstPart.includes('.') || firstPart.includes(':')) {
                tagHost = firstPart
                imageName = tag.substring(firstSlash + 1)
            }
        }
        const matchedRegistry = this.registries.find(r =>
            r.url === tagHost || r.url.replace(/^https?:\/\//, '') === tagHost
        )

        // 匹配到仓库：传 imageName 让后端拼接；否则传完整 tag
        const pullImageRef = matchedRegistry ? imageName : tag
        const registryUrl = matchedRegistry?.url || ''

        this.portal.showConfirm({
            title: '拉取镜像',
            message: `确定要重新拉取镜像 <strong class="text-slate-900">${tag || image.shortId}</strong> 吗？`,
            icon: 'fa-download',
            iconColor: 'blue',
            confirmText: '确认拉取',
            danger: false,
            onConfirm: async () => {
                try {
                    await api.dockerImagePull(pullImageRef, registryUrl, '')
                    this.portal.showNotification('success', '镜像拉取成功')
                    this.loadImages()
                } catch {}
            }
        })
    }

    handleImageAction(image: DockerImageInfo, action: string) {
        const tag = image.repoTags.find(t => t && t !== '<none>:<none>')
        this.portal.showConfirm({
            title: '删除镜像',
            message: `确定要删除镜像 <strong class="text-slate-900">${tag || image.shortId}</strong> 吗？`,
            icon: 'fa-trash',
            iconColor: 'red',
            confirmText: '确认删除',
            danger: true,
            onConfirm: async () => {
                await api.dockerImageAction(image.id, action)
                this.portal.showNotification('success', '镜像删除成功')
                this.loadImages()
            }
        })
    }

    handleImagePrune() {
        this.pruneAll = false
        this.pruneModalOpen = true
    }

    async confirmPrune() {
        this.pruneLoading = true
        try {
            const { payload } = await api.dockerImagePrune({ all: this.pruneAll })
            this.portal.showNotification('success', `镜像清理完成，回收 ${formatFileSize(payload?.spaceReclaimed || 0)}`)
            this.pruneModalOpen = false
            this.loadImages()
        } finally {
            this.pruneLoading = false
        }
    }

    // ─── 生命周期 ───
    mounted() {
        this.loadImages()
        this.loadRegistries()
    }
}

export default toNative(Images)
</script>

<template>
  <!-- Toolbar Bar -->
  <div class="page">
    <div class="page-toolbar">
      <!-- 桌面端 -->
      <div class="hidden md:flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="page-icon bg-blue-500">
            <i class="fas fa-layer-group text-white"></i>
          </div>
          <div>
            <h1 class="text-lg font-semibold text-slate-800 truncate">镜像管理</h1>
            <p class="text-xs text-slate-500">拉取、导入、导出和删除 Docker 镜像</p>
          </div>
        </div>
        <div class="flex items-center gap-2 flex-shrink-0">
          <PageSearch v-model="searchText" search-key="docker-images" placeholder="搜索镜像名称、标签、ID..." focus-color="blue" type-to-search />
          <div class="tab-group">
            <button :class="['tab-btn', !showAllImages ? 'tab-btn-active text-blue-600' : 'tab-btn-inactive']" @click="showAllImages = false; loadImages()">
              <i class="fas fa-cube"></i><span>顶层</span>
            </button>
            <button :class="['tab-btn', showAllImages ? 'tab-btn-active text-blue-600' : 'tab-btn-inactive']" @click="showAllImages = true; loadImages()">
              <i class="fas fa-layer-group"></i><span>全部</span>
            </button>
          </div>
          <button class="btn btn-secondary" @click="loadImages()">
            <i class="fas fa-rotate"></i>刷新
          </button>
          <button v-if="portal.hasPerm('POST /api/docker/image/prune')" class="btn btn-danger" @click="handleImagePrune()">
            <i class="fas fa-broom"></i>清理
          </button>
          <button v-if="portal.hasPerm('POST /api/docker/image/:id/action')" class="btn btn-blue" @click="buildModalRef?.show()">
            <i class="fas fa-hammer"></i>构建
          </button>
          <button v-if="portal.hasPerm('POST /api/docker/image/pull')" class="btn btn-blue" @click="pullModalRef?.show()">
            <i class="fas fa-download"></i>拉取
          </button>
        </div>
      </div>
      <!-- 移动端 -->
      <div class="block md:hidden">
        <div class="flex items-center justify-between mb-3">
          <div class="flex items-center gap-3 min-w-0 flex-1">
            <div class="page-icon bg-blue-500">
              <i class="fas fa-layer-group text-white"></i>
            </div>
            <div class="min-w-0">
              <h1 class="text-lg font-semibold text-slate-800 truncate">镜像管理</h1>
              <p class="text-xs text-slate-500 truncate">拉取与管理镜像</p>
            </div>
          </div>
          <div class="flex items-center gap-1 flex-shrink-0">
            <button class="btn btn-secondary w-9 h-9 !px-0" title="刷新" @click="loadImages()">
              <i class="fas fa-rotate text-sm"></i>
            </button>
            <button v-if="portal.hasPerm('POST /api/docker/image/prune')" class="btn btn-danger w-9 h-9 !px-0" title="清理镜像" @click="handleImagePrune()">
              <i class="fas fa-broom text-sm"></i>
            </button>
            <button v-if="portal.hasPerm('POST /api/docker/image/:id/action')" class="btn btn-blue w-9 h-9 !px-0" title="构建" @click="buildModalRef?.show()">
              <i class="fas fa-hammer text-sm"></i>
            </button>
            <button v-if="portal.hasPerm('POST /api/docker/image/pull')" class="btn btn-blue w-9 h-9 !px-0" title="拉取" @click="pullModalRef?.show()">
              <i class="fas fa-download text-sm"></i>
            </button>
          </div>
        </div>
        <div class="tab-group w-full justify-center">
          <button :class="['tab-btn flex-1 justify-center', !showAllImages ? 'tab-btn-active text-blue-600' : 'tab-btn-inactive']" @click="showAllImages = false; loadImages()">
            <i class="fas fa-cube"></i><span>顶层</span>
          </button>
          <button :class="['tab-btn flex-1 justify-center', showAllImages ? 'tab-btn-active text-blue-600' : 'tab-btn-inactive']" @click="showAllImages = true; loadImages()">
            <i class="fas fa-layer-group"></i><span>全部</span>
          </button>
        </div>
      </div>
    </div>

    <div class="mobile-search">
      <PageSearch v-model="searchText" search-key="docker-images" placeholder="搜索镜像名称、标签或 ID..." width-class="w-full" focus-color="blue" />
    </div>

    <!-- Loading -->
    <div v-if="loading" class="card-body">
      <div class="empty-state">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>
    </div>

    <!-- Image List -->
    <template v-else-if="filteredImages.length > 0">
      <!-- 桌面端表格视图 -->
      <div class="card-table hidden md:block">
        <table class="w-full border-collapse">
          <thead>
            <tr class="bg-slate-50 border-b border-slate-200">
              <th class="th">镜像</th>
              <th class="th">标签</th>
              <th class="w-32 th">大小</th>
              <th class="w-36 th">创建时间</th>
              <th class="w-40 th-right">操作</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <tr v-for="img in filteredImages" :key="img.id" class="hover:bg-slate-50 transition-colors">
              <td class="px-4 py-3 max-w-[280px]">
                <div class="flex items-center gap-2 min-w-0">
                  <div class="row-icon bg-blue-400">
                    <i class="fas fa-layer-group text-white text-sm"></i>
                  </div>
                  <div class="min-w-0">
                    <router-link v-if="portal.hasPerm('GET /api/docker/image/:id')" :to="'/docker/image/' + img.id" class="font-medium text-slate-800 hover:text-blue-600 transition-colors truncate block">{{ getImageName(img) }}</router-link>
                    <span v-else class="font-medium text-slate-800 truncate block">{{ getImageName(img) }}</span>
                    <code class="text-xs text-slate-400 font-mono truncate block mt-0.5">{{ img.shortId }}</code>
                  </div>
                </div>
              </td>
              <td class="px-4 py-3">
                <div v-if="img.repoTags && img.repoTags.length > 0" class="flex flex-wrap gap-1">
                  <span v-for="(tag, idx) in img.repoTags" :key="idx" class="inline-flex items-center px-1.5 py-0.5 rounded-lg text-xs font-mono bg-blue-50 text-blue-600">{{ tag }}</span>
                </div>
                <span v-else class="text-sm text-slate-400">-</span>
              </td>
              <td class="px-4 py-3 text-sm text-slate-600">{{ formatFileSize(img.size) }}</td>
              <td class="px-4 py-3 whitespace-nowrap text-sm text-slate-600">{{ formatTime(new Date(img.created * 1000).toISOString()) }}</td>
              <td class="px-4 py-3">
                <div class="flex justify-end items-center gap-1">
                  <button v-if="portal.hasPerm('GET /api/docker/image/:id')" class="btn-icon btn-icon-slate" title="查看详情" @click="$router.push('/docker/image/' + img.id)">
                    <i class="fas fa-circle-info text-xs"></i>
                  </button>
                  <button v-if="portal.hasPerm('POST /api/docker/image/:id/action')" class="btn-icon btn-icon-blue" title="打标签" @click="tagModalRef?.show(img)">
                    <i class="fas fa-tag text-xs"></i>
                  </button>
                  <button v-if="portal.hasPerm('POST /api/docker/image/pull')" class="btn-icon btn-icon-blue" title="拉取（更新）" @click="pullImage(img)">
                    <i class="fas fa-download text-xs"></i>
                  </button>
                  <button v-if="portal.hasPerm('POST /api/docker/image/push')" :disabled="registries.length === 0" class="btn-icon btn-icon-indigo disabled:opacity-40 disabled:cursor-not-allowed" :title="registries.length === 0 ? '暂无可用私有仓库' : '推送到仓库'" @click="openPush(img)">
                    <i class="fas fa-upload text-xs"></i>
                  </button>
                  <button v-if="portal.hasPerm('POST /api/docker/image/:id/action')" class="btn-icon btn-icon-red" title="删除" @click="handleImageAction(img, 'remove')">
                    <i class="fas fa-trash text-xs"></i>
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 移动端卡片视图 -->
      <div class="card-body md:hidden space-y-3">
        <div v-for="img in filteredImages" :key="img.id" class="card-interactive">
          <!-- 顶部：镜像信息和图标 -->
          <div class="card-info-row">
            <div class="list-icon bg-blue-400">
              <i class="fas fa-layer-group text-white text-base"></i>
            </div>
            <div class="min-w-0">
              <router-link v-if="portal.hasPerm('GET /api/docker/image/:id')" :to="'/docker/image/' + img.id" class="font-medium text-slate-800 hover:text-blue-600 transition-colors text-sm truncate block">{{ getImageName(img) }}</router-link>
              <span v-else class="font-medium text-slate-800 text-sm truncate block">{{ getImageName(img) }}</span>
              <code class="text-xs text-slate-400 font-mono truncate block mt-0.5">{{ img.shortId }}</code>
            </div>
          </div>

          <!-- 标签 -->
          <div v-if="img.repoTags && img.repoTags.length > 0" class="card-prop-row-start">
            <span class="text-xs text-slate-400 flex-shrink-0 mt-0.5">标签</span>
            <div class="flex flex-wrap gap-1">
              <span v-for="(tag, idx) in img.repoTags" :key="idx" class="inline-flex items-center px-1.5 py-0.5 rounded-lg text-xs font-mono bg-blue-50 text-blue-600">{{ tag }}</span>
            </div>
          </div>

          <!-- 创建时间 -->
          <div class="card-prop-row">
            <span class="text-xs text-slate-400 flex-shrink-0">创建时间</span>
            <span class="text-xs text-slate-500">{{ formatTime(new Date(img.created * 1000).toISOString()) }}</span>
          </div>
          <!-- 大小 -->
          <div class="card-prop-row">
            <span class="text-xs text-slate-400 flex-shrink-0">大小</span>
            <span class="text-xs text-slate-500">{{ formatFileSize(img.size) }}</span>
          </div>
        
          <!-- 底部：操作按钮 -->
          <div class="card-actions">
            <button v-if="portal.hasPerm('GET /api/docker/image/:id')" class="btn-icon btn-icon-slate" title="查看详情" @click="$router.push('/docker/image/' + img.id)">
              <i class="fas fa-circle-info text-xs"></i><span class="text-xs ml-1">详情</span>
            </button>
            <button v-if="portal.hasPerm('POST /api/docker/image/:id/action')" class="btn-icon btn-icon-blue" title="打标签" @click="tagModalRef?.show(img)">
              <i class="fas fa-tag text-xs"></i><span class="text-xs ml-1">标签</span>
            </button>
            <button v-if="portal.hasPerm('POST /api/docker/image/pull')" class="btn-icon btn-icon-blue" title="拉取（更新）" @click="pullImage(img)">
              <i class="fas fa-download text-xs"></i><span class="text-xs ml-1">拉取</span>
            </button>
            <button v-if="portal.hasPerm('POST /api/docker/image/push')" :disabled="registries.length === 0" class="btn-icon btn-icon-indigo disabled:opacity-40 disabled:cursor-not-allowed" :title="registries.length === 0 ? '暂无可用私有仓库' : '推送到仓库'" @click="openPush(img)">
              <i class="fas fa-upload text-xs"></i><span class="text-xs ml-1">推送</span>
            </button>
            <button v-if="portal.hasPerm('POST /api/docker/image/:id/action')" class="btn-icon btn-icon-red" title="删除" @click="handleImageAction(img, 'remove')">
              <i class="fas fa-trash text-xs"></i><span class="text-xs ml-1">删除</span>
            </button>
          </div>
        </div>
      </div>
    </template>

    <div v-else class="card-body">
      <div class="empty-state">
        <div class="empty-state-icon">
          <i class="fas fa-compact-disc text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">{{ images.length === 0 ? '暂无镜像' : '未找到匹配镜像' }}</p>
        <p class="text-sm text-slate-400">{{ images.length === 0 ? '点击「拉取」从 Registry 获取' : '尝试更换关键词或清空搜索条件' }}</p>
      </div>
    </div>
  </div>

  <ImagePullModal ref="pullModalRef" @success="loadImages" />
  <ImageTagModal ref="tagModalRef" @success="loadImages" />
  <ImageBuildModal ref="buildModalRef" @success="loadImages" />
  <RegistryPushModal ref="registryPushModalRef" />

  <!-- 清理镜像确认 -->
  <Modal
    v-model="pruneModalOpen"
    title="清理镜像"
    confirm-class="btn-danger"
    :loading="pruneLoading"
    @confirm="confirmPrune"
  >
    <template #confirm-text>确认清理</template>
    <div class="space-y-4">
      <p class="text-sm text-slate-600">将清理未被任何容器使用的悬空镜像层，不会删除正在被容器使用的镜像。</p>
      <ToggleCard v-model="pruneAll" label="同时清理有标签但未被使用的镜像">
        <template #desc>等同于 <code class="font-mono bg-slate-100 px-1 rounded">docker image prune -a</code>，会删除更多空间</template>
      </ToggleCard>
    </div>
  </Modal>
</template>
