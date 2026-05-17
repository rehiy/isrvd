<script lang="ts">
import { json } from '@codemirror/lang-json'
import { jsonrepair } from 'jsonrepair'
import { Codemirror } from 'vue-codemirror'
import { Component, toNative, Vue } from 'vue-facing-decorator'

import api from '@/service/api'
import type { AuditLog } from '@/service/types'

import BaseModal from '@/component/modal.vue'
import PageSearch from '@/component/page-search.vue'

import { usePortal } from '@/stores'

@Component({
    components: { BaseModal, PageSearch, Codemirror }
})
class AuditLogs extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    logs: AuditLog[] = []
    loading = false
    selectedUsername = ''
    searchText = ''

    // ─── 详情 Modal ───
    detailOpen = false
    detailLog: AuditLog | null = null
    readonly jsonExtensions = [json()]

    // ─── 计算属性 ───
    get detailBody(): string {
        if (!this.detailLog?.body) return ''
        try {
            let parsed = JSON.parse(this.detailLog.body)
            parsed = this.unwrapJson(parsed)
            return JSON.stringify(parsed, null, 2)
        } catch {
            try {
                const repaired = jsonrepair(this.detailLog.body)
                let parsed = JSON.parse(repaired)
                parsed = this.unwrapJson(parsed)
                return JSON.stringify(parsed, null, 2)
            } catch {
                return this.detailLog.body
            }
        }
    }

    get filteredLogs() {
        let list = this.logs
        if (this.selectedUsername) {
            list = list.filter(log => log.username === this.selectedUsername)
        }
        const keyword = this.searchText.trim().toLowerCase()
        if (!keyword) return list
        return list.filter((log: AuditLog) =>
            (log.username || '').toLowerCase().includes(keyword) ||
            (log.method || '').toLowerCase().includes(keyword) ||
            (log.uri || '').toLowerCase().includes(keyword) ||
            (log.body || '').toLowerCase().includes(keyword) ||
            (log.ip || '').toLowerCase().includes(keyword) ||
            String(log.statusCode || '').includes(keyword)
        )
    }

    get uniqueUsernames() {
        return Array.from(new Set(this.logs.map(log => log.username))).sort()
    }

    // ─── 方法 ───
    showDetail(log: AuditLog) {
        this.detailLog = log
        this.detailOpen = true
    }

    async loadLogs() {
        this.loading = true
        try {
            const res = await api.systemAuditLogs()
            this.logs = res.payload || []
        } catch {
            this.portal.showNotification('error', '获取审计日志失败')
        } finally {
            this.loading = false
        }
    }

    methodClass(method: string): string {
        const map: Record<string, string> = {
            POST: 'bg-emerald-100 text-emerald-700',
            PUT: 'bg-blue-100 text-blue-700',
            PATCH: 'bg-blue-100 text-blue-700',
            DELETE: 'bg-red-100 text-red-700',
        }
        return map[method] || 'bg-slate-100 text-slate-700'
    }

    formatTimestamp(timestamp: string): string {
        return new Date(timestamp).toLocaleString('zh-CN')
    }

    formatDuration(duration: number): string {
        return duration < 1000 ? `${duration}ms` : `${(duration / 1000).toFixed(2)}s`
    }

    unwrapJson(parsed: unknown): unknown {
        if (typeof parsed === 'string') {
            try { return JSON.parse(parsed) } catch { }
        }
        return parsed
    }

    formatBody(body: string): string {
        try {
            let parsed = JSON.parse(body)
            parsed = this.unwrapJson(parsed)
            return JSON.stringify(parsed)
        } catch {
            return body
        }
    }

    // ─── 生命周期 ───
    mounted() {
        this.loadLogs()
    }
}

export default toNative(AuditLogs)
</script>

