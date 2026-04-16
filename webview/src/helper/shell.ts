import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'

let term: Terminal | null = null
let socket: WebSocket | null = null
let fitAddon: FitAddon | null = null
let resizeHandler: (() => void) | null = null

export function create(el: HTMLElement, token: string, shell = 'bash'): void {
    if (!el) return
    destroy()

    term = new Terminal({ theme: { background: '#222' }, fontSize: 15, cursorBlink: true })
    fitAddon = new FitAddon()
    term.loadAddon(fitAddon)
    term.open(el)
    fitAddon.fit()

    resizeHandler = () => fitAddon?.fit()
    window.addEventListener('resize', resizeHandler)

    const protocol = location.protocol === 'https:' ? 'wss://' : 'ws://'
    socket = new WebSocket(`${protocol}${location.host}/ws/shell?token=${token}&shell=${encodeURIComponent(shell)}`)

    term.onData(data => socket?.readyState === WebSocket.OPEN && socket.send(data))
    socket.onopen = () => term && term.write('[连接中...]\r\n')
    socket.onmessage = e => term && term.write(e.data)
    socket.onclose = () => term && term.write('\r\n[连接已关闭]\r\n')
    socket.onerror = (e: Event) => term && term.write(`\r\n[连接错误: ${(e as ErrorEvent).message ?? ''}]\r\n`)

    term.focus()
}

export function destroy(): void {
    resizeHandler && window.removeEventListener('resize', resizeHandler)
    fitAddon = resizeHandler = null
    term?.dispose()
    socket?.close()
    term = socket = null
}
