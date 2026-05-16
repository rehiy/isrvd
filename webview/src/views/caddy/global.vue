<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { CaddyGlobal } from '@/service/types'

import { usePortal } from '@/stores'

@Component
class CaddyGlobalConfig extends Vue {
    portal = usePortal()

    loading = false
    saving  = false

    // ─── 表单字段 ───
    logLevel      = ''
    logFormat     = ''
    email         = ''
    acmeCA        = ''
    localCerts    = false
    onDemandTLS   = false
    autoHttpsDisable          = false
    autoHttpsDisableRedirects = false
    gracePeriod   = ''

    // ─── 生命周期 ───
    mounted() {
        this.load()
    }

    // ─── 方法 ───
    async load() {
        this.loading = true
        try {
            const data: CaddyGlobal = (await api.caddyGlobal()).payload || {}
            this.logLevel      = data.logLevel      ?? ''
            this.logFormat     = data.logFormat     ?? ''
            this.email         = data.email         ?? ''
            this.acmeCA        = data.acmeCA        ?? ''
            this.localCerts    = data.localCerts    ?? false
            this.onDemandTLS   = data.onDemandTLS   ?? false
            this.autoHttpsDisable          = data.autoHttpsDisable          ?? false
            this.autoHttpsDisableRedirects = data.autoHttpsDisableRedirects ?? false
            this.gracePeriod   = data.gracePeriod   ?? ''
        } catch {
            this.portal.showNotification('error', '加载全局选项失败')
        } finally {
            this.loading = false
        }
    }

    async save() {
        const payload: CaddyGlobal = {
            logLevel:      this.logLevel      || undefined,
            logFormat:     this.logFormat     || undefined,
            email:         this.email         || undefined,
            acmeCA:        this.acmeCA        || undefined,
            localCerts:    this.localCerts    || undefined,
            onDemandTLS:   this.onDemandTLS   || undefined,
            autoHttpsDisable:          this.autoHttpsDisable          || undefined,
            autoHttpsDisableRedirects: this.autoHttpsDisableRedirects || undefined,
            gracePeriod:   this.gracePeriod   || undefined,
        }
        this.saving = true
        try {
            await api.caddyGlobalUpdate(payload)
            this.portal.showNotification('success', '全局选项已保存')
            this.load()
        } catch (e: unknown) {
            this.portal.showNotification('error', (e instanceof Error ? e.message : '') || '保存失败')
        } finally {
            this.saving = false
        }
    }
}

export default toNative(CaddyGlobalConfig)
</script>

