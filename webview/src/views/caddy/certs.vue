<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { CaddyCert } from '@/service/types'

import PageSearch from '@/component/page-search.vue'

import { usePortal } from '@/stores'

import CertEditModal from './widget/cert-edit-modal.vue'

@Component({
    components: { PageSearch, CertEditModal }
})
class CaddyCerts extends Vue {
    portal = usePortal()

    @Ref readonly editModalRef!: InstanceType<typeof CertEditModal>

    certs: CaddyCert[] = []
    loading = false
    searchText = ''

    get filteredCerts() {
        const keyword = this.searchText.trim().toLowerCase()
        if (!keyword) return this.certs
        return this.certs.filter((c: CaddyCert) =>
            (c.subject || '').toLowerCase().includes(keyword) ||
            (c.certificate || '').toLowerCase().includes(keyword) ||
            (c.tags || []).some((t: string) => t.toLowerCase().includes(keyword))
        )
    }

    async loadCerts() {
        this.loading = true
        try {
            this.certs = (await api.caddyCertList()).payload || []
        } catch {
            this.portal.showNotification('error', '加载证书列表失败')
        } finally {
            this.loading = false
        }
    }

    openCreateModal() {
        this.editModalRef?.show(null)
    }

    openEditModal(cert: CaddyCert) {
        this.editModalRef?.show(cert)
    }

    sourceLabel(source: string) {
        const map: Record<string, string> = { file: '磁盘文件', pem: '内联 PEM', automate: '自动签发' }
        return map[source] || source
    }

    sourceTagClass(source: string) {
        if (source === 'file') return 'bg-indigo-50 text-indigo-700'
        if (source === 'pem') return 'bg-emerald-50 text-emerald-700'
        if (source === 'automate') return 'bg-amber-50 text-amber-700'
        return 'bg-slate-100 text-slate-500'
    }

    certSummary(cert: CaddyCert) {
        if (cert.source === 'automate') return cert.subject || '-'
        if (cert.source === 'file') return cert.certificate || '-'
        // pem：显示前两行（通常是 BEGIN CERTIFICATE）
        const pem = cert.certificate || ''
        return pem.split('\n').slice(0, 1).join('').trim() || '(空)'
    }

    deleteCert(cert: CaddyCert) {
        if (!cert.key) return
        this.portal.showConfirm({
            title: '删除证书',
            message: `确定要删除证书 <strong class="text-slate-900">${this.certSummary(cert)}</strong> 吗？此操作不可恢复。`,
            icon: 'fa-trash',
            iconColor: 'red',
            confirmText: '确认删除',
            danger: true,
            onConfirm: async () => {
                try {
                    await api.caddyCertDelete(cert.key as string)
                    this.portal.showNotification('success', '删除成功')
                    this.loadCerts()
                } catch (e: unknown) {
                    this.portal.showNotification('error', (e instanceof Error ? e.message : '') || '删除失败')
                }
            }
        })
    }

    mounted() {
        this.loadCerts()
    }
}

export default toNative(CaddyCerts)
</script>

