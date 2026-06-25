<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { ApisixSSL } from '@/service/types'

import PageSearch from '@/component/page-search.vue'

import SSLEditModal from './widget/ssl-edit-modal.vue'

@Component({
    components: { PageSearch, SSLEditModal }
})
class SSLs extends Vue {
    portal = usePortal()

    @Ref readonly editModalRef!: InstanceType<typeof SSLEditModal>

    ssls: ApisixSSL[] = []
    loading = false
    searchText = ''

    get filteredSSLs() {
        const keyword = this.searchText.trim().toLowerCase()
        if (!keyword) return this.ssls
        return this.ssls.filter((ssl: ApisixSSL) =>
            (ssl.id || '').toLowerCase().includes(keyword) ||
            (ssl.snis || []).some((sni: string) => sni.toLowerCase().includes(keyword))
        )
    }

    sortSSLs(data: ApisixSSL[]) {
        data.sort((a: ApisixSSL, b: ApisixSSL) => this.getPrimarySNI(a).localeCompare(this.getPrimarySNI(b)))
        return data
    }

    async loadSSLs() {
        this.loading = true
        try {
            this.ssls = this.sortSSLs((await api.apisixSSLList()).payload || [])
        } catch {
            this.portal.showNotification('error', '加载证书列表失败')
        } finally {
            this.loading = false
        }
    }

    openCreateModal() {
        this.editModalRef?.show()
    }

    openEditModal(ssl: ApisixSSL | null) {
        this.editModalRef?.show(ssl)
    }

    getPrimarySNI(ssl: ApisixSSL) {
        return ssl.snis?.[0] || ssl.id || '-'
    }

    getSNISummary(ssl: ApisixSSL) {
        const snis = ssl.snis || []
        if (snis.length === 0) return '-'
        if (snis.length === 1) return snis[0]
        return `${snis[0]} 等 ${snis.length} 个域名`
    }

    getStatusClass(ssl: ApisixSSL) {
        return (ssl.status ?? 1) === 0 ? 'text-slate-500' : 'text-emerald-600 font-medium'
    }

    getStatusText(ssl: ApisixSSL) {
        return (ssl.status ?? 1) === 0 ? '禁用' : '启用'
    }

    formatTs(ts?: number) {
        if (!ts) return '-'
        return new Date(ts * 1000).toLocaleString()
    }

    deleteSSL(ssl: ApisixSSL) {
        const id = ssl.id
        if (!id) return
        this.portal.showConfirm({
            title: '删除证书',
            message: `确定要删除证书 <strong class="text-slate-900">${this.getPrimarySNI(ssl)}</strong> 吗？正在被 SNI 使用时可能影响 HTTPS 访问。`,
            icon: 'fa-trash',
            iconColor: 'red',
            confirmText: '确认删除',
            danger: true,
            onConfirm: async () => {
                await api.apisixSSLDelete(id)
                this.portal.showNotification('success', '删除成功')
                this.loadSSLs()
            }
        })
    }

    mounted() {
        this.loadSSLs()
    }
}

export default toNative(SSLs)
</script>

