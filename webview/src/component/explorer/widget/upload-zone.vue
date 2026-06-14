<script lang="ts">
/**
 * 拖拽上传逻辑组件（无渲染）
 *
 * 不渲染任何 DOM，通过 expose 暴露拖拽事件绑定和状态，
 * 由调用方决定将 dragAttrs 绑定到哪个元素上。
 *
 * 用法：
 *   <UploadZone ref="uploadZoneRef" :enabled="can.upload" :upload-ref="uploadRef" />
 *
 *   // 在需要拖拽的元素上：
 *   <div v-bind="uploadZoneRef?.dragAttrs" class="relative">
 *     <div v-if="uploadZoneRef?.dragOver" class="absolute inset-0 ...遮罩...</div>
 *     <!-- 内容 -->
 *   </div>
 */
import { Component, Prop, Vue, toNative } from 'vue-facing-decorator'

import type Upload from './upload.vue'

@Component({ expose: ['dragAttrs', 'dragOver'] })
class UploadZone extends Vue {
    /** 是否启用拖拽（对应 can.upload） */
    @Prop({ default: true }) enabled!: boolean

    /** upload 组件 ref，落下时自动调用 startDrop */
    @Prop({ default: null }) uploadRef!: InstanceType<typeof Upload> | null

    dragOver = false
    private dragCounter = 0

    /** 绑定到目标容器元素的事件对象 */
    get dragAttrs() {
        return {
            onDragenter: this.onDragEnter,
            onDragover: this.onDragOver,
            onDragleave: this.onDragLeave,
            onDrop: this.onDrop,
        }
    }

    onDragEnter(e: DragEvent) {
        if (!this.enabled) return
        e.preventDefault()
        this.dragCounter++
        this.dragOver = true
    }

    onDragOver(e: DragEvent) {
        if (!this.enabled) return
        e.preventDefault()
    }

    onDragLeave(e: DragEvent) {
        if (!this.enabled) return
        e.preventDefault()
        this.dragCounter--
        if (this.dragCounter <= 0) { this.dragCounter = 0; this.dragOver = false }
    }

    async onDrop(e: DragEvent) {
        if (!this.enabled) return
        e.preventDefault()
        this.dragOver = false
        this.dragCounter = 0

        const items = e.dataTransfer?.items
        if (!items || items.length === 0) return

        const entries: FileSystemEntry[] = []
        for (let i = 0; i < items.length; i++) {
            const entry = items[i].webkitGetAsEntry()
            if (entry) entries.push(entry)
        }
        if (entries.length === 0) return

        if (this.uploadRef) await this.uploadRef.startDrop(entries)
    }
}

export default toNative(UploadZone)
</script>

<template>
  <!-- 无渲染组件，不输出任何 DOM -->
</template>
