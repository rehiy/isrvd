// 终端页底部文件面板的高度、拖拽边界和事件生命周期。
export class SplitPane {
    height = 0
    dragging = false
    private startY = 0
    private startHeight = 0
    private container: HTMLElement | null = null
    private moveHandler: ((event: MouseEvent) => void) | null = null
    private endHandler: (() => void) | null = null

    initialize(containerHeight = 600) {
        this.height = Math.floor(containerHeight * 0.4)
    }

    start(event: MouseEvent, container?: HTMLElement) {
        this.stop()
        this.container = container ?? null
        this.dragging = true
        this.startY = event.clientY
        this.startHeight = this.height
        this.moveHandler = event => {
            if (!this.dragging) return
            const delta = this.startY - event.clientY
            this.height = Math.min(Math.max(this.startHeight + delta, 120), (this.container?.clientHeight ?? 600) - 120)
        }
        this.endHandler = () => this.stop()
        document.addEventListener('mousemove', this.moveHandler)
        document.addEventListener('mouseup', this.endHandler)
        document.body.classList.add('drag-resizing')
    }

    stop() {
        this.dragging = false
        if (this.moveHandler) document.removeEventListener('mousemove', this.moveHandler)
        if (this.endHandler) document.removeEventListener('mouseup', this.endHandler)
        this.moveHandler = null
        this.endHandler = null
        this.container = null
        document.body.classList.remove('drag-resizing')
    }
}
