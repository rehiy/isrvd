<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { useConfigStore } from '@/stores'

import IconSelect from '@/component/icon-select.vue'

@Component({ components: { IconSelect } })
class ConfigExtensions extends Vue {
    config = useConfigStore()

    addLink() {
        this.config.draft.links.push({ label: '', url: '', icon: 'fas fa-link' })
    }

    removeLink(index: number) {
        this.config.draft.links.splice(index, 1)
    }
}

export default toNative(ConfigExtensions)
</script>

<template>
  <div class="max-w-4xl space-y-6">
    <!-- 监控采集 -->
    <section class="space-y-4">
      <div class="flex items-center gap-2">
        <span class="card-icon bg-indigo-100 text-indigo-600"><i class="fas fa-chart-line"></i></span>
        <div>
          <h2 class="text-sm font-semibold text-slate-700">监控日志</h2>
          <p class="text-xs text-slate-400 mt-0.5">系统与容器监控采集</p>
        </div>
      </div>
      <div>
        <label class="form-label">监控采集间隔</label>
        <select v-model.number="config.draft.monitor.interval" class="input">
          <option :value="0">禁用</option>
          <option :value="5">5 秒</option>
          <option :value="15">15 秒</option>
          <option :value="30">30 秒</option>
          <option :value="60">60 秒</option>
        </select>
        <p class="mt-1 text-xs text-slate-400">系统与容器监控数据的采集频率，禁用后不再写入监控文件</p>
      </div>
    </section>

    <!-- 应用市场 -->
    <section class="border-t border-slate-200 pt-6 space-y-4">
      <div class="flex items-center gap-2">
        <span class="card-icon bg-indigo-100 text-indigo-600"><i class="fas fa-store"></i></span>
        <div>
          <h2 class="text-sm font-semibold text-slate-700">应用市场</h2>
          <p class="text-xs text-slate-400 mt-0.5">市场 iframe 站点地址</p>
        </div>
      </div>
      <div>
        <label class="form-label">站点 URL</label>
        <input v-model="config.draft.marketplace.url" type="text" placeholder="请输入应用市场 URL" class="input" />
        <p class="mt-1 text-xs text-slate-400">应用市场页面以 iframe 方式嵌入，并通过 postMessage 协议接收安装事件</p>
      </div>
    </section>

    <!-- 导航链接 -->
    <section class="border-t border-slate-200 pt-6 space-y-4">
      <div class="flex items-center gap-2">
        <span class="card-icon bg-indigo-100 text-indigo-600"><i class="fas fa-link"></i></span>
        <div>
          <h2 class="text-sm font-semibold text-slate-700">导航链接</h2>
          <p class="text-xs text-slate-400 mt-0.5">顶部工具栏外部链接</p>
        </div>
      </div>
      <div v-if="config.draft.links.length === 0" class="empty-note">暂无链接，点击下方按钮添加</div>
      <div v-else class="space-y-4">
        <div v-for="(link, index) in config.draft.links" :key="index" class="panel-frame">
          <div class="card-body space-y-4">
            <div class="flex items-center justify-between">
              <span class="text-sm font-semibold text-slate-700">链接 {{ index + 1 }}</span>
              <button type="button" class="btn-icon btn-icon-red" title="删除链接" @click="removeLink(index)">
                <i class="fas fa-trash-can text-xs"></i>
              </button>
            </div>
            <div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
              <div>
                <label class="form-label">名称</label>
                <input v-model="link.label" type="text" required placeholder="如：工具箱" class="input" />
              </div>
              <div>
                <label class="form-label">链接地址</label>
                <input v-model="link.url" type="url" required pattern="https?://.*" placeholder="https://example.com" class="input" />
              </div>
              <div>
                <label class="form-label">图标</label>
                <IconSelect v-model="link.icon" />
              </div>
            </div>
          </div>
        </div>
      </div>
      <button type="button" class="btn-add-row" @click="addLink">
        <i class="fas fa-plus text-xs"></i>添加链接
      </button>
    </section>
  </div>
</template>
