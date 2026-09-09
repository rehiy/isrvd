<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { useConfigStore } from '@/stores'

@Component
class ConfigGateway extends Vue {
    config = useConfigStore()
}

export default toNative(ConfigGateway)
</script>

<template>
  <div class="max-w-4xl space-y-6">
    <!-- APISIX -->
    <section class="space-y-4">
      <div class="config-section-heading">
        <span class="config-section-icon"><i class="fas fa-route"></i></span>
        <div>
          <h2 class="config-section-title">APISIX</h2>
          <p class="config-section-description">Admin API 连接参数</p>
        </div>
      </div>
      <div>
        <label class="form-label">Admin URL</label>
        <input v-model="config.draft.apisix.adminUrl" type="text" placeholder="请输入 Admin URL" class="input" />
        <p class="mt-1 text-xs text-slate-400">APISIX Admin API 地址，默认 http://127.0.0.1:9180</p>
      </div>
      <div>
        <label class="form-label">Admin Key</label>
        <input v-model="config.draft.apisix.adminKey" type="password" placeholder="留空则保持不变" class="input" autocomplete="new-password" />
        <p class="mt-1 text-xs text-slate-400">访问 APISIX Admin API 的密钥</p>
      </div>
    </section>

    <!-- Caddy -->
    <section class="space-y-4">
      <div class="config-section-heading">
        <span class="config-section-icon"><i class="fas fa-globe"></i></span>
        <div>
          <h2 class="config-section-title">Caddy</h2>
          <p class="config-section-description">Admin API 连接参数</p>
        </div>
      </div>
      <div>
        <label class="form-label">Admin URL</label>
        <input v-model="config.draft.caddy.adminUrl" type="text" placeholder="请输入 Admin URL" class="input" />
        <p class="mt-1 text-xs text-slate-400">Caddy Admin API 地址，默认 http://127.0.0.1:2019</p>
      </div>
    </section>

    <!-- Docker -->
    <section class="space-y-4">
      <div class="config-section-heading">
        <span class="config-section-icon"><i class="fas fa-boxes-stacked"></i></span>
        <div>
          <h2 class="config-section-title">Docker</h2>
          <p class="config-section-description">引擎连接与容器根目录</p>
        </div>
      </div>
      <div>
        <label class="form-label">Docker Host</label>
        <input v-model="config.draft.docker.host" type="text" placeholder="请输入 Docker Host" class="input" />
        <p class="mt-1 text-xs text-slate-400">示例：unix:///var/run/docker.sock 或 tcp://host:2375；留空则使用环境变量 DOCKER_HOST</p>
      </div>
      <div>
        <label class="form-label">容器数据根目录</label>
        <input v-model="config.draft.docker.containerRoot" type="text" placeholder="请输入容器数据根目录" class="input" />
        <p class="mt-1 text-xs text-slate-400">用于存放容器数据卷的基础目录（相对于基础目录），默认 containers</p>
      </div>
    </section>
  </div>
</template>
