<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { AllSettings, ServerSettings, AgentSettings, ApisixSettings, DockerSettings } from '@/service/types'
import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

@Component({})
class Settings extends Vue {
  @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

  // ─── 数据属性 ───
  loading = false
  saving = false

  server: ServerSettings = { debug: false, listenAddr: '', jwtSecret: '', proxyHeaderName: '', rootDirectory: '' }
  agent: AgentSettings = { model: '', baseUrl: '', apiKey: '' }
  apisix: ApisixSettings = { adminUrl: '', adminKey: '' }
  docker: DockerSettings = { host: '', containerRoot: '' }

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
      const payload = res.payload as AllSettings
      // 敏感字段统一置空，仅用标志位驱动 placeholder
      this.server = { ...payload.server, jwtSecret: '' }
      this.agent = { ...payload.agent, apiKey: '' }
      this.apisix = { ...payload.apisix, adminKey: '' }
      this.docker = { ...payload.docker }
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
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-lg bg-slate-700 flex items-center justify-center">
              <i class="fas fa-gear text-white"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">系统设置</h1>
              <p class="text-xs text-slate-500">服务器、Agent、APISIX、Docker 配置</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <button type="button" @click="loadSettings()" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-rotate"></i>刷新
            </button>
          </div>
        </div>
      </div>

      <!-- Loading -->
      <div v-if="loading" class="flex flex-col items-center justify-center py-20">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>

      <form v-else @submit.prevent="saveAll" class="p-4 md:p-6 space-y-6">

        <!-- 服务器配置 -->
        <section class="max-w-3xl">
          <div class="flex items-center gap-2 mb-4 pb-2 border-b border-slate-200">
            <div class="w-7 h-7 rounded-lg bg-blue-500 flex items-center justify-center flex-shrink-0">
              <i class="fas fa-server text-white text-xs"></i>
            </div>
            <h2 class="text-sm font-semibold text-slate-800">服务器</h2>
            <span class="text-xs text-slate-400">监听地址、JWT 密钥及代理配置</span>
          </div>
          <div class="space-y-4">
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
            <div class="flex items-center gap-2">
              <input id="debugSwitch" type="checkbox" v-model="server.debug" class="w-4 h-4 rounded" />
              <label for="debugSwitch" class="text-sm text-slate-700 cursor-pointer">启用 Debug 模式</label>
            </div>
          </div>
        </section>

        <!-- Agent 配置 -->
        <section class="max-w-3xl">
          <div class="flex items-center gap-2 mb-4 pb-2 border-b border-slate-200">
            <div class="w-7 h-7 rounded-lg bg-emerald-500 flex items-center justify-center flex-shrink-0">
              <i class="fas fa-robot text-white text-xs"></i>
            </div>
            <h2 class="text-sm font-semibold text-slate-800">Agent</h2>
            <span class="text-xs text-slate-400">LLM 模型、API 地址及密钥</span>
          </div>
          <div class="space-y-4">
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
          </div>
        </section>

        <!-- APISIX 配置 -->
        <section class="max-w-3xl">
          <div class="flex items-center gap-2 mb-4 pb-2 border-b border-slate-200">
            <div class="w-7 h-7 rounded-lg bg-indigo-500 flex items-center justify-center flex-shrink-0">
              <i class="fas fa-cloud text-white text-xs"></i>
            </div>
            <h2 class="text-sm font-semibold text-slate-800">APISIX</h2>
            <span class="text-xs text-slate-400">Admin API 连接信息</span>
          </div>
          <div class="space-y-4">
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
          </div>
        </section>

        <!-- Docker 配置 -->
        <section class="max-w-3xl">
          <div class="flex items-center gap-2 mb-4 pb-2 border-b border-slate-200">
            <div class="w-7 h-7 rounded-lg bg-sky-500 flex items-center justify-center flex-shrink-0">
              <i class="fas fa-cube text-white text-xs"></i>
            </div>
            <h2 class="text-sm font-semibold text-slate-800">Docker</h2>
            <span class="text-xs text-slate-400">守护进程及容器数据目录</span>
          </div>
          <div class="space-y-4">
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
          </div>
        </section>

        <!-- 统一保存 -->
        <div class="max-w-3xl pt-4 border-t border-slate-200 flex flex-col sm:flex-row sm:items-center gap-3">
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
