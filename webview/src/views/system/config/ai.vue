<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { useConfigStore } from '@/stores'

@Component
class ConfigAI extends Vue {
    config = useConfigStore()
}

export default toNative(ConfigAI)
</script>

<template>
  <section class="max-w-3xl space-y-4">
    <div class="flex items-center gap-2">
      <span class="card-icon bg-indigo-100 text-indigo-600"><i class="fas fa-robot"></i></span>
      <div>
        <h2 class="text-sm font-semibold text-slate-700">AI 助手</h2>
        <p class="text-xs text-slate-400 mt-0.5">LLM 代理与模型改写</p>
      </div>
    </div>

    <div>
      <label class="form-label">模型名称</label>
      <input v-model="config.draft.copilot.model" type="text" placeholder="请输入模型名称" class="input" />
      <p class="mt-1 text-xs text-slate-400">代理转发时强制改写请求体中的 model 字段，留空则不改写</p>
    </div>

    <div>
      <label class="form-label">基础地址</label>
      <input v-model="config.draft.copilot.baseUrl" type="text" placeholder="请输入基础地址" class="input" />
      <p class="mt-1 text-xs text-slate-400">示例：https://api.openai.com/v1；OpenAI 兼容的 LLM API 基础地址，留空则禁用代理</p>
    </div>

    <div>
      <label class="form-label">API 密钥</label>
      <input v-model="config.draft.copilot.apiKey" type="password" placeholder="留空则保持不变" class="input" autocomplete="new-password" />
      <p class="mt-1 text-xs text-slate-400">代理转发时以 Bearer 形式注入 Authorization 请求头</p>
    </div>
  </section>
</template>