<template>
  <div>
    <div class="card mb-4">
      <div class="card-toolbar">
        <!-- 桌面端 -->
        <div class="hidden md:flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="page-icon bg-rose-500">
              <i class="fas fa-clipboard-list text-white"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">操作审计</h1>
              <p class="text-xs text-slate-500">查看和检索所有用户的操作记录</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <PageSearch v-model="searchText" search-key="system-audit-logs" placeholder="搜索用户、方法、URI、IP 或状态..." width-class="w-64" focus-color="rose" type-to-search />
            <select v-model="selectedUsername" class="select-sm min-w-[140px]">
              <option value="">所有用户</option>
              <option v-for="username in uniqueUsernames" :key="username" :value="username">{{ username }}</option>
            </select>
            <button class="btn btn-sm btn-secondary" @click="loadLogs()">
              <i class="fas fa-rotate"></i>刷新
            </button>
          </div>
        </div>
        <!-- 移动端 -->
        <div class="block md:hidden">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-3 min-w-0 flex-1">
              <div class="page-icon bg-rose-500">
                <i class="fas fa-clipboard-list text-white"></i>
              </div>
              <div class="min-w-0">
                <h1 class="text-lg font-semibold text-slate-800 truncate">操作审计</h1>
                <p class="text-xs text-slate-500 truncate">查看用户操作记录</p>
              </div>
            </div>
            <div class="flex items-center gap-1.5 flex-shrink-0">
              <select v-model="selectedUsername" class="w-28 select-sm">
                <option value="">所有用户</option>
                <option v-for="username in uniqueUsernames" :key="username" :value="username">{{ username }}</option>
              </select>
              <button class="btn btn-sm btn-secondary w-9 h-9 !px-0" title="刷新" @click="loadLogs()">
                <i class="fas fa-rotate text-sm"></i>
              </button>
            </div>
          </div>
        </div>
      </div>
      <div class="mobile-search">
        <PageSearch v-model="searchText" search-key="system-audit-logs" placeholder="搜索用户、方法、URI、IP 或状态..." width-class="w-full" focus-color="rose" />

        <!-- Loading -->
        <div v-if="loading" class="loading-state">
          <div class="w-12 h-12 spinner mb-3"></div>
          <p class="text-slate-500">加载中...</p>
        </div>

        <!-- 空状态 -->
        <div v-else-if="filteredLogs.length === 0" class="empty-state">
          <div class="empty-state-icon">
            <i class="fas fa-clipboard-list text-4xl text-slate-300"></i>
          </div>
          <p class="text-slate-600 font-medium mb-1">{{ logs.length === 0 ? '暂无审计日志' : '未找到匹配日志' }}</p>
          <p class="text-sm text-slate-400">{{ logs.length === 0 ? '用户操作记录将在此展示' : '尝试更换关键词或清空搜索条件' }}</p>
        </div>

        <!-- 日志列表 -->
        <div v-else>
          <!-- 桌面表格 -->
          <div class="hidden md:block overflow-x-auto">
            <table class="w-full border-collapse">
              <thead>
                <tr class="bg-slate-50 border-b border-slate-200">
                  <th class="th">用户</th>
                  <th class="w-20 th">方法</th>
                  <th class="th">URI</th>
                  <th class="th">Body</th>
                  <th class="w-24 th">状态</th>
                  <th class="w-20 th">耗时</th>
                  <th class="w-36 th">时间</th>
                </tr>
              </thead>
              <tbody class="bg-white divide-y divide-slate-100">
                <tr v-for="(log, idx) in filteredLogs" :key="idx" class="hover:bg-slate-50 transition-colors">
                  <!-- 用户 -->
                  <td class="px-4 py-3 max-w-[280px]">
                    <div class="flex items-center gap-2 min-w-0">
                      <div class="row-icon bg-rose-400">
                        <i class="fas fa-user text-white text-sm"></i>
                      </div>
                      <div class="min-w-0">
                        <span class="font-medium text-slate-800 truncate block">{{ log.username }}</span>
                        <span class="text-xs text-slate-400 font-mono truncate block mt-0.5">{{ log.ip }}</span>
                      </div>
                    </div>
                  </td>
                  <!-- 方法 -->
                  <td class="px-4 py-3">
                    <span
                      class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium font-mono"
                      :class="methodClass(log.method)"
                    >{{ log.method }}</span>
                  </td>
                  <!-- URI -->
                  <td class="px-4 py-3 max-w-[240px]">
                    <code class="text-xs text-slate-700 font-mono truncate block">{{ log.uri }}</code>
                  </td>
                  <!-- Body -->
                  <td class="px-4 py-3 max-w-[200px]">
                    <button v-if="log.body" class="text-left w-full group" @click="showDetail(log)">
                      <code class="text-xs text-slate-600 font-mono truncate block group-hover:text-primary-600 transition-colors">{{ formatBody(log.body) }}</code>
                    </button>
                    <span v-else class="text-xs text-slate-300">-</span>
                  </td>
                  <!-- 状态 -->
                  <td class="px-4 py-3">
                    <span v-if="log.success" class="inline-flex items-center gap-1 text-xs font-medium text-emerald-600">
                      <i class="fas fa-circle-check"></i>{{ log.statusCode }}
                    </span>
                    <span v-else class="inline-flex items-center gap-1 text-xs font-medium text-red-600">
                      <i class="fas fa-circle-xmark"></i>{{ log.statusCode }}
                    </span>
                  </td>
                  <!-- 耗时 -->
                  <td class="px-4 py-3 text-xs text-slate-500">{{ formatDuration(log.duration) }}</td>
                  <!-- 时间 -->
                  <td class="px-4 py-3 text-xs text-slate-600 whitespace-nowrap">{{ formatTimestamp(log.timestamp) }}</td>
                </tr>
              </tbody>
            </table>
          </div>

          <!-- 移动卡片列表 -->
          <div class="md:hidden space-y-3 p-4">
            <div
              v-for="(log, idx) in filteredLogs" :key="idx"
              class="card-interactive"
            >
              <!-- 顶部：用户 + 时间 -->
              <div class="flex items-center justify-between mb-3">
                <div class="flex items-center gap-2 min-w-0 flex-1">
                  <div class="list-icon bg-rose-400">
                    <i class="fas fa-user text-white text-base"></i>
                  </div>
                  <div class="min-w-0">
                    <span class="font-medium text-slate-800 text-sm truncate block">{{ log.username }}</span>
                    <span class="text-xs text-slate-400 font-mono truncate block mt-0.5">{{ log.ip }}</span>
                  </div>
                </div>
                <span class="text-xs text-slate-400 whitespace-nowrap ml-2 flex-shrink-0">{{ formatTimestamp(log.timestamp) }}</span>
              </div>

              <!-- 方法 + URI -->
              <div class="flex items-center gap-2 mb-3">
                <span
                  class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium font-mono flex-shrink-0"
                  :class="methodClass(log.method)"
                >{{ log.method }}</span>
                <code class="text-xs text-slate-700 font-mono truncate">{{ log.uri }}</code>
              </div>

              <!-- Body -->
              <div v-if="log.body" class="flex items-center gap-2 mb-3">
                <span class="text-xs text-slate-400 flex-shrink-0">Body</span>
                <button class="flex items-center gap-1 min-w-0" @click="showDetail(log)">
                  <code class="text-xs text-slate-600 font-mono truncate">{{ formatBody(log.body) }}</code>
                  <i class="fas fa-arrow-up-right-from-square text-xs text-primary-500 flex-shrink-0"></i>
                </button>
              </div>

              <!-- 状态 + 耗时 -->
              <div class="flex items-center gap-3 pt-2 border-t border-slate-100">
                <span v-if="log.success" class="inline-flex items-center gap-1 text-xs font-medium text-emerald-600">
                  <i class="fas fa-circle-check"></i>{{ log.statusCode }}
                </span>
                <span v-else class="inline-flex items-center gap-1 text-xs font-medium text-red-600">
                  <i class="fas fa-circle-xmark"></i>{{ log.statusCode }}
                </span>
                <span class="text-xs text-slate-400">{{ formatDuration(log.duration) }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Body 详情 Modal -->
      <BaseModal v-model="detailOpen" :show-footer="false">
        <template #title>
          <div class="flex items-center gap-2">
            <span
              class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium font-mono"
              :class="methodClass(detailLog?.method || '')"
            >{{ detailLog?.method }}</span>
            <code class="text-sm text-slate-700 font-mono truncate">{{ detailLog?.uri }}</code>
          </div>
        </template>
        <div class="editor-container">
          <Codemirror
            :model-value="detailBody"
            :style="{ height: '50vh' }"
            :extensions="jsonExtensions"
            :disabled="true"
          />
        </div>
      </BaseModal>
    </div>
  </div>
</template>
