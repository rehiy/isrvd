<script lang="ts">
import { Component, Inject, Ref, Vue, toNative } from 'vue-facing-decorator'

import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

import api from '@/service/api'
import type { ApisixSSL } from '@/service/types'

import SSLEditModal from './widget/ssl-edit-modal.vue'

@Component({
    components: { SSLEditModal }
})
class SSLs extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    @Ref readonly editModalRef!: InstanceType<typeof SSLEditModal>

    ssls: ApisixSSL[] = []
    loading = false
    searchText = ''

    get filteredSSLs() {
        if (!this.searchText) return this.ssls
        const s = this.searchText.toLowerCase()
        return this.ssls.filter((ssl: ApisixSSL) =>
            (ssl.id || '').toLowerCase().includes(s) ||
            (ssl.snis || []).some((sni: string) => sni.toLowerCase().includes(s))
        )
    }

    sortSSLs(data: ApisixSSL[]) {
        data.sort((a: ApisixSSL, b: ApisixSSL) => this.getPrimarySNI(a).localeCompare(this.getPrimarySNI(b)))
        return data
    }

    async loadSSLs() {
        this.loading = true
        try {
            this.ssls = this.sortSSLs((await api.apisixListSSLs()).payload || [])
        } catch {
            this.actions.showNotification('error', '加载证书列表失败')
        }
        this.loading = false
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

    getStatusClass(status?: number) {
        return status === 0 ? 'bg-slate-100 text-slate-600' : 'bg-emerald-50 text-emerald-700'
    }

    getStatusText(status?: number) {
        return status === 0 ? '禁用' : '启用'
    }

    formatTs(ts?: number) {
        if (!ts) return '-'
        return new Date(ts * 1000).toLocaleString()
    }

    deleteSSL(ssl: ApisixSSL) {
        const id = ssl.id
        if (!id) return
        this.actions.showConfirm({
            title: '删除证书',
            message: `确定要删除证书 <strong class="text-slate-900">${this.getPrimarySNI(ssl)}</strong> 吗？正在被 SNI 使用时可能影响 HTTPS 访问。`,
            icon: 'fa-trash',
            iconColor: 'red',
            confirmText: '确认删除',
            danger: true,
            onConfirm: async () => {
                await api.apisixDeleteSSL(id)
                this.actions.showNotification('success', '删除成功')
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
  <div>
    <div class="card mb-4">
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-4 md:px-6 py-3">
        <div class="hidden md:flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-lg bg-cyan-500 flex items-center justify-center">
              <i class="fas fa-certificate text-white"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">证书管理</h1>
              <p class="text-xs text-slate-500">管理 APISIX SSL 证书与 SNI 域名</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <div class="relative">
              <input v-model="searchText" type="text" placeholder="搜索证书、SNI 或 ID..." class="pl-8 pr-3 py-1.5 text-xs border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:border-transparent w-56" />
              <i class="fas fa-search absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-400 text-xs"></i>
            </div>
            <button @click="loadSSLs()" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-rotate"></i>刷新
            </button>
            <button v-if="actions.hasPerm('apisix', true)" @click="openCreateModal()" class="px-3 py-1.5 rounded-lg bg-cyan-500 hover:bg-cyan-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-plus"></i>创建
            </button>
          </div>
        </div>

        <div class="flex md:hidden items-center justify-between">
          <div class="flex items-center gap-3 min-w-0 flex-1">
            <div class="w-9 h-9 rounded-lg bg-cyan-500 flex items-center justify-center flex-shrink-0">
              <i class="fas fa-certificate text-white"></i>
            </div>
            <div class="min-w-0">
              <h1 class="text-lg font-semibold text-slate-800 truncate">证书管理</h1>
              <p class="text-xs text-slate-500 truncate">管理 SSL 证书与 SNI</p>
            </div>
          </div>
          <div class="flex items-center gap-1 flex-shrink-0">
            <button @click="loadSSLs()" class="w-9 h-9 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 flex items-center justify-center text-slate-600 transition-colors" title="刷新">
              <i class="fas fa-rotate text-sm"></i>
            </button>
            <button v-if="actions.hasPerm('apisix', true)" @click="openCreateModal()" class="w-9 h-9 rounded-lg bg-cyan-500 hover:bg-cyan-600 flex items-center justify-center text-white transition-colors" title="创建">
              <i class="fas fa-plus text-sm"></i>
            </button>
          </div>
        </div>
      </div>

      <div class="md:hidden px-4 py-2 border-b border-slate-100">
        <div class="relative">
          <input v-model="searchText" type="text" placeholder="搜索证书或 SNI..." class="w-full pl-8 pr-3 py-1.5 text-xs border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:border-transparent" />
          <i class="fas fa-search absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-400 text-xs"></i>
        </div>
      </div>

      <div v-if="loading" class="flex flex-col items-center justify-center py-20">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>

      <div v-else-if="filteredSSLs.length === 0" class="flex flex-col items-center justify-center py-20">
        <div class="w-20 h-20 rounded-full bg-slate-100 flex items-center justify-center mb-4">
          <i class="fas fa-certificate text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">暂无证书</p>
        <p class="text-sm text-slate-400">点击「创建」添加 SSL 证书</p>
      </div>

      <div v-else class="space-y-3">
        <div class="hidden md:block overflow-x-auto">
          <table class="w-full border-collapse">
            <thead>
              <tr class="bg-slate-50 border-b border-slate-200">
                <th class="text-left px-4 py-3 text-xs font-semibold text-slate-600 uppercase tracking-wider">SNI</th>
                <th class="text-left px-4 py-3 text-xs font-semibold text-slate-600 uppercase tracking-wider">状态</th>
                <th class="text-left px-4 py-3 text-xs font-semibold text-slate-600 uppercase tracking-wider">更新时间</th>
                <th class="w-32 px-4 py-3 text-right text-xs font-semibold text-slate-600 uppercase tracking-wider">操作</th>
              </tr>
            </thead>
            <tbody class="bg-white divide-y divide-slate-100">
              <tr v-for="ssl in filteredSSLs" :key="ssl.id" class="hover:bg-slate-50 transition-colors">
                <td class="px-4 py-3 max-w-[280px]">
                  <div class="flex items-center gap-2 min-w-0">
                    <div class="w-8 h-8 rounded-lg bg-cyan-400 flex items-center justify-center flex-shrink-0">
                      <i class="fas fa-certificate text-white text-sm"></i>
                    </div>
                    <div class="min-w-0">
                      <span class="font-medium text-slate-800 truncate block">{{ getPrimarySNI(ssl) }}</span>
                      <span class="text-xs text-slate-400 truncate block mt-0.5 font-mono">{{ ssl.id }}</span>
                    </div>
                  </div>
                </td>
                <td class="px-4 py-3">
                  <span :class="['text-xs px-2 py-1 rounded-full', getStatusClass(ssl.status)]">{{ getStatusText(ssl.status) }}</span>
                </td>
                <td class="px-4 py-3 whitespace-nowrap text-sm text-slate-600">{{ formatTs(ssl.update_time || ssl.create_time) }}</td>
                <td class="px-4 py-3">
                  <div class="flex justify-end items-center gap-0.5">
                    <button v-if="actions.hasPerm('apisix', true)" @click="openEditModal(ssl)" class="btn-icon text-blue-600 hover:bg-blue-50" title="编辑">
                      <i class="fas fa-pen text-xs"></i>
                    </button>
                    <button v-if="actions.hasPerm('apisix', true)" @click="deleteSSL(ssl)" class="btn-icon text-red-600 hover:bg-red-50" title="删除">
                      <i class="fas fa-trash text-xs"></i>
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div class="md:hidden space-y-3 p-4">
          <div
            v-for="ssl in filteredSSLs"
            :key="ssl.id"
            class="rounded-xl border border-slate-200 bg-white p-4 transition-all hover:shadow-sm"
          >
            <div class="flex items-center justify-between mb-3">
              <div class="flex items-center gap-3 min-w-0 flex-1">
                <div class="w-10 h-10 rounded-lg bg-cyan-400 flex items-center justify-center flex-shrink-0">
                  <i class="fas fa-certificate text-white text-base"></i>
                </div>
                <div class="min-w-0">
                  <div class="font-medium text-sm text-slate-800 truncate">{{ getPrimarySNI(ssl) }}</div>
                  <div class="text-xs text-slate-400 mt-0.5 truncate font-mono">{{ ssl.id }}</div>
                </div>
              </div>
              <span :class="['text-xs px-2 py-1 rounded-full flex-shrink-0', getStatusClass(ssl.status)]">{{ getStatusText(ssl.status) }}</span>
            </div>

            <div class="flex items-center gap-2 mb-2">
              <span class="text-xs text-slate-400 flex-shrink-0">SNI</span>
              <span class="text-xs text-slate-600 break-all">{{ getSNISummary(ssl) }}</span>
            </div>
            <div class="flex items-center gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0">更新</span>
              <span class="text-xs text-slate-600">{{ formatTs(ssl.update_time || ssl.create_time) }}</span>
            </div>

            <div class="flex flex-wrap gap-1 pt-2 border-t border-slate-100">
              <button v-if="actions.hasPerm('apisix', true)" @click="openEditModal(ssl)" class="btn-icon text-blue-600 hover:bg-blue-50" title="编辑">
                <i class="fas fa-pen text-xs"></i><span class="text-xs ml-1">编辑</span>
              </button>
              <button v-if="actions.hasPerm('apisix', true)" @click="deleteSSL(ssl)" class="btn-icon text-red-600 hover:bg-red-50" title="删除">
                <i class="fas fa-trash text-xs"></i><span class="text-xs ml-1">删除</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <SSLEditModal ref="editModalRef" @success="loadSSLs" />
  </div>
</template>
