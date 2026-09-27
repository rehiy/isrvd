import { nextTick } from 'vue'

import { absUrl } from '@/service/client'

const MAX_LOG_LENGTH = 300000

type StreamState = 'snapshot' | 'connecting' | 'streaming'

// Docker 和 Swarm 日志页共用快照加载、SSE 状态与有界日志缓冲。
export class LogStream {
    loading = false
    content = ''
    tail = '100'
    state: StreamState = 'snapshot'
    private source: EventSource | null = null

    get active() {
        return this.state !== 'snapshot'
    }

    async load(fetchLogs: () => Promise<string[]>) {
        this.stop()
        this.loading = true
        this.content = ''
        try {
            this.content = (await fetchLogs()).join('')
        } catch {
            this.content = '加载日志失败'
        }
        this.loading = false
    }

    start(path: string, token: string, onError: (message: string) => void) {
        if (this.active) return
        this.loading = false
        this.content = ''
        this.state = 'connecting'
        this.source?.close()
        const params = new URLSearchParams({ token, tail: this.tail })
        this.source = new EventSource(absUrl(`${path}?${params.toString()}`))
        this.source.onopen = () => {
            this.state = 'streaming'
        }
        this.source.onmessage = event => this.append(event.data)
        this.source.addEventListener('error', event => {
            const message = (event as MessageEvent).data ?? ''
            this.stop()
            onError(message || '实时日志连接失败')
        })
        this.source.onerror = () => {
            if (this.source?.readyState === EventSource.CLOSED) this.stop()
        }
    }

    stop() {
        this.source?.close()
        this.source = null
        this.state = 'snapshot'
    }

    private append(data: string) {
        this.content += data + '\n'
        if (this.content.length > MAX_LOG_LENGTH) {
            this.content = this.content.slice(-MAX_LOG_LENGTH)
        }
        // 用户位于底部 100px 内时才追尾，保留手动翻阅历史的位置。
        const nearBottom = document.body.scrollHeight - window.scrollY - window.innerHeight < 100
        if (nearBottom) {
            nextTick(() => requestAnimationFrame(() => {
                window.scrollTo({ top: document.body.scrollHeight })
            }))
        }
    }
}
