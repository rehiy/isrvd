<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { SystemAllSettings, SystemServerSettings, SystemAgentSettings, SystemApisixSettings, SystemDockerSettings, SystemMarketplaceSettings } from '@/service/types'
import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

@Component({})
class Settings extends Vue {
  @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

  // ─── 数据属性 ───
  loading = false
  saving = false
  activeTab: 'server' | 'agent' | 'apisix' | 'docker' | 'marketplace' = 'server'

  server: SystemServerSettings = { debug: false, listenAddr: '', jwtSecret: '', proxyHeaderName: '', rootDirectory: '' }
  agent: SystemAgentSettings = { model: '', baseUrl: '', apiKey: '' }
  apisix: SystemApisixSettings = { adminUrl: '', adminKey: '' }
  docker: SystemDockerSettings = { host: '', containerRoot: '' }
  marketplace: SystemMarketplaceSettings = { url: '' }

  // 敏感字段当前是否已设置（后端返回）
  jwtSecretSet = false
  adminKeySet = false
  agentApiKeySet = false

  // 敏感字段 placeholder
  get jwtSecretPlaceholder() {
    return this.jwtSecretSet ? '已设置（留空保持不变）' : '尚未设置'
  }

  get adminKeyPlaceholder() {
    return this.adminKeySet ? '已设置（留空保持不变）' : '尚未设置'
  }

  get agentApiKeyPlaceholder() {
    return this.agentApiKeySet ? '已设置（留空保持不变）' : '尚未设置'
  }

  // ─── 方法 ───
  async loadSettings() {
    this.loading = true
    try {
      const res = await api.getSettings()
      const payload = res.payload as SystemAllSettings
      // 敏感字段统一置空，仅用标志位驱动 placeholder
      this.server = { ...payload.server, jwtSecret: '' }
      this.agent = { ...payload.agent, apiKey: '' }
      this.apisix = { ...payload.apisix, adminKey: '' }
      this.docker = { ...payload.docker }
      this.marketplace = { ...(payload.marketplace || { url: '' }) }
      this.jwtSecretSet = !!payload.server.jwtSecretSet
      this.adminKeySet = !!payload.apisix.adminKeySet
      this.agentApiKeySet = !!payload.agent.apiKeySet
    } catch (e) {
      this.actions.showNotification('error', '加载配置失败')
    }
    this.loading = false
  }

  async saveAll() {
    this.saving = true
    try {
      await api.updateAllSettings({
        server: this.server,
        agent: this.agent,
        apisix: this.apisix,
        docker: this.docker,
        marketplace: this.marketplace,
      })
      this.actions.showNotification('success', '全部配置已保存，部分项需重启生效')
      this.loadSettings()
    } catch (e) { }
    this.saving = false
  }

  // ─── 生命周期 ───
  mounted() {
    this.loadSettings()
  }
}

export default toNative(Settings)
</script>