<template>
  <div>
    <div class="card mb-4">

      <!-- 页头 -->
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-4 md:px-6 py-3">
        <!-- 桌面端 -->
        <div class="hidden md:flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-lg bg-violet-500 flex items-center justify-center">
              <i class="fas fa-sliders text-white"></i>
            </div>
            <div class="min-w-0">
              <h1 class="text-lg font-semibold text-slate-800 truncate">Caddy 全局选项</h1>
              <p class="text-xs text-slate-500 truncate">配置 TLS 自动化、日志级别与 HTTP 服务器参数</p>
            </div>
          </div>
          <div class="flex items-center gap-2 flex-shrink-0">
            <button
              class="btn btn-sm btn-secondary"
              @click="load()"
            ><i class="fas fa-rotate"></i>刷新</button>
            <button
              v-if="portal.hasPerm('PUT /api/caddy/global')"
              :disabled="saving || loading"
              class="btn btn-sm btn-violet"
              @click="save()"
            >
              <i v-if="saving" class="fas fa-spinner fa-spin"></i>
              <i v-else class="fas fa-floppy-disk"></i>
              {{ saving ? '保存中...' : '保存配置' }}
            </button>
          </div>
        </div>
        <!-- 移动端 -->
        <div class="flex md:hidden items-center justify-between">
          <div class="flex items-center gap-3 min-w-0 flex-1">
            <div class="w-9 h-9 rounded-lg bg-violet-500 flex items-center justify-center flex-shrink-0">
              <i class="fas fa-sliders text-white"></i>
            </div>
            <div class="min-w-0">
              <h1 class="text-lg font-semibold text-slate-800 truncate">Caddy 全局选项</h1>
              <p class="text-xs text-slate-500 truncate">TLS、日志与 HTTP 参数</p>
            </div>
          </div>
          <div class="flex items-center gap-1 flex-shrink-0">
            <button class="btn btn-sm btn-secondary w-9 h-9 !px-0" title="刷新" @click="load()">
              <i class="fas fa-rotate text-sm"></i>
            </button>
            <button
              v-if="portal.hasPerm('PUT /api/caddy/global')"
              :disabled="saving || loading"
              class="btn btn-sm btn-violet w-9 h-9 !px-0"
              title="保存"
              @click="save()"
            ><i class="fas fa-floppy-disk text-sm"></i></button>
          </div>
        </div>
      </div>

      <!-- 内容区 -->
      <div v-if="loading" class="flex flex-col items-center justify-center py-20">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>

      <div v-else class="p-4 md:p-6 divide-y divide-slate-100">

        <!-- 证书签发 -->
        <div class="pb-6">
          <h2 class="text-sm font-semibold text-slate-700 mb-4 flex items-center gap-2">
            <i class="fas fa-certificate text-violet-500"></i>证书签发
          </h2>
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4" :class="{ 'opacity-50 pointer-events-none': localCerts }">
            <div>
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">ACME 邮箱</label>
              <input v-model="email" type="email" class="input" placeholder="your@email.com" />
              <p class="text-xs text-slate-400 mt-1">Let's Encrypt 证书申请邮箱，启用 HTTPS 自动签发时必填</p>
            </div>
            <div>
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">ACME 目录 URL</label>
              <input v-model="acmeCA" type="text" class="input" placeholder="留空使用 Let's Encrypt" />
              <p class="text-xs text-slate-400 mt-1">自定义 CA 目录，如 ZeroSSL 或私有 ACME CA</p>
            </div>
          </div>
          <div class="space-y-3">
            <div>
              <label class="flex items-center gap-2 cursor-pointer select-none w-fit">
                <input v-model="localCerts" type="checkbox" class="rounded border-slate-300 text-violet-600 focus:ring-violet-500" />
                <span class="text-sm text-slate-700">使用本地自签证书（internal issuer）</span>
              </label>
              <p class="text-xs text-slate-400 mt-1">不走 ACME，由 Caddy 自动签发本地信任证书；启用后 ACME 邮箱和目录设置将被忽略</p>
            </div>
            <div>
              <label class="flex items-center gap-2 cursor-pointer select-none w-fit">
                <input v-model="onDemandTLS" type="checkbox" class="rounded border-slate-300 text-violet-600 focus:ring-violet-500" />
                <span class="text-sm text-slate-700">启用 On-Demand TLS</span>
              </label>
              <p class="text-xs text-slate-400 mt-1">连接时动态申请证书，适合域名数量不固定的多租户场景；生产环境需配合 <code class="px-1 bg-slate-100 rounded">ask</code> 端点防滥用</p>
            </div>
          </div>
        </div>

        <!-- HTTPS 行为 -->
        <div class="py-6">
          <h2 class="text-sm font-semibold text-slate-700 mb-4 flex items-center gap-2">
            <i class="fas fa-arrow-right-arrow-left text-violet-500"></i>HTTPS 行为
          </h2>
          <div class="space-y-3">
            <div>
              <label class="flex items-center gap-2 cursor-pointer select-none w-fit">
                <input v-model="autoHttpsDisable" type="checkbox" class="rounded border-slate-300 text-violet-600 focus:ring-violet-500" />
                <span class="text-sm text-slate-700">禁用自动 HTTPS</span>
              </label>
              <p class="text-xs text-slate-400 mt-1">勾选后 Caddy 不再自动申请或管理证书；取消勾选并配置好 ACME 邮箱后，Caddy 将自动为路由中的域名申请和续签证书</p>
            </div>
            <div>
              <label class="flex items-center gap-2 cursor-pointer select-none w-fit">
                <input v-model="autoHttpsDisableRedirects" type="checkbox" class="rounded border-slate-300 text-violet-600 focus:ring-violet-500" />
                <span class="text-sm text-slate-700">禁用 HTTP→HTTPS 自动跳转</span>
              </label>
              <p class="text-xs text-slate-400 mt-1">不插入 301 重定向路由；适合需要同时提供 HTTP 和 HTTPS 服务的场景</p>
            </div>
          </div>
        </div>

        <!-- 系统 -->
        <div class="py-6">
          <h2 class="text-sm font-semibold text-slate-700 mb-4 flex items-center gap-2">
            <i class="fas fa-gear text-violet-500"></i>系统
          </h2>
          <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div>
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">日志级别</label>
              <select v-model="logLevel" class="input">
                <option value="">默认（INFO）</option>
                <option value="DEBUG">DEBUG</option>
                <option value="INFO">INFO</option>
                <option value="WARN">WARN</option>
                <option value="ERROR">ERROR</option>
              </select>
              <p class="text-xs text-slate-400 mt-1">全局默认日志级别</p>
            </div>
            <div>
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">日志格式</label>
              <select v-model="logFormat" class="input">
                <option value="">默认（console）</option>
                <option value="console">console</option>
                <option value="json">json</option>
              </select>
              <p class="text-xs text-slate-400 mt-1">结构化日志输出格式</p>
            </div>
            <div>
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">优雅关闭等待</label>
              <input v-model="gracePeriod" type="text" class="input" placeholder="例如 10s、30s" />
              <p class="text-xs text-slate-400 mt-1">重载/关闭时等待现有连接结束的最长时间</p>
            </div>
          </div>
        </div>

        <!-- 底部提示 -->
        <p class="text-xs text-slate-400 flex items-start gap-1.5 pt-4">
          <i class="fas fa-circle-info mt-0.5 flex-shrink-0"></i>
          <span>保存后通过 <code class="px-1 bg-slate-100 rounded">POST /load</code> 整体原子替换 Caddy 运行配置，立即生效。</span>
        </p>
      </div>
    </div>
  </div>
</template>
