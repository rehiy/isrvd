<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { useConfigStore } from '@/stores'

import type { WebhookConfig } from '@/service/types'

@Component
class ConfigNotify extends Vue {
    config = useConfigStore()

    // Go text/template 变量说明；定义为变量后插值输出，避免与 Vue 的 {{ }} 语法冲突
    templateHint = '留空则发送标准 JSON；可用变量 {{.Title}}、{{.Message}}、{{.Level}}、{{.Event}}、{{.Timestamp}}'

    addWebhook() {
        this.config.draft.notify.webhooks.push({ name: '', url: '', template: '' })
    }

    removeWebhook(index: number) {
        this.config.draft.notify.webhooks.splice(index, 1)
    }

    addRule() {
        this.config.draft.notify.rules.push({ metric: 'cpu', threshold: 80, duration: 3 })
    }

    removeRule(index: number) {
        this.config.draft.notify.rules.splice(index, 1)
    }

    applyWebhookTemplate(hook: WebhookConfig, event: Event) {
        const templates: Record<string, string> = {
            dingtalk: '{"msgtype":"text","text":{"content":"isrvd 告警\\n{{.Title}}\\n{{.Message}}"} }',
            feishu: '{"msg_type":"text","content":{"text":"isrvd 告警\\n{{.Title}}\\n{{.Message}}"} }',
            wecom: '{"msgtype":"text","text":{"content":"isrvd 告警\\n{{.Title}}\\n{{.Message}}"} }',
            slack: '{"text":"isrvd 告警：{{.Title}}\\n{{.Message}}"}',
        }
        const select = event.target as HTMLSelectElement
        const kind = select.value
        if (templates[kind]) hook.template = templates[kind]
        // 复位选择器，保证再次选择同一渠道仍能触发
        select.value = ''
    }
}

export default toNative(ConfigNotify)
</script>

<template>
  <section class="max-w-4xl space-y-6">
    <div class="flex items-center gap-2">
      <span class="card-icon bg-indigo-100 text-indigo-600"><i class="fas fa-bell"></i></span>
      <div>
        <h2 class="text-sm font-semibold text-slate-700">告警通知</h2>
        <p class="text-xs text-slate-400 mt-0.5">规则触发和恢复时推送到全部已配置通道</p>
      </div>
    </div>

    <fieldset class="space-y-4">
      <legend class="section-title w-full">Webhook 通道</legend>
      <div v-if="config.draft.notify.webhooks.length === 0" class="empty-note">暂无通道，点击下方按钮添加</div>
      <div v-else class="space-y-4">
        <div v-for="(hook, index) in config.draft.notify.webhooks" :key="index" class="panel-frame">
          <div class="card-body space-y-4">
            <div class="flex items-center justify-between">
              <span class="text-sm font-semibold text-slate-700">通道 {{ index + 1 }}</span>
              <button type="button" class="btn-icon btn-icon-red" title="删除通道" @click="removeWebhook(index)">
                <i class="fas fa-trash-can text-xs"></i>
              </button>
            </div>
            <div>
              <label class="form-label">通道名称</label>
              <input v-model="hook.name" type="text" required placeholder="如：钉钉告警" class="input" />
            </div>
            <div>
              <label class="form-label">接收地址</label>
              <input v-model="hook.url" type="url" required pattern="https?://.*" placeholder="https://example.com/webhook" class="input" />
            </div>
            <div>
              <label class="form-label">模板目标</label>
              <select class="input" @change="applyWebhookTemplate(hook, $event)">
                <option value="">选择渠道填充…</option>
                <option value="dingtalk">钉钉机器人</option>
                <option value="feishu">飞书机器人</option>
                <option value="wecom">企业微信机器人</option>
                <option value="slack">Slack</option>
              </select>
            </div>
            <div>
              <label class="form-label">请求体模板</label>
              <textarea v-model="hook.template" rows="3" class="input font-mono text-xs" placeholder="留空则发送标准 JSON"></textarea>
              <p class="mt-1 text-xs text-slate-400">{{ templateHint }}</p>
            </div>
          </div>
        </div>
      </div>
      <button type="button" class="btn-add-row" @click="addWebhook">
        <i class="fas fa-plus text-xs"></i>添加通道
      </button>
    </fieldset>

    <fieldset class="border-t border-slate-200 pt-6 space-y-4">
      <legend class="section-title w-full">资源告警规则</legend>
      <div v-if="config.draft.notify.rules.length === 0" class="empty-note">暂无规则，点击下方按钮添加</div>
      <div v-else class="space-y-4">
        <div v-for="(rule, index) in config.draft.notify.rules" :key="index" class="panel-frame">
          <div class="card-body space-y-4">
            <div class="flex items-center justify-between">
              <span class="text-sm font-semibold text-slate-700">规则 {{ index + 1 }}</span>
              <button type="button" class="btn-icon btn-icon-red" title="删除规则" @click="removeRule(index)">
                <i class="fas fa-trash-can text-xs"></i>
              </button>
            </div>
            <div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
              <div>
                <label class="form-label">指标</label>
                <select v-model="rule.metric" class="input">
                  <option value="cpu">CPU 使用率</option>
                  <option value="memory">内存使用率</option>
                  <option value="disk">磁盘使用率</option>
                </select>
              </div>
              <div>
                <label class="form-label">阈值(%)</label>
                <input v-model.number="rule.threshold" type="number" min="1" max="100" required class="input" />
              </div>
              <div>
                <label class="form-label">持续次数</label>
                <input v-model.number="rule.duration" type="number" min="1" required class="input" />
              </div>
            </div>
          </div>
        </div>
      </div>
      <button type="button" class="btn-add-row" @click="addRule">
        <i class="fas fa-plus text-xs"></i>添加规则
      </button>
      <p class="text-xs text-slate-400">持续次数指连续多少个采集周期超阈值才告警，用于抑制瞬时抖动。</p>
    </fieldset>
  </section>
</template>