<template>
  <div class="page">
    <div class="page-toolbar">
      <div class="hidden md:flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="page-icon bg-cyan-500">
            <i class="fas fa-certificate text-white"></i>
          </div>
          <div>
            <h1 class="text-lg font-semibold text-slate-800 truncate">SSL 证书</h1>
            <p class="text-xs text-slate-500">管理 APISIX 的 SSL 证书绑定与 SNI 配置</p>
          </div>
        </div>
        <div class="flex items-center gap-2 flex-shrink-0">
          <PageSearch v-model="searchText" search-key="apisix-ssls" placeholder="搜索证书、SNI 或 ID..." focus-color="cyan" type-to-search />
          <button class="btn btn-secondary" @click="loadSSLs()">
            <i class="fas fa-rotate"></i>刷新
          </button>
          <button v-if="portal.hasPerm('POST /api/apisix/ssl')" class="btn btn-cyan" @click="openCreateModal()">
            <i class="fas fa-plus"></i>新建证书
          </button>
        </div>
      </div>

      <div class="flex md:hidden items-center justify-between">
        <div class="flex items-center gap-3 min-w-0 flex-1">
          <div class="page-icon bg-cyan-500">
            <i class="fas fa-certificate text-white"></i>
          </div>
          <div class="min-w-0">
            <h1 class="text-lg font-semibold text-slate-800 truncate">SSL 证书</h1>
            <p class="text-xs text-slate-500 truncate">管理证书与 SNI 绑定</p>
          </div>
        </div>
        <div class="flex items-center gap-1 flex-shrink-0">
          <button class="btn btn-secondary w-9 h-9 !px-0" title="刷新" @click="loadSSLs()">
            <i class="fas fa-rotate text-sm"></i>
          </button>
          <button v-if="portal.hasPerm('POST /api/apisix/ssl')" class="btn btn-cyan w-9 h-9 !px-0" title="新建证书" @click="openCreateModal()">
            <i class="fas fa-plus text-sm"></i>
          </button>
        </div>
      </div>
    </div>

    <div class="mobile-search">
      <PageSearch v-model="searchText" search-key="apisix-ssls" placeholder="搜索证书或 SNI..." width-class="w-full" focus-color="cyan" />
    </div>

    <div v-if="loading" class="card-body">
      <div class="empty-state">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>
    </div>

    <div v-else-if="filteredSSLs.length === 0" class="card-body">
      <div class="empty-state">
        <div class="empty-state-icon">
          <i class="fas fa-certificate text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">{{ ssls.length === 0 ? '暂无证书' : '未找到匹配证书' }}</p>
        <p class="text-sm text-slate-400">{{ ssls.length === 0 ? '点击「新建证书」开始创建' : '尝试更换关键词或清空搜索条件' }}</p>
      </div>
    </div>

    <template v-else>
      <div class="card-table hidden md:block">
        <table class="w-full border-collapse">
          <thead>
            <tr class="bg-slate-50 border-b border-slate-200">
              <th class="th">SNI</th>
              <th class="th">状态</th>
              <th class="th">更新时间</th>
              <th class="w-32 th-right">操作</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <tr v-for="ssl in filteredSSLs" :key="ssl.id" class="hover:bg-slate-50 transition-colors">
              <td class="px-4 py-3 max-w-[280px]">
                <div class="flex items-center gap-2 min-w-0">
                  <div class="row-icon bg-cyan-400">
                    <i class="fas fa-certificate text-white text-sm"></i>
                  </div>
                  <div class="min-w-0">
                    <span class="font-medium text-slate-800 truncate block">{{ getPrimarySNI(ssl) }}</span>
                    <span class="text-mono-muted">{{ ssl.id }}</span>
                  </div>
                </div>
              </td>
              <td class="px-4 py-3">
                <span :class="getStatusClass(ssl)" class="text-sm">{{ getStatusText(ssl) }}</span>
              </td>
              <td class="px-4 py-3 whitespace-nowrap text-sm text-slate-600">{{ formatTs(ssl.update_time || ssl.create_time) }}</td>
              <td class="px-4 py-3">
                <div class="flex justify-end items-center gap-1">
                  <button v-if="portal.hasPerm('PUT /api/apisix/ssl/:id')" class="btn-icon btn-icon-cyan" title="编辑" @click="openEditModal(ssl)">
                    <i class="fas fa-pen text-xs"></i>
                  </button>
                  <button v-if="portal.hasPerm('DELETE /api/apisix/ssl/:id')" class="btn-icon btn-icon-red" title="删除" @click="deleteSSL(ssl)">
                    <i class="fas fa-trash text-xs"></i>
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="card-body md:hidden space-y-3">
        <div v-for="ssl in filteredSSLs" :key="ssl.id" class="card-interactive">
          <div class="flex items-center justify-between gap-3 mb-3">
            <div class="card-info-row !mb-0">
              <div class="list-icon bg-cyan-400">
                <i class="fas fa-certificate text-white text-base"></i>
              </div>
              <div class="min-w-0">
                <span class="font-medium text-slate-800 text-sm truncate block">{{ getPrimarySNI(ssl) }}</span>
                <span class="text-mono-muted">{{ ssl.id }}</span>
              </div>
            </div>
            <span :class="getStatusClass(ssl)" class="text-xs flex-shrink-0">{{ getStatusText(ssl) }}</span>
          </div>

          <div class="card-prop-row">
            <span class="text-xs text-slate-400 flex-shrink-0">SNI</span>
            <span class="text-xs text-slate-500 break-all">{{ getSNISummary(ssl) }}</span>
          </div>
          <div class="card-prop-row">
            <span class="text-xs text-slate-400 flex-shrink-0">更新</span>
            <span class="text-xs text-slate-500">{{ formatTs(ssl.update_time || ssl.create_time) }}</span>
          </div>

          <div class="card-actions">
            <button v-if="portal.hasPerm('PUT /api/apisix/ssl/:id')" class="btn-icon btn-icon-cyan" title="编辑" @click="openEditModal(ssl)">
              <i class="fas fa-pen text-xs"></i><span class="text-xs ml-1">编辑</span>
            </button>
            <button v-if="portal.hasPerm('DELETE /api/apisix/ssl/:id')" class="btn-icon btn-icon-red" title="删除" @click="deleteSSL(ssl)">
              <i class="fas fa-trash text-xs"></i><span class="text-xs ml-1">删除</span>
            </button>
          </div>
        </div>
      </div>
    </template>
  </div>

  <SSLEditModal ref="editModalRef" @success="loadSSLs" />
</template>
