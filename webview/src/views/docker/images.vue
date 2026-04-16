<script lang="ts">
import { Component, Inject, Ref, Vue, toNative } from 'vue-facing-decorator'

import { formatFileSize, formatTime } from '@/helper/utils'
import api from '@/service/api'
import { APP_ACTIONS_KEY } from '@/store/state'

import ImagePullModal from '@/views/docker/widget/image-pull-modal.vue'
import ImageTagModal from '@/views/docker/widget/image-tag-modal.vue'
import ImageBuildModal from '@/views/docker/widget/image-build-modal.vue'

@Component({
    components: { ImagePullModal, ImageTagModal, ImageBuildModal }
})
class Images extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: any

    // ─── Refs ───
    @Ref readonly pullModalRef!: InstanceType<typeof ImagePullModal>
    @Ref readonly tagModalRef!: InstanceType<typeof ImageTagModal>
    @Ref readonly buildModalRef!: InstanceType<typeof ImageBuildModal>

    // ─── 数据属性 ───
    images: any[] = []
    loading = false
    showAllImages = false
    formatFileSize = formatFileSize
    formatTime = formatTime

    // ─── 方法 ───
    async loadImages() {
        this.loading = true
        try {
            const res = await api.listImages(this.showAllImages)
            this.images = res.payload || []
        } catch (e) {
            this.actions.showNotification('error', '加载镜像列表失败')
        }
        this.loading = false
    }

    handleImageAction(image: any, action: string) {
        this.actions.showConfirm({
            title: '删除镜像',
            message: `确定要删除镜像 <strong class="text-slate-900">${image.repoTags[0] || image.id}</strong> 吗？`,
            icon: 'fa-trash',
            iconColor: 'red',
            confirmText: '确认删除',
            danger: true,
            onConfirm: async () => {
                await api.imageAction(image.id, action)
                this.actions.showNotification('success', '镜像删除成功')
                this.loadImages()
            }
        })
    }

    // ─── 生命周期 ───
    mounted() {
        this.loadImages()
    }
}

export default toNative(Images)
</script>

