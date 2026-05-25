interface LogStreamCallbacks {
    onOpen?: () => void
    onMessage?: (data: string) => void
    onError?: (msg: string) => void
    onClose?: () => void
}

let source: EventSource | null = null

const streamUrl = (path: string) => new URL(`api/${path.replace(/^\/+/, '')}`, window.location.href).toString()

export function create(token: string, containerId: string, tail: string, callbacks: LogStreamCallbacks): void {
    destroy()

    const params = new URLSearchParams({ token, tail })
    source = new EventSource(streamUrl(`docker/container/${encodeURIComponent(containerId)}/logs/stream?${params.toString()}`))
    source.onopen = () => callbacks.onOpen?.()
    source.onmessage = event => callbacks.onMessage?.(event.data)
    source.addEventListener('error', event => {
        const msg = (event as MessageEvent).data ?? ''
        callbacks.onError?.(msg)
    })
    source.onerror = () => {
        // readyState 2 = CLOSED：服务端正常关闭（容器停止后 EOF）
        if (source?.readyState === EventSource.CLOSED) {
            destroy()
            callbacks.onClose?.()
        }
    }
}

export function destroy(): void {
    source?.close()
    source = null
}
