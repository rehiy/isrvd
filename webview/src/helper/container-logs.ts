import { wsUrl } from '@/service/axios'

interface LogStreamCallbacks {
    onOpen?: () => void
    onMessage?: (data: string) => void
    onClose?: () => void
    onError?: () => void
}

let socket: WebSocket | null = null

function readMessage(data: unknown, callback?: (data: string) => void) {
    if (!callback) return
    if (typeof data === 'string') {
        callback(data)
        return
    }
    if (data instanceof Blob) {
        data.text().then(callback)
    }
}

export function create(token: string, containerId: string, tail: string, callbacks: LogStreamCallbacks): void {
    destroy()

    const params = new URLSearchParams({
        token,
        tail
    })
    socket = new WebSocket(wsUrl(`docker/container/${encodeURIComponent(containerId)}/logs/stream?${params.toString()}`))

    socket.onopen = () => callbacks.onOpen?.()
    socket.onmessage = event => readMessage(event.data, callbacks.onMessage)
    socket.onclose = () => callbacks.onClose?.()
    socket.onerror = () => callbacks.onError?.()
}

export function destroy(): void {
    if (socket) {
        socket.onopen = null
        socket.onmessage = null
        socket.onclose = null
        socket.onerror = null
        socket.close()
    }
    socket = null
}
