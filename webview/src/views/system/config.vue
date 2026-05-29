<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { AllConfig, ServerConfig, AgentConfig, OIDCConfig, ApisixConfig, CaddyConfig, DockerConfig, MonitorConfig, MarketplaceConfig, LinkConfig } from '@/service/types'

import IconSelect from '@/component/icon-select.vue'

@Component({ components: { IconSelect } })
class Config extends Vue {
  portal = usePortal()

  // ─── 数据属性 ───
  loading = false
  saving = false
  activeTab: 'server' | 'agent' | 'oidc' | 'app' | 'links' = 'server'

  server: ServerConfig = { debug: false, listenAddr: '', jwtExpiration: 86400, maxUploadSize: 104857600, proxyHeaderName: '', proxyTrustedCIDRs: [], rootDirectory: '', allowedOrigins: [] }
  allowedOriginsText = ''
  proxyTrustedCIDRsText = ''
  monitor: MonitorConfig = { interval: 0 }
  oidc: OIDCConfig = { enabled: false, issuerUrl: '', clientId: '', redirectUrl: '', usernameClaim: 'sub', scopes: ['openid', 'profile', 'email'] }
  oidcScopes = 'openid profile email'
  agent: AgentConfig = { model: '', baseUrl: '' }
  apisix: ApisixConfig = { adminUrl: '' }
  caddy: CaddyConfig = { adminUrl: '' }
  docker: DockerConfig = { host: '', containerRoot: '' }
  marketplace: MarketplaceConfig = { url: '' }
  links: LinkConfig[] = []

  // ─── 方法 ───
  async loadConfig(reload = false) {
    this.loading = true
    try {
      const res = await api.systemConfig(reload ? { reload: 'true' } : undefined)
      const payload = res.payload as AllConfig
      this.server = { ...payload.server }
      this.allowedOriginsText = (this.server.allowedOrigins || []).join('\n')
      this.proxyTrustedCIDRsText = (this.server.proxyTrustedCIDRs || []).join('\n')
      this.monitor = { ...(payload.monitor || { interval: 0 }) }
      this.oidc = { ...(payload.oidc || { enabled: false, issuerUrl: '', clientId: '', redirectUrl: '', usernameClaim: 'sub', scopes: ['openid', 'profile', 'email'] }) }
      this.oidcScopes = (this.oidc.scopes || []).join(' ')
      this.agent = { ...payload.agent }
      this.apisix = { ...payload.apisix }
      this.caddy = { ...(payload.caddy || { adminUrl: '' }) }
      this.docker = { ...payload.docker }
      this.marketplace = { ...(payload.marketplace || { url: '' }) }
      this.links = payload.links ? payload.links.map(l => ({ ...l })) : []
      if (reload) {
        this.portal.showNotification('success', '配置已从文件重载配置')
      }
    } catch {
      this.portal.showNotification('error', reload ? '重载配置失败' : '加载配置失败')
    } finally {
      this.loading = false
    }
  }

  async saveAll() {
    this.saving = true
    try {
      await api.systemConfigUpdate({
        server: { ...this.server, allowedOrigins: this.allowedOriginsText.split(/\s+/).filter(Boolean), proxyTrustedCIDRs: this.proxyTrustedCIDRsText.split(/\s+/).filter(Boolean) },
        oidc: { ...this.oidc, scopes: this.oidcScopes.split(/\s+/).filter(Boolean) },
        agent: this.agent,
        apisix: this.apisix,
        caddy: this.caddy,
        docker: this.docker,
        monitor: this.monitor,
        marketplace: this.marketplace,
        links: this.links,
      })
this.portal.showNotification('success', '全部配置已保存，监听地址变更需重启生效')
      this.loadConfig()
    } catch {
    } finally {
      this.saving = false
    }
  }

  addLink() {
    this.links.push({ label: '', url: '', icon: 'fas fa-link' })
  }

  removeLink(index: number) {
    this.links.splice(index, 1)
  }

  // ─── 生命周期 ───
  mounted() {
    this.loadConfig()
  }
}

export default toNative(Config)
</script>

