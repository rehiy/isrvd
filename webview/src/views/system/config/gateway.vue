<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { useConfigStore } from '@/stores'

import ToggleCard from '@/component/toggle-card.vue'

@Component({ components: { ToggleCard } })
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
        <div>
          <h2 class="config-section-title">APISIX</h2>
          <p class="config-section-description">{{ $t('Admin API 连接参数') }}</p>
        </div>
      </div>
      <div>
        <label class="form-label">Admin URL</label>
        <input v-model="config.draft.apisix.adminUrl" type="text" :placeholder="$t('请输入 Admin URL')" class="input" />
        <p class="mt-1 text-xs text-slate-400">{{ $t('APISIX Admin API 地址，默认 http://127.0.0.1:9180') }}</p>
      </div>
      <div>
        <label class="form-label">Admin Key</label>
        <input v-model="config.draft.apisix.adminKey" type="password" :placeholder="$t('留空则保持不变')" class="input" autocomplete="new-password" />
        <p class="mt-1 text-xs text-slate-400">{{ $t('访问 APISIX Admin API 的密钥') }}</p>
      </div>
    </section>

    <!-- Caddy -->
    <section class="space-y-4">
      <div class="config-section-heading">
        <div>
          <h2 class="config-section-title">Caddy</h2>
          <p class="config-section-description">{{ $t('Admin API 连接参数') }}</p>
        </div>
      </div>
      <div>
        <label class="form-label">Admin URL</label>
        <input v-model="config.draft.caddy.adminUrl" type="text" :placeholder="$t('请输入 Admin URL')" class="input" />
        <p class="mt-1 text-xs text-slate-400">{{ $t('Caddy Admin API 地址，默认 http://127.0.0.1:2019') }}</p>
      </div>
    </section>

    <!-- Docker -->
    <section class="space-y-4">
      <div class="config-section-heading">
        <div>
          <h2 class="config-section-title">Docker</h2>
          <p class="config-section-description">{{ $t('引擎连接、TLS 证书与容器根目录') }}</p>
        </div>
      </div>
      <div>
        <label class="form-label">Docker Host</label>
        <input v-model="config.draft.docker.host" type="text" :placeholder="$t('请输入 Docker Host')" class="input" />
        <p class="mt-1 text-xs text-slate-400">{{ $t('示例：unix:///var/run/docker.sock、tcp://host:2375（明文）或 tcp://host:2376（配合下方 TLS）；留空则使用环境变量 DOCKER_HOST') }}</p>
      </div>
      <ToggleCard v-model="config.draft.docker.tls.enabled" :label="$t('启用 TLS')" :desc="$t('连接远程 Docker / Swarm 时使用；Docker Host 必须是 tcp:// 地址。Swarm 与 Docker 共用此连接，目标节点需为 manager')">
        <div class="space-y-4">
          <ToggleCard v-model="config.draft.docker.tls.skipVerify" :label="$t('跳过服务端证书校验')" :desc="$t('仅用于测试环境；开启后无法防范中间人攻击')" />
          <div>
            <label class="form-label">{{ $t('CA 证书') }}</label>
            <textarea v-model="config.draft.docker.tls.ca" rows="5" placeholder="-----BEGIN CERTIFICATE-----" class="input font-mono text-xs"></textarea>
            <p class="mt-1 text-xs text-slate-400">{{ $t('用于校验服务端证书的 CA（PEM，即 ca.pem）；留空则使用系统根证书') }}</p>
          </div>
          <div>
            <label class="form-label">{{ $t('客户端证书') }}</label>
            <textarea v-model="config.draft.docker.tls.cert" rows="5" placeholder="-----BEGIN CERTIFICATE-----" class="input font-mono text-xs"></textarea>
            <p class="mt-1 text-xs text-slate-400">{{ $t('双向认证（daemon 启用 --tlsverify）时填写（PEM，即 cert.pem），需与私钥成对；清空证书会同时清除私钥') }}</p>
          </div>
          <div>
            <label class="form-label">{{ $t('客户端私钥') }}</label>
            <textarea v-model="config.draft.docker.tls.key" rows="5" :placeholder="$t('留空则保持不变')" class="input font-mono text-xs" autocomplete="off"></textarea>
            <p class="mt-1 text-xs text-slate-400">{{ $t('客户端证书对应的私钥（PEM，即 key.pem）；保存后不会再显示') }}</p>
          </div>
        </div>
      </ToggleCard>
      <div>
        <label class="form-label">{{ $t('容器数据根目录') }}</label>
        <input v-model="config.draft.docker.containerRoot" type="text" :placeholder="$t('请输入容器数据根目录')" class="input" />
        <p class="mt-1 text-xs text-slate-400">{{ $t('用于存放容器数据卷的基础目录（相对于基础目录），默认 containers') }}</p>
      </div>
    </section>
  </div>
</template>
