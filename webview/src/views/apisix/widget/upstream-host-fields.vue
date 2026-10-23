<script lang="ts">
import { Component, Prop, Ref, Vue, toNative } from 'vue-facing-decorator'

import type { ApisixUpstreamPassHost } from '@/service/types'

@Component({
    expose: ['focusHost'],
    emits: ['update:passHost', 'update:upstreamHost']
})
class UpstreamHostFields extends Vue {
    @Prop({ required: true }) passHost!: ApisixUpstreamPassHost
    @Prop({ required: true }) upstreamHost!: string
    @Ref readonly hostRef!: HTMLInputElement

    get mode() { return this.passHost }
    set mode(value: ApisixUpstreamPassHost) { this.$emit('update:passHost', value) }

    get host() { return this.upstreamHost }
    set host(value: string) { this.$emit('update:upstreamHost', value) }

    focusHost() {
        this.hostRef?.focus()
    }
}

export default toNative(UpstreamHostFields)
</script>

<template>
  <div class="space-y-4">
    <div>
      <label class="form-label">{{ $t('Host 传递方式') }}</label>
      <select v-model="mode" class="input">
        <option value="pass">{{ $t('透传客户端 Host（默认）') }}</option>
        <option value="node">{{ $t('使用节点 Host') }}</option>
        <option value="rewrite">{{ $t('指定上游 Host') }}</option>
      </select>
      <p class="text-xs text-slate-400 mt-1">{{ $t('控制转发到后端的 Host 请求头，与路由的 Host 匹配条件无关。') }}</p>
    </div>
    <div v-if="mode === 'rewrite'">
      <label class="form-label">{{ $t('上游 Host') }} <span class="text-red-500">*</span></label>
      <input ref="hostRef" v-model="host" type="text" class="input" :placeholder="$t('例如：backend.example.com')" />
    </div>
  </div>
</template>
