import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'

/**
 * 终端适配器接口
 */
export interface TerminalAdapter {
    readonly connected: boolean
    connect(el: HTMLElement): void
    disconnect(): void
    fit(): void
}

/**
 * 基于 WebSocket 的通用终端实现，传入 wsUrl 即可。
 */
export class WsTerminal implements TerminalAdapter {
    private term: Terminal | null = null
    private socket: WebSocket | null = null
    private fitAddon: FitAddon | null = null

    get connected(): boolean {
        return this.socket !== null
    }

    constructor(private wsUrl: string) {}

    connect(el: HTMLElement): void {
        if (!el) return
        this.disconnect()

        const fitAddon = new FitAddon()
        const term = new Terminal({
            fontSize: 15,
            cursorBlink: true,
            theme: { background: '#0f172a' }
        })
        term.loadAddon(fitAddon)
        term.open(el)
        fitAddon.fit()

        this.fitAddon = fitAddon
        this.term = term

        const socket = new WebSocket(this.wsUrl)
        this.socket = socket

        // 所有回调都通过 this.term 访问，disconnect 后 this.term 为 null，回调自动失效
        term.onData(data => socket.readyState === WebSocket.OPEN && socket.send(data))
        socket.onopen = () => this.term?.write('[连接中...]\r\n')
        socket.onmessage = e => this.term?.write(e.data)
        socket.onclose = () => this.term?.write('\r\n[连接已关闭]\r\n')
        socket.onerror = (e: Event) => this.term?.write(`\r\n[连接错误: ${(e as ErrorEvent).message ?? ''}]\r\n`)

        term.focus()
    }

    disconnect(): void {
        if (!this.term && !this.socket) return  // 幂等保护
        this.socket?.close()
        this.socket = null
        // 先清空 this.term，socket 的 onclose 回调通过 this.term?.write 访问，此后回调自动失效
        const term = this.term
        this.term = null
        this.fitAddon = null
        term?.dispose()
    }

    fit(): void {
        if (this.term && this.fitAddon) this.fitAddon.fit()
    }
}
