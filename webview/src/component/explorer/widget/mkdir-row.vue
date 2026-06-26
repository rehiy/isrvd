<script lang="ts">
/**
 * 新建目录输入行组件（表格行）
 *
 * 用法（在 <tbody> 内）：
 *   <MkdirRow v-if="mkdirMode" :can-select="canSelect" @confirm="onConfirm" @cancel="onCancel" />
 *
 * confirm 事件携带目录名称字符串，父组件负责拼接路径和调用 adapter.mkdir。
 */
import { Component, Prop, Vue, toNative } from 'vue-facing-decorator'

@Component({
    emits: ['confirm', 'cancel'],
})
class MkdirRow extends Vue {
    /** 是否显示 checkbox 占位列（与文件列表对齐） */
    @Prop({ default: false }) canSelect!: boolean

    name = ''

    handleConfirm() {
        const trimmed = this.name.trim()
        if (!trimmed) return
        this.$emit('confirm', trimmed)
    }

    handleCancel() {
        this.$emit('cancel')
    }
}

export default toNative(MkdirRow)
</script>

<template>
  <tr class="border-b border-teal-100 bg-teal-50">
    <!-- checkbox 占位列 -->
    <td v-if="canSelect" class="px-4 py-3 w-12">
      <input type="checkbox" class="rounded border-slate-300 opacity-30" disabled>
    </td>
    <!-- 名称列：图标 + 输入框 -->
    <td class="px-4 py-3" colspan="4">
      <div class="inline-info">
        <div class="row-icon bg-amber-400 flex-shrink-0">
          <i class="fas fa-folder text-white text-sm"></i>
        </div>
        <input
          v-model="name"
          class="input text-sm py-1 flex-1"
          placeholder="请输入目录名称"
          autofocus
          @keyup.enter="handleConfirm()"
          @keyup.esc="handleCancel()"
        />
      </div>
    </td>
    <!-- 操作列 -->
    <td class="px-4 py-3 whitespace-nowrap">
      <div class="flex justify-end items-center gap-2">
        <button class="btn btn-primary" @click="handleConfirm()">
          <i class="fas fa-check"></i><span>确认</span>
        </button>
        <button class="btn btn-secondary" @click="handleCancel()">
          <i class="fas fa-xmark"></i><span>取消</span>
        </button>
      </div>
    </td>
  </tr>
</template>
