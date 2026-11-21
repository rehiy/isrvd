import { nextTick } from 'vue'

import { useLocaleStore } from '@/stores'

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
    private snapshot: AbortController | null = null

    get active() {
        return this.state !== 'snapshot'
    }

    async load(fetchLogs: (signal: AbortSignal) => Promise<string[]>) {
        this.stop()
        const request = new AbortController()
        this.snapshot = request
        this.loading = true
        this.content = ''
        try {
            const lines = await fetchLogs(request.signal)
            if (!request.signal.aborted) {
                this.content = lines.join('')
            }
        } catch {
            if (!request.signal.aborted) {
                this.content = useLocaleStore().t('加载日志失败')
            }
        } finally {
            if (!request.signal.aborted) {
                this.snapshot = null
                this.loading = false
            }
        }
    }

    start(path: string, token: string, onError: (message: string) => void) {
        if (this.active) {
            return
        }
        this.stop()
        this.content = ''
        this.state = 'connecting'
        // EventSource 同样无法自定义请求头，用 lang 查询参数携带界面语言
        const params = new URLSearchParams({ token, tail: this.tail, lang: useLocaleStore().locale })
        const source = new EventSource(absUrl(`${path}?${params.toString()}`))
        this.source = source
        source.onopen = () => {
            if (this.source === source) {
                this.state = 'streaming'
            }
        }
        source.onmessage = event => {
            if (this.source === source) {
                this.append(event.data)
            }
        }
        source.addEventListener('error', event => {
            if (this.source !== source) {
                return
            }
            const message = (event as MessageEvent).data ?? ''
            this.stop()
            onError(message || useLocaleStore().t('实时日志连接失败'))
        })
    }

    stop() {
        this.snapshot?.abort()
        this.snapshot = null
        this.source?.close()
        this.source = null
        this.loading = false
        this.state = 'snapshot'
    }

    reloadTail(loadSnapshot: () => void, startStream: () => void) {
        if (this.active) {
            this.stop()
            startStream()
        } else {
            loadSnapshot()
        }
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