<template>
  <div>
    <div class="card mb-4">
      <div class="card-toolbar">
        <!-- 桌面端 -->
        <div class="hidden md:flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="page-icon bg-cyan-500"><i class="fas fa-certificate text-white"></i></div>
            <div class="min-w-0"><h1 class="text-lg font-semibold text-slate-800 truncate">TLS 证书</h1><p class="text-xs text-slate-500 truncate">管理 Caddy 证书来源：文件、PEM 或自动签发</p></div>
          </div>
          <div class="flex items-center gap-2 flex-shrink-0">
            <PageSearch v-model="searchText" search-key="caddy-certs" placeholder="搜索主机、路径、标签..." width-class="w-64" focus-color="cyan" type-to-search />
            <button class="btn btn-secondary" @click="loadCerts()"><i class="fas fa-rotate"></i>刷新</button>
            <button v-if="portal.hasPerm('POST /api/caddy/cert')" class="btn btn-cyan" @click="openCreateModal()"><i class="fas fa-plus"></i>新建证书</button>
          </div>
        </div>
        <!-- 移动端 -->
        <div class="flex md:hidden items-center justify-between">
          <div class="flex items-center gap-3 min-w-0 flex-1">
            <div class="page-icon bg-cyan-500"><i class="fas fa-certificate text-white"></i></div>
            <div class="min-w-0">
              <h1 class="text-lg font-semibold text-slate-800 truncate">TLS 证书</h1>
              <p class="text-xs text-slate-500 truncate">管理证书来源</p>
            </div>
          </div>
          <div class="flex items-center gap-1 flex-shrink-0">
            <button class="btn btn-secondary w-9 h-9 !px-0" title="刷新" @click="loadCerts()">
              <i class="fas fa-rotate text-sm"></i>
            </button>
            <button v-if="portal.hasPerm('POST /api/caddy/cert')" class="btn btn-cyan w-9 h-9 !px-0" title="新建证书" @click="openCreateModal()">
              <i class="fas fa-plus text-sm"></i>
            </button>
          </div>
        </div>
      </div>
      <!-- 移动端搜索栏 -->
      <div class="mobile-search">
        <PageSearch v-model="searchText" search-key="caddy-certs" placeholder="搜索主机、路径、标签..." width-class="w-full" focus-color="cyan" />
      </div>
      <div v-if="loading" class="loading-state"><div class="w-12 h-12 spinner mb-3"></div><p class="text-slate-500">加载中...</p></div>
      <div v-else-if="filteredCerts.length === 0" class="empty-state">
        <div class="empty-state-icon"><i class="fas fa-certificate text-4xl text-slate-300"></i></div>
        <p class="text-slate-600 font-medium mb-1">{{ certs.length === 0 ? '暂无证书' : '未找到匹配证书' }}</p>
        <p class="text-sm text-slate-400">{{ certs.length === 0 ? '点击「新建证书」开始创建' : '尝试更换关键词或清空搜索条件' }}</p>
      </div>
      <div v-else class="space-y-3">
        <!-- 桌面端表格 -->
        <div class="hidden md:block overflow-x-auto">
          <table class="w-full border-collapse">
            <thead>
              <tr class="bg-slate-50 border-b border-slate-200">
                <th class="th">来源</th>
                <th class="th">主体</th>
                <th class="th">标签</th>
                <th class="w-32 th-right">操作</th>
              </tr>
            </thead>
            <tbody class="bg-white divide-y divide-slate-100">
              <tr v-for="cert in filteredCerts" :key="cert.key" class="hover:bg-slate-50 transition-colors">
                <td class="px-4 py-3 max-w-[280px]">
                  <div class="flex items-center gap-2 min-w-0">
                    <div class="row-icon bg-cyan-400">
                      <i class="fas fa-certificate text-white text-sm"></i>
                    </div>
                    <div class="min-w-0">
                      <span class="font-medium text-slate-800 truncate block">{{ sourceLabel(cert.source) }}</span>
                      <span v-if="cert.key" class="text-xs text-slate-400 truncate block mt-0.5">{{ cert.key }}</span>
                    </div>
                  </div>
                </td>
                <td class="px-4 py-3"><code class="text-xs font-mono text-slate-700 break-all">{{ certSummary(cert) }}</code></td>
                <td class="px-4 py-3">
                  <span v-if="!cert.tags || cert.tags.length === 0" class="text-xs text-slate-400">-</span>
                  <span v-for="tag in cert.tags" v-else :key="tag" class="inline-block text-xs px-2 py-0.5 rounded bg-slate-100 text-slate-600 mr-1">{{ tag }}</span>
                </td>
                <td class="px-4 py-3">
                  <div class="flex justify-end items-center gap-1">
                    <button v-if="portal.hasPerm('PUT /api/caddy/cert/:key')" class="btn-icon btn-icon-blue" title="编辑" @click="openEditModal(cert)"><i class="fas fa-pen text-xs"></i></button>
                    <button v-if="portal.hasPerm('DELETE /api/caddy/cert/:key')" class="btn-icon btn-icon-red" title="删除" @click="deleteCert(cert)"><i class="fas fa-trash text-xs"></i></button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- 移动端卡片 -->
        <div class="md:hidden space-y-3 p-4">
          <div
            v-for="cert in filteredCerts"
            :key="cert.key"
            class="card-interactive"
          >
            <div class="card-info-row">
              <div class="list-icon bg-cyan-400">
                <i class="fas fa-certificate text-white text-base"></i>
              </div>
              <div class="min-w-0">
                <span class="font-medium text-slate-800 text-sm truncate block">{{ certSummary(cert) }}</span>
                <span class="text-xs text-slate-400 truncate block mt-0.5">{{ sourceLabel(cert.source) }}</span>
              </div>
            </div>

            <div v-if="cert.tags && cert.tags.length" class="flex items-start gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0 mt-0.5">标签</span>
              <span class="flex flex-wrap gap-1">
                <span v-for="tag in cert.tags" :key="tag" class="inline-block text-xs px-2 py-0.5 rounded bg-slate-100 text-slate-600">{{ tag }}</span>
              </span>
            </div>

            <div class="card-actions">
              <button v-if="portal.hasPerm('PUT /api/caddy/cert/:key')" class="btn-icon btn-icon-blue" title="编辑" @click="openEditModal(cert)">
                <i class="fas fa-pen text-xs"></i><span class="text-xs ml-1">编辑</span>
              </button>
              <button v-if="portal.hasPerm('DELETE /api/caddy/cert/:key')" class="btn-icon btn-icon-red" title="删除" @click="deleteCert(cert)">
                <i class="fas fa-trash text-xs"></i><span class="text-xs ml-1">删除</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <CertEditModal ref="editModalRef" @success="loadCerts" />
  </div>
</template>
