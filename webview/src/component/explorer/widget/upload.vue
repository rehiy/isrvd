<script lang="ts">
/**
 * 文件上传进度组件（嵌入式）
 *
 * 嵌入在文件列表上方，有任务时显示，全部完成后自动收起。
 *
 * 触发方式：
 *   uploadRef.triggerFileSelect()  —— 打开多选文件框（点击上传按钮调用）
 *   uploadRef.startDrop(entries)   —— 传入拖拽 FileSystemEntry[]（拖拽落下调用）
 */
import axios from 'axios'
import { defineComponent, h, resolveComponent, type VNodeChild } from 'vue'
import { Component, Prop, Ref, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import { buildUploadTree, type UploadDirNode, type UploadFileNode, type UploadNode, flattenUploadTree } from './upload-ssh'

import type { ExplorerAdapter } from '../types'

// ─── 递归节点渲染子组件 ─────────────────────────────────────────────────────────

const UploadNodeItem = defineComponent({
    name: 'UploadNodeItem',
    props: {
        node: { type: Object as () => UploadNode, required: true },
        depth: { type: Number, default: 0 },
    },
    emits: ['cancel', 'retry', 'clearCancelled'],
    setup(props, { emit }): () => VNodeChild {
        return (): VNodeChild => {
            const node = props.node
            const depth = props.depth
            const pl = `${12 + depth * 16}px`
            const uploadNodeItem = resolveComponent('UploadNodeItem')

            // 操作按钮：与 btn-icon 尺寸对齐（28px）
            const actionBtn = (title: string, icon: string, onClick: (e: Event) => void) =>
                h('button', {
                    class: 'w-7 h-7 flex items-center justify-center rounded-lg hover:bg-slate-200 flex-shrink-0 transition-colors',
                    title,
                    onClick,
                }, h('i', { class: `fas ${icon} text-slate-400 text-xs` }))

            if (node.type === 'dir') {
                const dir = node as UploadDirNode
                const files = flattenUploadTree(dir.children)
                const total = files.length
                const done = files.filter(f => f.done && !f.error && !f.cancelled).length
                const fail = files.filter(f => !!f.error).length
                const cancelled = files.filter(f => f.cancelled).length
                const allFinished = done + fail + cancelled >= total
                const hasFailed = fail > 0
                const hasCancelled = cancelled > 0

                return [
                    h('div', {
                        class: 'flex items-center gap-2 px-4 py-2 cursor-pointer hover:bg-slate-100 select-none transition-colors',
                        style: { paddingLeft: pl },
                        onClick: () => { dir.expanded = !dir.expanded },
                    }, [
                        h('i', { class: `fas fa-chevron-right text-slate-400 w-3 flex-shrink-0 transition-transform${dir.expanded ? ' rotate-90' : ''}` }),
                        // 与文件列表 row-icon 保持一致：32×32 圆角色块
                        h('div', { class: 'row-icon bg-amber-400' },
                            h('i', { class: 'fas fa-folder text-white text-sm' })
                        ),
                        h('span', { class: 'text-sm text-slate-700 font-medium flex-1 truncate' }, node.name),
                        h('span', { class: 'text-xs text-slate-400 flex-shrink-0' }, [
                            `${done}/${total}`,
                            fail > 0 ? h('span', { class: 'text-red-400 ml-1.5' }, `${fail} 失败`) : null,
                            cancelled > 0 ? h('span', { class: 'text-slate-300 ml-1.5' }, `${cancelled} 已取消`) : null,
                        ]),
                        depth === 0 && !allFinished
                            ? actionBtn('取消上传', 'fa-xmark', (e) => { e.stopPropagation(); emit('cancel') })
                            : null,
                        depth === 0 && hasFailed
                            ? actionBtn('重试失败项', 'fa-rotate-right', (e) => { e.stopPropagation(); emit('retry') })
                            : null,
                        depth === 0 && hasCancelled
                            ? actionBtn('清理已取消', 'fa-broom', (e) => { e.stopPropagation(); emit('clearCancelled') })
                            : null,
                    ]),
                    dir.expanded
                        ? dir.children.map(child =>
                            h(uploadNodeItem, {
                                key: child.name,
                                node: child,
                                depth: depth + 1,
                                onCancel: () => emit('cancel'),
                                onRetry: () => emit('retry'),
                                onClearCancelled: () => emit('clearCancelled'),
                            })
                        )
                        : null,
                ]
            } else {
                const file = node as UploadFileNode
                // 图标色块与 row-icon 对齐
                const iconBg = file.cancelled ? 'bg-slate-200'
                    : file.error ? 'bg-red-400'
                    : file.done ? 'bg-teal-500'
                    : 'bg-blue-400'
                const iconName = file.cancelled ? 'fa-ban'
                    : file.error ? 'fa-circle-exclamation'
                    : file.done ? 'fa-circle-check'
                    : 'fa-arrow-up-from-bracket'

                return h('div', {
                    class: 'flex items-center gap-2 px-4 py-2 transition-colors',
                    style: { paddingLeft: pl },
                }, [
                    h('div', { class: `row-icon ${iconBg}` },
                        h('i', { class: `fas ${iconName} text-white text-sm` })
                    ),
                    h('div', { class: 'flex-1 min-w-0' }, [
                        h('p', {
                            class: `text-sm font-medium truncate ${file.cancelled ? 'text-slate-300 line-through' : 'text-slate-700'}`,
                        }, node.name),
                        // 进度条（上传中）或状态文字（完成/失败/取消）
                        !file.cancelled && !file.error
                            ? h('div', { class: 'flex items-center gap-2 mt-1' }, [
                                h('div', { class: 'flex-1 h-1.5 bg-slate-200 rounded-full overflow-hidden' },
                                    h('div', {
                                        class: `h-full rounded-full transition-all duration-300 ${file.done ? 'bg-teal-500' : 'bg-teal-400'}`,
                                        style: { width: `${file.percent}%` },
                                    })
                                ),
                                h('span', { class: 'text-xs text-slate-400 w-8 text-right flex-shrink-0' }, `${file.percent}%`),
                            ])
                            : h('p', {
                                class: `text-xs mt-0.5 ${file.error ? 'text-red-400' : 'text-slate-300'}`,
                            }, file.error ? file.error : '已取消'),
                    ]),
                    depth === 0 && !file.done && !file.cancelled
                        ? actionBtn('取消上传', 'fa-xmark', () => emit('cancel'))
                        : null,
                    depth === 0 && !!file.error
                        ? actionBtn('重试', 'fa-rotate-right', () => emit('retry'))
                        : null,
                    depth === 0 && file.cancelled
                        ? actionBtn('清理', 'fa-broom', () => emit('clearCancelled'))
                        : null,
                ])
            }
        }
    },
})

// ─── 上传主组件 ────────────────────────────────────────────────────────────────

@Component({
    expose: ['triggerFileSelect', 'startDrop'],
    components: { UploadNodeItem },
    emits: ['done', 'refresh'],
})
class Upload extends Vue {
    @Prop({ required: true }) adapter!: ExplorerAdapter
    @Prop({ required: true }) currentPath!: string
    @Prop({ default: 3 }) concurrency!: number

    @Ref readonly fileInput!: HTMLInputElement

    portal = usePortal()

    uploadNodes: UploadNode[] = []
    uploadVersion = 0
    private uploadQueue: Promise<void> = Promise.resolve()
    private pendingBatches = 0
    private clearTimer: ReturnType<typeof setTimeout> | null = null

    get hasNodes() { return this.uploadNodes.length > 0 }

    // ─── 公开方法 ─────────────────────────────────────────────────────────────

    /** 点击上传按钮时调用：打开文件多选框 */
    triggerFileSelect() {
        this.fileInput?.click()
    }

    /** 拖拽落下时调用：传入 FileSystemEntry[] 直接上传 */
    async startDrop(entries: FileSystemEntry[]) {
        if (entries.length === 0) return
        const nodes = await buildUploadTree(entries, this.currentPath)
        if (nodes.length === 0) return
        this.startUpload(nodes)
    }

    // ─── 文件选择处理 ─────────────────────────────────────────────────────────

    onFileInputChange(e: Event) {
        const files = (e.target as HTMLInputElement).files
        if (!files || files.length === 0) return
        const fileList = Array.from(files)
        ;(e.target as HTMLInputElement).value = ''
        const nodes: UploadNode[] = fileList.map(f => ({
            type: 'file' as const,
            name: f.name,
            size: f.size,
            destDir: this.currentPath,
            file: f,
            percent: 0,
            done: false,
            error: '',
            cancelled: false,
        }))
        this.startUpload(nodes)
    }

    // ─── 内部上传入口 ─────────────────────────────────────────────────────────

    private startUpload(nodes: UploadNode[]) {
        if (nodes.length === 0) return
        if (this.clearTimer) { clearTimeout(this.clearTimer); this.clearTimer = null }
        const existingNames = new Set(this.uploadNodes.map(n => n.name))
        const newNodes = nodes.filter(n => {
            if (existingNames.has(n.name)) {
                this.portal.showNotification('error', `「${n.name}」已在上传队列中`)
                return false
            }
            return true
        })
        if (newNodes.length === 0) return
        this.uploadNodes = [...this.uploadNodes, ...newNodes]
        this.pendingBatches++
        this.uploadQueue = this.uploadQueue.then(() => this.doUpload(newNodes))
    }

    // ─── 上传核心逻辑 ─────────────────────────────────────────────────────────

    private getUploadState() {
        const files = flattenUploadTree(this.uploadNodes)
        const allDone = files.every(f => f.done || f.cancelled)
        const hasFailed = files.some(f => !!f.error)
        return { files, allDone, hasFailed }
    }

    private collectDirs(nodes: UploadNode[]): string[] {
        const dirs = new Set<string>()
        const walk = (items: UploadNode[]) => {
            for (const item of items) {
                if (item.type === 'dir') {
                    if (item.cancelled) continue
                    dirs.add(item.destDir)
                    walk(item.children)
                }
            }
        }
        walk(nodes)
        return [...dirs]
    }

    private isNodeCancelled(node: UploadNode): boolean {
        if (node.type === 'file') return node.cancelled
        return node.cancelled || flattenUploadTree(node.children).every(f => f.cancelled)
    }

    private isBatchCancelled(files: UploadFileNode[], nodesForDirs?: UploadNode[]): boolean {
        if (nodesForDirs && nodesForDirs.length > 0 && nodesForDirs.every(n => this.isNodeCancelled(n))) return true
        return files.length > 0 && files.every(f => f.cancelled)
    }

    private markNodeCancelled(node: UploadNode) {
        if (node.type === 'file') {
            if (!node.done) { node.cancelled = true; node.controller?.abort() }
            return
        }
        node.cancelled = true
        for (const child of node.children) this.markNodeCancelled(child)
    }

    private scheduleClear(delay: number) {
        if (this.clearTimer) clearTimeout(this.clearTimer)
        this.clearTimer = setTimeout(() => {
            this.clearTimer = null
            if (this.pendingBatches !== 0) return
            const { allDone } = this.getUploadState()
            if (!allDone) return
            this.uploadNodes = []
            this.uploadVersion++
        }, delay)
    }

    private finishBatch() {
        this.pendingBatches = Math.max(0, this.pendingBatches - 1)
        if (this.pendingBatches !== 0) return
        const { allDone, hasFailed } = this.getUploadState()
        if (!allDone) return
        this.$emit('done')
        // 成功：1.5s 后关闭；有失败：5s 后关闭（让用户看清错误）
        this.scheduleClear(hasFailed ? 5000 : 1500)
    }

    private async cleanupPartialFile(fileNode: UploadFileNode): Promise<boolean> {
        try {
            await this.adapter.remove(`${fileNode.destDir.replace(/\/+$/, '')}/${fileNode.name}`, false)
            return true
        } catch { return false }
    }

    private retryFiles(files: UploadFileNode[]) {
        const failed = files.filter(f => !!f.error)
        if (failed.length === 0) return
        if (this.clearTimer) { clearTimeout(this.clearTimer); this.clearTimer = null }
        for (const f of failed) { f.percent = 0; f.done = false; f.error = ''; f.cancelled = false; f.controller = undefined }
        this.pendingBatches++
        this.uploadVersion++
        this.uploadQueue = this.uploadQueue.then(() => this.doUploadFiles(failed))
    }

    private pruneCancelledFromNode(node: UploadNode): UploadNode[] {
        if (node.type === 'file') return node.cancelled ? [] : [node]
        node.children = node.children.flatMap(child => this.pruneCancelledFromNode(child))
        return node.children.length > 0 ? [node] : []
    }

    private cleanupAfterPrune() {
        if (this.uploadNodes.length === 0) {
            if (this.clearTimer) { clearTimeout(this.clearTimer); this.clearTimer = null }
            return
        }
        const { allDone, hasFailed } = this.getUploadState()
        if (allDone) this.scheduleClear(hasFailed ? 5000 : 800)
    }

    private async doUpload(nodes: UploadNode[]) {
        await this.doUploadFiles(flattenUploadTree(nodes), nodes)
    }

    private async doUploadFiles(files: UploadFileNode[], nodesForDirs?: UploadNode[]) {
        let failCount = 0, cancelCount = 0, cleanupFailCount = 0

        const dirsToCreate = new Set([
            ...(nodesForDirs ? this.collectDirs(nodesForDirs) : []),
            ...files.map(f => f.destDir),
        ].filter(d => d !== '/'))
        for (const dir of dirsToCreate) {
            if (this.isBatchCancelled(files, nodesForDirs)) break
            try { await this.adapter.mkdir(dir) } catch { /* 已存在则忽略 */ }
        }

        if (this.isBatchCancelled(files, nodesForDirs)) { this.finishBatch(); return }

        const queue = files.slice()
        const concurrency = Math.max(1, this.concurrency)

        const uploadOne = async (fileNode: UploadFileNode) => {
            if (fileNode.cancelled) { cancelCount++; return }
            const controller = new AbortController()
            fileNode.controller = controller
            try {
                await this.adapter.upload(fileNode.destDir, fileNode.file, fileNode.name, (percent) => {
                    fileNode.percent = percent
                    this.uploadVersion++
                }, controller.signal)
                fileNode.percent = 100
                fileNode.done = true
                this.$emit('refresh')
            } catch (err: unknown) {
                if (axios.isCancel(err) || controller.signal.aborted || fileNode.cancelled) {
                    fileNode.cancelled = true; cancelCount++
                } else {
                    const ok = await this.cleanupPartialFile(fileNode)
                    if (!ok) cleanupFailCount++
                    fileNode.error = ok
                        ? ((err instanceof Error ? err.message : '') || '上传失败，已尝试清理残留文件')
                        : ((err instanceof Error ? err.message : '') || '上传失败，可能残留不完整文件')
                    fileNode.done = true; failCount++
                }
            } finally { fileNode.controller = undefined }
            this.uploadVersion++
        }

        const worker = async () => {
            while (queue.length > 0) {
                if (this.isBatchCancelled(files, nodesForDirs)) return
                const f = queue.shift()
                if (f) await uploadOne(f)
            }
        }
        await Promise.all(Array.from({ length: concurrency }, worker))

        const total = files.length - cancelCount
        if (total > 0) {
            failCount === 0
                ? this.portal.showNotification('success', `上传成功（${total} 个文件）`)
                : this.portal.showNotification('error', `${total - failCount} 个成功，${failCount} 个失败${cleanupFailCount > 0 ? '，部分残留文件清理失败' : ''}`)
        }
        this.finishBatch()
    }

    cancelNode(node: UploadNode) {
        this.markNodeCancelled(node)
        this.uploadNodes = [...this.uploadNodes]
        this.uploadVersion++
        if (this.pendingBatches === 0) {
            const { allDone, hasFailed } = this.getUploadState()
            if (allDone) { this.$emit('done'); this.scheduleClear(hasFailed ? 5000 : 800) }
        }
    }

    retryNode(node: UploadNode) {
        const files = node.type === 'file' ? [node as UploadFileNode] : flattenUploadTree((node as UploadDirNode).children)
        this.retryFiles(files)
    }

    clearCancelledNode(node: UploadNode) {
        this.uploadNodes = this.uploadNodes.flatMap(item => item === node ? this.pruneCancelledFromNode(item) : [item])
        this.uploadVersion++
        this.cleanupAfterPrune()
    }
}

export default toNative(Upload)
</script>

<template>
  <!-- 隐藏的文件选择 input -->
  <input ref="fileInput" type="file" multiple class="hidden" @change="onFileInputChange" />

  <!-- 上传进度列表（有任务时显示，嵌入在文件列表上方） -->
  <div
    v-if="hasNodes"
    class="flex-shrink-0 border-b border-slate-200 bg-slate-50"
  >
    <!-- 标题栏 -->
    <div class="flex items-center gap-2 px-4 py-2 border-b border-slate-100">
      <div class="w-5 h-5 rounded bg-primary-500 flex items-center justify-center flex-shrink-0">
        <i class="fas fa-arrow-up-from-bracket text-white text-[10px]"></i>
      </div>
      <span class="text-xs font-medium text-slate-600 flex-1">上传队列</span>
      <span class="text-xs text-slate-400">{{ uploadNodes.length }} 项</span>
    </div>
    <!-- 列表（最大高度 240px，超出滚动） -->
    <div class="max-h-60 overflow-y-auto divide-y divide-slate-100">
      <UploadNodeItem
        v-for="uploadNode in uploadNodes"
        :key="uploadNode.name + '-' + uploadVersion"
        :node="uploadNode"
        :depth="0"
        @cancel="cancelNode(uploadNode)"
        @retry="retryNode(uploadNode)"
        @clear-cancelled="clearCancelledNode(uploadNode)"
      />
    </div>
  </div>
</template>