<template>
  <div class="card">
    <!-- Toolbar -->
    <div class="card-toolbar">
      <!-- 桌面端 -->
      <div class="hidden md:flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="page-icon bg-indigo-500">
            <i class="fas fa-gear text-white"></i>
          </div>
          <div>
            <h1 class="text-lg font-semibold text-slate-800">系统配置</h1>
            <p class="text-xs text-slate-500">管理服务器、认证、网关与容器引擎参数</p>
          </div>
        </div>
        <div class="flex items-center gap-3">
          <!-- Tab 切换 -->
          <div class="tab-group">
            <button type="button" :class="['tab-btn', activeTab === 'server' ? 'tab-btn-active text-blue-600' : 'tab-btn-inactive']" @click="activeTab = 'server'">
              <i class="fas fa-server"></i>全局
            </button>
            <button type="button" :class="['tab-btn', activeTab === 'oidc' ? 'tab-btn-active text-purple-600' : 'tab-btn-inactive']" @click="activeTab = 'oidc'">
              <i class="fas fa-id-card"></i>OIDC
            </button>
            <button type="button" :class="['tab-btn', activeTab === 'agent' ? 'tab-btn-active text-emerald-600' : 'tab-btn-inactive']" @click="activeTab = 'agent'">
              <i class="fas fa-robot"></i>Agent
            </button>
            <button type="button" :class="['tab-btn', activeTab === 'app' ? 'tab-btn-active text-indigo-600' : 'tab-btn-inactive']" @click="activeTab = 'app'">
              <i class="fas fa-layer-group"></i>应用
            </button>
            <button type="button" :class="['tab-btn', activeTab === 'links' ? 'tab-btn-active text-orange-600' : 'tab-btn-inactive']" @click="activeTab = 'links'">
              <i class="fas fa-link"></i>导航
            </button>
          </div>
          <button type="button" class="btn btn-indigo" @click="loadConfig(true)">
            <i class="fas fa-rotate"></i>重载配置
          </button>
        </div>
      </div>
      <!-- 移动端 -->
      <div class="flex md:hidden items-center justify-between">
        <div class="flex items-center gap-3 min-w-0 flex-1">
          <div class="page-icon bg-indigo-500">
            <i class="fas fa-gear text-white"></i>
          </div>
          <div class="min-w-0">
            <h1 class="text-lg font-semibold text-slate-800 truncate">系统配置</h1>
            <p class="text-xs text-slate-500 truncate">服务器、认证、网关与容器参数</p>
          </div>
        </div>
        <button type="button" class="btn btn-indigo w-9 h-9 !px-0" title="重载配置" @click="loadConfig(true)">
          <i class="fas fa-rotate text-sm"></i>
        </button>
      </div>
      <!-- 移动端 Tab -->
      <div class="tab-group md:hidden mt-3 overflow-x-auto">
        <button type="button" :class="['tab-btn flex-1 justify-center whitespace-nowrap', activeTab === 'server' ? 'tab-btn-active text-blue-600' : 'tab-btn-inactive']" @click="activeTab = 'server'">
          <i class="fas fa-server"></i>全局
        </button>
        <button type="button" :class="['tab-btn flex-1 justify-center whitespace-nowrap', activeTab === 'oidc' ? 'tab-btn-active text-purple-600' : 'tab-btn-inactive']" @click="activeTab = 'oidc'">
          <i class="fas fa-id-card"></i>OIDC
        </button>
        <button type="button" :class="['tab-btn flex-1 justify-center whitespace-nowrap', activeTab === 'agent' ? 'tab-btn-active text-emerald-600' : 'tab-btn-inactive']" @click="activeTab = 'agent'">
          <i class="fas fa-robot"></i>Agent
        </button>
        <button type="button" :class="['tab-btn flex-1 justify-center whitespace-nowrap', activeTab === 'app' ? 'tab-btn-active text-indigo-600' : 'tab-btn-inactive']" @click="activeTab = 'app'">
          <i class="fas fa-layer-group"></i>应用
        </button>
        <button type="button" :class="['tab-btn flex-1 justify-center whitespace-nowrap', activeTab === 'links' ? 'tab-btn-active text-orange-600' : 'tab-btn-inactive']" @click="activeTab = 'links'">
          <i class="fas fa-link"></i>导航
        </button>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="empty-state">
      <div class="w-12 h-12 spinner mb-3"></div>
      <p class="text-slate-500">加载中...</p>
    </div>

    <form v-else class="p-4 md:p-6" @submit.prevent="saveAll">
      <!-- 服务器配置 -->
      <section v-if="activeTab === 'server'" class="max-w-3xl space-y-4">
        <div>
          <label class="form-label">Debug 模式</label>
          <select v-model="server.debug" class="input">
            <option :value="false">禁用</option>
            <option :value="true">启用</option>
          </select>
        </div>
        <div>
          <label class="form-label">监听地址</label>
          <input v-model="server.listenAddr" type="text" placeholder=":8080" class="input" />
          <p class="mt-1 text-xs text-slate-400">HTTP 服务监听端口，例如 :8080 或 127.0.0.1:8080（重启生效）</p>
        </div>
        <div>
          <label class="form-label">JWT 认证密钥</label>
          <input v-model="server.jwtSecret" type="password" placeholder="留空保持不变" class="input" autocomplete="new-password" />
          <p class="mt-1 text-xs text-slate-400">用于签名登录令牌，修改后所有用户需要重新登录</p>
        </div>
        <div>
          <label class="form-label">JWT 有效期（秒）</label>
          <input v-model.number="server.jwtExpiration" type="number" min="60" placeholder="86400" class="input" />
          <p class="mt-1 text-xs text-slate-400">登录令牌的有效期，默认 86400（24 小时）</p>
        </div>
        <div>
          <label class="form-label">内网代理认证 Header</label>
          <input v-model="server.proxyHeaderName" type="text" placeholder="例如 X-Auth-User（留空禁用）" class="input" />
          <p class="mt-1 text-xs text-slate-400">启用时，将使用上游传入的 Header {{ server.proxyHeaderName }} 值作为登录用户</p>
        </div>
        <div>
          <label class="form-label">代理可信来源 CIDR</label>
          <textarea v-model="proxyTrustedCIDRsText" rows="3" placeholder="127.0.0.1/32&#10;10.0.0.0/8" class="input font-mono text-xs"></textarea>
          <p class="mt-1 text-xs text-slate-400">每行一个 CIDR，仅列出的来源 IP 允许使用代理认证；留空则仅信任本机（127.0.0.1）</p>
        </div>
        <div>
          <label class="form-label">允许的跨域 Origin</label>
          <textarea v-model="allowedOriginsText" rows="3" placeholder="https://example.com&#10;https://*.example.com" class="input font-mono text-xs"></textarea>
          <p class="mt-1 text-xs text-slate-400">每行一个，支持通配符 *；留空则不限制</p>
        </div>
        <div>
          <label class="form-label">文件上传大小限制（字节）</label>
          <input v-model.number="server.maxUploadSize" type="number" min="0" placeholder="104857600" class="input" />
          <p class="mt-1 text-xs text-slate-400">单次上传的最大文件大小，默认 104857600（100 MB）</p>
        </div>
        <div>
          <label class="form-label">基础目录</label>
          <input v-model="server.rootDirectory" type="text" placeholder="." class="input" />
          <p class="mt-1 text-xs text-slate-400">成员家目录及容器数据的基础目录</p>
        </div>
        <div>
          <label class="form-label">监控采集间隔</label>
          <select v-model.number="monitor.interval" class="input">
            <option :value="0">禁用</option>
            <option :value="5">5 秒</option>
            <option :value="15">15 秒</option>
            <option :value="30">30 秒</option>
            <option :value="60">60 秒</option>
          </select>
          <p class="mt-1 text-xs text-slate-400">系统与容器监控数据的采集频率，禁用后不再写入监控文件</p>
        </div>
      </section>

      <!-- OIDC 配置 -->
      <section v-if="activeTab === 'oidc'" class="max-w-3xl space-y-4">
        <div>
          <label class="form-label">启用 OIDC 登录</label>
          <select v-model="oidc.enabled" class="input">
            <option :value="false">禁用</option>
            <option :value="true">启用</option>
          </select>
        </div>
        <div>
          <label class="form-label">颁发者地址</label>
          <input v-model="oidc.issuerUrl" type="text" placeholder="https://idp.example.com" class="input" />
          <p class="mt-1 text-xs text-slate-400">Issuer URL；用于自动发现 authorization_endpoint、token_endpoint、jwks_uri 等元数据；保存后立即生效</p>
        </div>
        <div>
          <label class="form-label">客户端 ID</label>
          <input v-model="oidc.clientId" type="text" placeholder="iSrvd" class="input" />
          <p class="mt-1 text-xs text-slate-400">Client ID；在 OIDC Provider 处注册应用时获得</p>
        </div>
        <div>
          <label class="form-label">客户端密钥</label>
          <input v-model="oidc.clientSecret" type="password" placeholder="留空保持不变" class="input" autocomplete="new-password" />
          <p class="mt-1 text-xs text-slate-400">Client Secret；留空表示保持原值不变</p>
        </div>
        <div>
          <label class="form-label">回调地址</label>
          <input v-model="oidc.redirectUrl" type="text" placeholder="https://iSrvd.example.com/api/account/oidc/callback" class="input" />
          <p class="mt-1 text-xs text-slate-400">Redirect URL；开发环境可留空自动生成，生产环境建议填写固定 HTTPS 回调地址</p>
        </div>
        <div>
          <label class="form-label">用户名字段</label>
          <input v-model="oidc.usernameClaim" type="text" placeholder="sub" class="input" />
          <p class="mt-1 text-xs text-slate-400">Username Claim；默认 sub，该字段的值必须与 members.username 完全一致，用户不存在时登录失败</p>
        </div>
        <div>
          <label class="form-label">登录按钮名称</label>
          <input v-model="oidc.loginLabel" type="text" placeholder="使用 OIDC 登录" class="input" />
          <p class="mt-1 text-xs text-slate-400">自定义 OIDC 登录按钮显示名称；留空则使用默认文案"使用 OIDC 登录"</p>
        </div>
        <div>
          <label class="form-label">授权范围</label>
          <input v-model="oidcScopes" type="text" placeholder="openid profile email" class="input" />
          <p class="mt-1 text-xs text-slate-400">Scopes；以空格分隔，系统会自动确保包含 openid</p>
        </div>
      </section>

      <!-- Agent 配置 -->
      <section v-if="activeTab === 'agent'" class="max-w-3xl space-y-4">
        <div>
          <label class="form-label">模型名称</label>
          <input v-model="agent.model" type="text" placeholder="例如 gpt-4o-mini" class="input" />
          <p class="mt-1 text-xs text-slate-400">代理转发时强制改写请求体中的 model 字段，留空则不改写</p>
        </div>
        <div>
          <label class="form-label">基础地址</label>
          <input v-model="agent.baseUrl" type="text" placeholder="https://api.openai.com/v1" class="input" />
          <p class="mt-1 text-xs text-slate-400">OpenAI 兼容的 LLM API 基础地址，留空则禁用代理</p>
        </div>
        <div>
          <label class="form-label">API 密钥</label>
          <input v-model="agent.apiKey" type="password" placeholder="留空保持不变" class="input" autocomplete="new-password" />
          <p class="mt-1 text-xs text-slate-400">代理转发时以 Bearer 形式注入 Authorization 请求头</p>
        </div>
      </section>

      <!-- 应用配置（APISIX + Docker） -->
      <section v-if="activeTab === 'app'" class="max-w-3xl space-y-4">
        <p class="text-sm font-medium text-slate-500">APISIX</p>
        <div>
          <label class="form-label">Admin URL</label>
          <input v-model="apisix.adminUrl" type="text" placeholder="http://127.0.0.1:9180" class="input" />
          <p class="mt-1 text-xs text-slate-400">APISIX Admin API 地址</p>
        </div>
        <div>
          <label class="form-label">Admin Key</label>
          <input v-model="apisix.adminKey" type="password" placeholder="留空保持不变" class="input" autocomplete="new-password" />
          <p class="mt-1 text-xs text-slate-400">访问 APISIX Admin API 的密钥</p>
        </div>
        <div class="border-t border-slate-200 pt-4">
          <p class="text-sm font-medium text-slate-500 mb-4">Caddy</p>
          <div class="space-y-4">
            <div>
              <label class="form-label">Admin URL</label>
              <input v-model="caddy.adminUrl" type="text" placeholder="http://127.0.0.1:2019" class="input" />
              <p class="text-xs text-slate-400 mt-1">Caddy Admin API 地址（默认 127.0.0.1:2019）</p>
            </div>
          </div>
        </div>
        <div class="border-t border-slate-200 pt-4">
          <p class="text-sm font-medium text-slate-500 mb-4">Docker</p>
          <div class="space-y-4">
            <div>
              <label class="form-label">Docker Host</label>
              <input v-model="docker.host" type="text" placeholder="unix:///var/run/docker.sock 或 tcp://host:2375" class="input" />
              <p class="mt-1 text-xs text-slate-400">Docker 守护进程地址，留空则使用环境变量</p>
            </div>
            <div>
              <label class="form-label">容器数据根目录</label>
              <input v-model="docker.containerRoot" type="text" placeholder="containers" class="input" />
              <p class="mt-1 text-xs text-slate-400">用于存放容器数据卷的基础目录（相对于基础目录）</p>
            </div>
          </div>
        </div>
        <div class="border-t border-slate-200 pt-4">
          <p class="text-sm font-medium text-slate-500 mb-4">应用市场</p>
          <div>
            <label class="form-label">站点 URL</label>
            <input v-model="marketplace.url" type="text" placeholder="例如 http://21.214.54.113:8000/" class="input" />
            <p class="mt-1 text-xs text-slate-400">应用市场页面以 iframe 方式嵌入，并通过 postMessage 协议接收安装事件</p>
          </div>
        </div>
      </section>

      <!-- Links 配置 -->
      <section v-if="activeTab === 'links'" class="max-w-3xl space-y-3">
        <!-- 列标题（仅在有数据时显示） -->
        <div v-if="links.length" class="hidden sm:grid sm:grid-cols-[1fr_2fr_1.2fr_auto] gap-3 px-0.5">
          <span class="text-xs font-medium text-slate-500">名称</span>
          <span class="text-xs font-medium text-slate-500">URL</span>
          <span class="text-xs font-medium text-slate-500">图标</span>
          <span></span>
        </div>
        <div v-for="(link, index) in links" :key="index" class="grid grid-cols-1 sm:grid-cols-[1fr_2fr_1.2fr_auto] gap-3 items-center">
          <input v-model="link.label" type="text" placeholder="例如 Grafana" class="input" />
          <input v-model="link.url" type="text" placeholder="https://example.com" class="input" />
          <!-- 图标选择器 -->
          <IconSelect v-model="link.icon" />
          <button type="button" class="btn-icon btn-icon-red w-11 h-11" @click="removeLink(index)">
            <i class="fas fa-trash-can text-sm"></i>
          </button>
        </div>
        <div class="grid grid-cols-1 sm:grid-cols-[1fr_2fr_1.2fr_auto] gap-3 items-center">
          <div class="hidden sm:block sm:col-span-3"></div>
          <button type="button" class="btn-icon btn-icon-slate w-11 h-11 border border-dashed border-slate-300" @click="addLink()">
            <i class="fas fa-plus text-sm"></i>
          </button>
        </div>
      </section>

      <!-- 统一保存 -->
      <div class="max-w-3xl mt-6 pt-4 border-t border-slate-200 flex flex-col sm:flex-row sm:items-center gap-3">
        <button type="submit" :disabled="saving" class="btn btn-indigo rounded-xl whitespace-nowrap flex-shrink-0 self-start">
          <i v-if="saving" class="fas fa-spinner fa-spin"></i>
          <i v-else class="fas fa-save"></i>
          <span>{{ saving ? '保存中...' : '保存配置' }}</span>
        </button>
        <p class="text-xs text-slate-400 flex items-start gap-1">
          <i class="fas fa-circle-info mt-0.5 flex-shrink-0"></i>
          <span>保存后立即写入配置文件，监听地址变更需重启服务生效。</span>
        </p>
      </div>
    </form>
  </div>
</template>