<template>
  <div>
    <div class="card mb-4">
      <!-- Toolbar -->
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-4 md:px-6 py-3">
        <!-- 桌面端 -->
        <div class="hidden md:flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-lg bg-slate-700 flex items-center justify-center">
              <i class="fas fa-gear text-white"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">系统设置</h1>
              <p class="text-xs text-slate-500">服务器、Agent、APISIX、Docker 配置</p>
            </div>
          </div>
          <div class="flex items-center gap-3">
            <!-- Tab 切换 -->
            <div class="bg-slate-100 p-1 rounded-lg flex items-center gap-0.5">
              <button type="button" @click="activeTab = 'server'" :class="['px-3 py-1 rounded-md text-xs font-medium transition-colors', activeTab === 'server' ? 'bg-white text-blue-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']">
                <i class="fas fa-server mr-1"></i>服务器
              </button>
              <button type="button" @click="activeTab = 'agent'" :class="['px-3 py-1 rounded-md text-xs font-medium transition-colors', activeTab === 'agent' ? 'bg-white text-emerald-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']">
                <i class="fas fa-robot mr-1"></i>Agent
              </button>
              <button type="button" @click="activeTab = 'apisix'" :class="['px-3 py-1 rounded-md text-xs font-medium transition-colors', activeTab === 'apisix' ? 'bg-white text-indigo-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']">
                <i class="fas fa-cloud mr-1"></i>APISIX
              </button>
              <button type="button" @click="activeTab = 'docker'" :class="['px-3 py-1 rounded-md text-xs font-medium transition-colors', activeTab === 'docker' ? 'bg-white text-sky-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']">
                <i class="fas fa-cube mr-1"></i>Docker
              </button>
              <button type="button" @click="activeTab = 'marketplace'" :class="['px-3 py-1 rounded-md text-xs font-medium transition-colors', activeTab === 'marketplace' ? 'bg-white text-amber-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']">
                <i class="fas fa-store mr-1"></i>市场
              </button>
            </div>
            <button type="button" @click="loadSettings()" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-rotate"></i>刷新
            </button>
          </div>
        </div>
        <!-- 移动端 -->
        <div class="flex md:hidden items-center justify-between">
          <div class="flex items-center gap-3 min-w-0 flex-1">
            <div class="w-9 h-9 rounded-lg bg-slate-700 flex items-center justify-center flex-shrink-0">
              <i class="fas fa-gear text-white"></i>
            </div>
            <div class="min-w-0">
              <h1 class="text-lg font-semibold text-slate-800 truncate">系统设置</h1>
              <p class="text-xs text-slate-500 truncate">服务器、Agent、APISIX、Docker 配置</p>
            </div>
          </div>
          <button type="button" @click="loadSettings()" class="w-9 h-9 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 flex items-center justify-center text-slate-600 transition-colors flex-shrink-0" title="刷新">
            <i class="fas fa-rotate text-sm"></i>
          </button>
        </div>
        <!-- 移动端 Tab -->
        <div class="flex md:hidden mt-3 bg-slate-100 p-1 rounded-lg gap-0.5 overflow-x-auto">
          <button type="button" @click="activeTab = 'server'" :class="['flex-1 min-w-0 px-2 py-1 rounded-md text-xs font-medium transition-colors whitespace-nowrap', activeTab === 'server' ? 'bg-white text-blue-600 shadow-sm' : 'text-slate-500']">
            <i class="fas fa-server mr-1"></i>服务器
          </button>
          <button type="button" @click="activeTab = 'agent'" :class="['flex-1 min-w-0 px-2 py-1 rounded-md text-xs font-medium transition-colors whitespace-nowrap', activeTab === 'agent' ? 'bg-white text-emerald-600 shadow-sm' : 'text-slate-500']">
            <i class="fas fa-robot mr-1"></i>Agent
          </button>
          <button type="button" @click="activeTab = 'apisix'" :class="['flex-1 min-w-0 px-2 py-1 rounded-md text-xs font-medium transition-colors whitespace-nowrap', activeTab === 'apisix' ? 'bg-white text-indigo-600 shadow-sm' : 'text-slate-500']">
            <i class="fas fa-cloud mr-1"></i>APISIX
          </button>
          <button type="button" @click="activeTab = 'docker'" :class="['flex-1 min-w-0 px-2 py-1 rounded-md text-xs font-medium transition-colors whitespace-nowrap', activeTab === 'docker' ? 'bg-white text-sky-600 shadow-sm' : 'text-slate-500']">
            <i class="fas fa-cube mr-1"></i>Docker
          </button>
          <button type="button" @click="activeTab = 'marketplace'" :class="['flex-1 min-w-0 px-2 py-1 rounded-md text-xs font-medium transition-colors whitespace-nowrap', activeTab === 'marketplace' ? 'bg-white text-amber-600 shadow-sm' : 'text-slate-500']">
            <i class="fas fa-store mr-1"></i>市场
          </button>
        </div>
      </div>

      <!-- Loading -->
      <div v-if="loading" class="flex flex-col items-center justify-center py-20">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>

      <form v-else @submit.prevent="saveAll" class="p-4 md:p-6">

        <!-- 服务器配置 -->
        <section v-if="activeTab === 'server'" class="max-w-3xl space-y-4">
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-1.5">Debug 模式</label>
            <select v-model="server.debug" class="input">
              <option :value="false">禁用</option>
              <option :value="true">启用</option>
            </select>
          </div>
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-1.5">监听地址</label>
            <input type="text" v-model="server.listenAddr" placeholder=":8080" class="input" />
            <p class="mt-1 text-xs text-slate-400">HTTP 服务监听端口，例如 :8080 或 127.0.0.1:8080（重启生效）</p>
          </div>
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-1.5">基础目录</label>
            <input type="text" v-model="server.rootDirectory" placeholder="." class="input" />
            <p class="mt-1 text-xs text-slate-400">成员 home 目录及容器数据的基础目录</p>
          </div>
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-1.5">
              JWT 密钥
              <span v-if="jwtSecretSet" class="ml-1.5 inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-green-50 text-green-700"><i class="fas fa-check mr-0.5"></i>已设置</span>
              <span v-else class="ml-1.5 inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-slate-100 text-slate-500">未设置</span>
            </label>
            <input type="password" v-model="server.jwtSecret" :placeholder="jwtSecretPlaceholder" class="input" autocomplete="new-password" />
            <p class="mt-1 text-xs text-slate-400">用于签名登录令牌，修改后所有用户需要重新登录</p>
          </div>
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-1.5">内网代理认证 Header</label>
            <input type="text" v-model="server.proxyHeaderName" placeholder="例如 X-Auth-User（留空禁用）" class="input" />
            <p class="mt-1 text-xs text-slate-400">启用时，将使用上游传入的 Header {{ server.proxyHeaderName }} 值作为登录用户</p>
          </div>
        </section>

        <!-- Agent 配置 -->
        <section v-if="activeTab === 'agent'" class="max-w-3xl space-y-4">
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-1.5">模型名称</label>
            <input type="text" v-model="agent.model" placeholder="例如 gpt-4o-mini" class="input" />
            <p class="mt-1 text-xs text-slate-400">代理转发时强制改写请求体中的 model 字段，留空则不改写</p>
          </div>
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-1.5">基础地址</label>
            <input type="text" v-model="agent.baseUrl" placeholder="https://api.openai.com/v1" class="input" />
            <p class="mt-1 text-xs text-slate-400">OpenAI 兼容的 LLM API 基础地址，留空则禁用代理</p>
          </div>
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-1.5">
              API 密钥
              <span v-if="agentApiKeySet" class="ml-1.5 inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-green-50 text-green-700"><i class="fas fa-check mr-0.5"></i>已设置</span>
              <span v-else class="ml-1.5 inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-slate-100 text-slate-500">未设置</span>
            </label>
            <input type="password" v-model="agent.apiKey" :placeholder="agentApiKeyPlaceholder" class="input" autocomplete="new-password" />
            <p class="mt-1 text-xs text-slate-400">代理转发时以 Bearer 形式注入 Authorization 请求头</p>
          </div>
        </section>

        <!-- APISIX 配置 -->
        <section v-if="activeTab === 'apisix'" class="max-w-3xl space-y-4">
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-1.5">Admin URL</label>
            <input type="text" v-model="apisix.adminUrl" placeholder="http://127.0.0.1:9180" class="input" />
            <p class="mt-1 text-xs text-slate-400">APISIX Admin API 地址</p>
          </div>
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-1.5">
              Admin Key
              <span v-if="adminKeySet" class="ml-1.5 inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-green-50 text-green-700"><i class="fas fa-check mr-0.5"></i>已设置</span>
              <span v-else class="ml-1.5 inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-slate-100 text-slate-500">未设置</span>
            </label>
            <input type="password" v-model="apisix.adminKey" :placeholder="adminKeyPlaceholder" class="input" autocomplete="new-password" />
            <p class="mt-1 text-xs text-slate-400">访问 APISIX Admin API 的密钥</p>
          </div>
        </section>

        <!-- Docker 配置 -->
        <section v-if="activeTab === 'docker'" class="max-w-3xl space-y-4">
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-1.5">Docker Host</label>
            <input type="text" v-model="docker.host" placeholder="unix:///var/run/docker.sock 或 tcp://host:2375" class="input" />
            <p class="mt-1 text-xs text-slate-400">Docker 守护进程地址，留空则使用环境变量</p>
          </div>
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-1.5">容器数据根目录</label>
            <input type="text" v-model="docker.containerRoot" placeholder="containers" class="input" />
            <p class="mt-1 text-xs text-slate-400">用于存放容器数据卷的基础目录（相对于基础目录）</p>
          </div>
        </section>

        <!-- 应用市场配置 -->
        <section v-if="activeTab === 'marketplace'" class="max-w-3xl space-y-4">
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-1.5">站点 URL</label>
            <input type="text" v-model="marketplace.url" placeholder="例如 http://21.214.54.113:8000/" class="input" />
            <p class="mt-1 text-xs text-slate-400">应用市场页面以 iframe 方式嵌入，并通过 postMessage 协议接收安装事件</p>
          </div>
        </section>

        <!-- 统一保存 -->
        <div class="max-w-3xl mt-6 pt-4 border-t border-slate-200 flex flex-col sm:flex-row sm:items-center gap-3">
          <button type="submit" :disabled="saving" class="self-start px-4 py-2 rounded-lg bg-blue-500 hover:bg-blue-600 disabled:opacity-50 text-white text-xs font-medium flex items-center gap-1.5 transition-colors whitespace-nowrap flex-shrink-0">
            <i :class="['fas', saving ? 'fa-spinner fa-spin' : 'fa-save']"></i>{{ saving ? '保存中...' : '保存配置' }}
          </button>
          <p class="text-xs text-slate-400 flex items-start gap-1">
            <i class="fas fa-circle-info mt-0.5 flex-shrink-0"></i>
            <span>保存后立即写入配置文件，监听地址 / JWT 密钥 / Docker Host / APISIX Admin 等项需重启服务生效。</span>
          </p>
        </div>

      </form>
    </div>
  </div>
</template>
