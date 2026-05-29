<script lang="ts">
import { defineComponent, h, resolveComponent, type VNodeChild } from 'vue'
import { Component, Prop, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'

import { type UploadNode, type UploadFileNode, type UploadDirNode, flattenUploadTree } from '@/helper/ssh'

// ─── 递归节点渲染子组件 ───────────────────────────────────────────────────────

const UploadNodeItem = defineComponent({
    name: 'UploadNodeItem',
    props: {
        node: { type: Object as () => UploadNode, required: true },
        depth: { type: Number, default: 0 },
    },
    emits: ['cancel'],
    setup(props, { emit }): () => VNodeChild {
        return (): VNodeChild => {
            const node = props.node
            const depth = props.depth
            const pl = `${12 + depth * 16}px`
            const uploadNodeItem = resolveComponent('UploadNodeItem')

            if (node.type === 'dir') {
                const dir = node as UploadDirNode
                const files = flattenUploadTree(dir.children)
                const total = files.length
                const done = files.filter(f => f.done && !f.error && !f.cancelled).length
                const fail = files.filter(f => !!f.error).length
                const cancelled = files.filter(f => f.cancelled).length
                const allFinished = done + fail + cancelled >= total

                return [
                    h('div', {
                        class: 'flex items-center gap-2 py-1 text-xs cursor-pointer hover:bg-slate-100 select-none',
                        style: { paddingLeft: pl, paddingRight: '12px' },
                        onClick: () => { dir.expanded = !dir.expanded },
                    }, [
                        h('i', { class: `fas fa-chevron-right text-slate-400 w-3 flex-shrink-0 transition-transform${dir.expanded ? ' rotate-90' : ''}` }),
                        h('div', { class: 'w-4 h-4 rounded bg-amber-400 flex items-center justify-center flex-shrink-0' },
                            h('i', { class: 'fas fa-folder text-white text-[10px]' })
                        ),
                        h('span', { class: 'text-slate-700 font-medium flex-1 truncate' }, node.name),
                        h('span', { class: 'text-slate-400 flex-shrink-0 text-[11px]' }, [
                            `${done}/${total}`,
                            fail > 0 ? h('span', { class: 'text-red-400 ml-1' }, `${fail} 失败`) : null,
                            cancelled > 0 ? h('span', { class: 'text-slate-300 ml-1' }, `${cancelled} 已取消`) : null,
                        ]),
                        depth === 0 && !allFinished
                            ? h('button', {
                                class: 'ml-1 w-4 h-4 flex items-center justify-center rounded hover:bg-slate-200 flex-shrink-0',
                                title: '取消上传',
                                onClick: (e: Event) => { e.stopPropagation(); emit('cancel') },
                            }, h('i', { class: 'fas fa-xmark text-slate-400 text-[10px]' }))
                            : null,
                    ]),
                    dir.expanded
                        ? dir.children.map(child =>
                            h(uploadNodeItem, {
                                key: child.name,
                                node: child,
                                depth: depth + 1,
                            })
                        )
                        : null,
                ]
            } else {
                const file = node as UploadFileNode
                const iconClass = file.cancelled
                    ? 'fa-ban text-slate-300'
                    : file.error ? 'fa-circle-exclamation text-red-400'
                    : file.done ? 'fa-circle-check text-teal-500'
                    : 'fa-arrow-up-from-bracket text-slate-400'

                return h('div', {
                    class: 'flex items-center gap-2 py-1 text-xs',
                    style: { paddingLeft: pl, paddingRight: '12px' },
                }, [
                    h('i', { class: `fas w-3 flex-shrink-0 ${iconClass}` }),
                    h('span', {
                        class: `truncate flex-1 min-w-0 ${file.cancelled ? 'text-slate-300 line-through' : 'text-slate-600'}`,
                    }, node.name),
                    file.cancelled
                        ? h('span', { class: 'text-slate-300 flex-shrink-0 text-[11px]' }, '已取消')
                        : file.error
                        ? h('span', { class: 'text-red-400 flex-shrink-0 text-[11px]' }, '失败')
                        : [
                            h('div', { class: 'w-16 h-1.5 bg-slate-200 rounded-full flex-shrink-0 overflow-hidden' },
                                h('div', {
                                    class: `h-full rounded-full transition-all duration-200 ${file.done ? 'bg-teal-500' : 'bg-teal-400'}`,
                                    style: { width: `${file.percent}%` },
                                })
                            ),
                            h('span', { class: 'text-slate-400 w-7 text-right flex-shrink-0 text-[11px]' }, `${file.percent}%`),
                        ],
                    depth === 0 && !file.done && !file.cancelled
                        ? h('button', {
                            class: 'ml-1 w-4 h-4 flex items-center justify-center rounded hover:bg-slate-200 flex-shrink-0',
                            title: '取消上传',
                            onClick: () => emit('cancel'),
                        }, h('i', { class: 'fas fa-xmark text-slate-400 text-[10px]' }))
                        : null,
                ])
            }
        }
    },
})

// ─── 上传 Widget 主组件 ───────────────────────────────────────────────────────

@Component({
    components: { UploadNodeItem },
    emits: ['done'],
})
class UploadWidget extends Vue {
    @Prop({ required: true }) hostId!: string
    @Prop({ required: true }) sftpPath!: string
    @Prop({ default: 3 }) concurrency!: number

    portal = usePortal()

    uploadNodes: UploadNode[] = []
    uploadVersion = 0
    private uploadQueue: Promise<void> = Promise.resolve()
    private pendingBatches = 0
    private clearTimer: ReturnType<typeof setTimeout> | null = null

    get hasNodes() { return this.uploadNodes.length > 0 }

    private collectDirs(nodes: UploadNode[]): string[] {
        const dirs = new Set<string>()
        const walk = (items: UploadNode[]) => {
            for (const item of items) {
                if (item.type === 'dir') {
                    dirs.add(item.destDir)
                    walk(item.children)
                }
            }
        }
        walk(nodes)
        return [...dirs]
    }

    private scheduleClear(delay: number) {
        if (this.clearTimer) clearTimeout(this.clearTimer)
        this.clearTimer = setTimeout(() => {
            this.clearTimer = null
            if (this.pendingBatches !== 0) return
            const allDone = flattenUploadTree(this.uploadNodes).every(f => f.done || f.cancelled)
            if (!allDone) return
            this.uploadNodes = []
            this.uploadVersion++
        }, delay)
    }

    // 供父组件通过 ref 调用
    upload(nodes: UploadNode[]) {
        if (nodes.length === 0) return
        if (this.clearTimer) {
            clearTimeout(this.clearTimer)
            this.clearTimer = null
        }
        // 过滤已在队列中的同名顶层节点
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

    async doUpload(nodes: UploadNode[]) {
        const files = flattenUploadTree(nodes)
        let failCount = 0
        let cancelCount = 0
        // 收集本批次需要创建的目录（去重），包含空目录和文件所在目录
        const dirsToCreate = new Set([...this.collectDirs(nodes), ...files.map(f => f.destDir)])
        for (const dir of dirsToCreate) {
            try { await api.sftpMkdir(this.hostId, { path: dir }) } catch { /* 已存在则忽略 */ }
        }

        // 并发池上传
        const queue = files.slice()
        const concurrency = Math.max(1, this.concurrency)

        const uploadOne = async (fileNode: UploadFileNode) => {
            if (fileNode.cancelled) { cancelCount++; return }
            const form = new FormData()
            form.append('file', fileNode.file)
            try {
                await api.sftpUpload(this.hostId, fileNode.destDir, form, (percent) => {
                    fileNode.percent = percent
                    this.uploadVersion++
                })
                fileNode.percent = 100
                fileNode.done = true
            } catch (err: unknown) {
                fileNode.error = (err instanceof Error ? err.message : '') || '上传失败'
                fileNode.done = true
                failCount++
            }
            this.uploadVersion++
        }

        const worker = async () => {
            while (queue.length > 0) {
                const fileNode = queue.shift()
                if (!fileNode) return
                await uploadOne(fileNode)
            }
        }

        await Promise.all(Array.from({ length: concurrency }, worker))

        const total = files.length - cancelCount
        if (total > 0) {
            if (failCount === 0) {
                this.portal.showNotification('success', `上传成功（${total} 个文件）`)
            } else {
                this.portal.showNotification('error', `${total - failCount} 个成功，${failCount} 个失败`)
            }
        }
        this.pendingBatches--
        // 只有所有批次都完成后才清空列表
        if (this.pendingBatches === 0) {
            const allDone = flattenUploadTree(this.uploadNodes).every(f => f.done || f.cancelled)
            if (allDone) {
                this.scheduleClear(1500)
                this.$emit('done')
            }
        }
    }

    cancelNode(node: UploadNode) {
        const files = node.type === 'file' ? [node] : flattenUploadTree((node as UploadDirNode).children)
        for (const f of files) {
            if (!f.done) f.cancelled = true
        }
        this.uploadNodes = [...this.uploadNodes]
        this.uploadVersion++
        const allDone = flattenUploadTree(this.uploadNodes).every(f => f.done || f.cancelled)
        if (allDone) {
            this.scheduleClear(800)
            this.$emit('done')
        }
    }
}

export default toNative(UploadWidget)
</script>

<template>
  <div
    v-if="hasNodes"
    class="flex-shrink border-b border-slate-200 bg-slate-50 overflow-y-auto"
    style="max-height: min(160px, calc(100% - 164px))"
  >
    <UploadNodeItem
      v-for="uploadNode in uploadNodes"
      :key="uploadNode.name + '-' + uploadVersion"
      :node="uploadNode"
      :depth="0"
      @cancel="cancelNode(uploadNode)"
    />
  </div>
</template>