<template>
  <div>
    <div class="card mb-4">
      <!-- Toolbar Bar -->
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-6 py-3">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-lg bg-blue-500 flex items-center justify-center">
              <i class="fas fa-compact-disc text-white"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">镜像管理</h1>
              <p class="text-xs text-slate-500">管理 Docker 镜像</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <div class="flex items-center gap-1 bg-slate-100 p-1 rounded-lg">
              <button 
                @click="showAllImages = false; loadImages()" 
                :class="[
                  'px-3 py-1 text-xs font-medium rounded-md transition-all duration-200',
                  !showAllImages 
                    ? 'bg-white text-blue-600 shadow-sm' 
                    : 'text-slate-500 hover:text-slate-700'
                ]"
              >
                <i class="fas fa-cube mr-1"></i><span class="hidden sm:inline">顶层</span>
              </button>
              <button 
                @click="showAllImages = true; loadImages()" 
                :class="[
                  'px-3 py-1 text-xs font-medium rounded-md transition-all duration-200',
                  showAllImages 
                    ? 'bg-white text-blue-600 shadow-sm' 
                    : 'text-slate-500 hover:text-slate-700'
                ]"
              >
                <i class="fas fa-layer-group mr-1"></i><span class="hidden sm:inline">全部</span>
              </button>
            </div>
            <button @click="loadImages()" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-rotate"></i>刷新
            </button>
            <button @click="buildModalRef?.show()" class="px-3 py-1.5 rounded-lg bg-blue-500 hover:bg-blue-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-hammer"></i>构建
            </button>
            <button @click="pullModalRef?.show()" class="px-3 py-1.5 rounded-lg bg-blue-500 hover:bg-blue-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-download"></i>拉取
            </button>
          </div>
        </div>
      </div>

    <!-- Loading -->
    <div v-if="loading" class="flex flex-col items-center justify-center py-20">
      <div class="w-12 h-12 spinner mb-3"></div>
      <p class="text-slate-500">加载中...</p>
    </div>

    <!-- Image List -->
    <div v-else-if="images.length > 0" class="space-y-3">
      <!-- 桌面端表格视图 -->
      <div class="hidden md:block overflow-x-auto">
        <table class="w-full border-collapse">
          <thead>
            <tr class="bg-slate-50 border-b border-slate-200">
              <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">镜像</th>
              <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">ID</th>
              <th class="w-32 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">大小</th>
              <th class="w-36 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">创建时间</th>
              <th class="w-40 px-4 py-3 text-right text-xs font-semibold text-slate-600 uppercase tracking-wider">操作</th>
            </tr>
          </thead>
          <tbody class="bg-white divide-y divide-slate-100">
            <tr v-for="img in images" :key="img.id" class="hover:bg-slate-50 transition-colors">
              <td class="px-4 py-3">
                <div class="flex flex-col gap-0.5">
                  <div class="flex items-center gap-2">
                    <div class="w-8 h-8 rounded-lg bg-blue-400 flex items-center justify-center flex-shrink-0">
                      <i class="fas fa-compact-disc text-white text-sm"></i>
                    </div>
                    <span class="font-medium text-slate-800">{{ img.repoTags[0] || '<none>' }}</span>
                  </div>
                  <div v-if="img.repoTags.length > 1" class="ml-10 flex flex-wrap gap-1">
                    <span v-for="(tag, idx) in img.repoTags.slice(1)" :key="idx" class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-blue-50 text-blue-600">
                      {{ tag }}
                    </span>
                  </div>
                </div>
              </td>
              <td class="px-4 py-3"><code class="text-xs text-slate-500 font-mono">{{ img.shortId }}</code></td>
              <td class="px-4 py-3 text-sm text-slate-600">{{ formatFileSize(img.size) }}</td>
              <td class="px-4 py-3 whitespace-nowrap text-sm text-slate-600">{{ formatTime(new Date(img.created * 1000).toISOString()) }}</td>
              <td class="px-4 py-3">
                <div class="flex justify-end items-center gap-0.5">
                  <button @click="$router.push('/docker/image/' + img.id)" class="btn-icon text-slate-600 hover:bg-slate-100" title="查看详情">
                    <i class="fas fa-circle-info text-xs"></i>
                  </button>
                  <button @click="tagModalRef?.show(img)" class="btn-icon text-blue-600 hover:bg-blue-50" title="打标签">
                    <i class="fas fa-tag text-xs"></i>
                  </button>
                  <button @click="handleImageAction(img, 'remove')" class="btn-icon text-red-600 hover:bg-red-50" title="删除">
                    <i class="fas fa-trash text-xs"></i>
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 移动端卡片视图 -->
      <div class="md:hidden space-y-3">
        <div 
          v-for="img in images" 
          :key="img.id"
          class="rounded-xl border border-slate-200 bg-white p-4 transition-all hover:shadow-sm"
        >
          <!-- 顶部：镜像信息和图标 -->
          <div class="flex items-center justify-between mb-3">
            <div class="flex items-center gap-3">
              <div class="w-10 h-10 rounded-lg bg-blue-400 flex items-center justify-center flex-shrink-0">
                <i class="fas fa-compact-disc text-white text-base"></i>
              </div>
              <div class="min-w-0">
                <div class="flex items-center gap-2">
                  <span class="font-medium text-slate-800 text-sm">{{ img.repoTags[0] || '<none>' }}</span>
                </div>
                <div class="flex items-center gap-3 mt-1">
                  <span class="text-xs text-slate-500">{{ formatFileSize(img.size) }}</span>
                  <span class="text-xs text-slate-500">{{ formatTime(new Date(img.created * 1000).toISOString()) }}</span>
                </div>
              </div>
            </div>
          </div>
          
          <!-- 中间：镜像ID和标签 -->
          <div class="mb-3">
            <p class="text-xs text-slate-500 mb-1">镜像ID</p>
            <code class="text-xs text-slate-500 font-mono break-all">{{ img.shortId }}</code>
          </div>
          
          <!-- 其他标签 -->
          <div v-if="img.repoTags.length > 1" class="mb-3">
            <p class="text-xs text-slate-500 mb-1">其他标签</p>
            <div class="flex flex-wrap gap-1">
              <span v-for="(tag, idx) in img.repoTags.slice(1)" :key="idx" class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-blue-50 text-blue-600">
                {{ tag }}
              </span>
            </div>
          </div>
          
          <!-- 底部：操作按钮 -->
          <div class="flex flex-wrap gap-1 pt-2 border-t border-slate-100">
            <button @click="$router.push('/docker/image/' + img.id)" class="btn-icon text-slate-600 hover:bg-slate-50" title="查看详情">
              <i class="fas fa-circle-info text-xs"></i>
              <span class="text-xs ml-1 hidden xs:inline">详情</span>
            </button>
            <button @click="tagModalRef?.show(img)" class="btn-icon text-blue-600 hover:bg-blue-50" title="打标签">
              <i class="fas fa-tag text-xs"></i>
              <span class="text-xs ml-1 hidden xs:inline">标签</span>
            </button>
            <button @click="handleImageAction(img, 'remove')" class="btn-icon text-red-600 hover:bg-red-50" title="删除">
              <i class="fas fa-trash text-xs"></i>
              <span class="text-xs ml-1 hidden xs:inline">删除</span>
            </button>
          </div>
        </div>
      </div>
    </div>

    <div v-else class="flex flex-col items-center justify-center py-20">
      <div class="w-20 h-20 rounded-full bg-slate-100 flex items-center justify-center mb-4">
        <i class="fas fa-compact-disc text-4xl text-slate-300"></i>
      </div>
      <p class="text-slate-600 font-medium mb-1">暂无镜像</p>
      <p class="text-sm text-slate-400">点击「拉取镜像」从 Registry 获取镜像</p>
    </div>
    </div>

    <ImagePullModal ref="pullModalRef" @success="loadImages" />
    <ImageTagModal ref="tagModalRef" @success="loadImages" />
    <ImageBuildModal ref="buildModalRef" @success="loadImages" />
  </div>
</template>
