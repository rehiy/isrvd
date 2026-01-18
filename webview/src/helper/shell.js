// ==================== Shell 终端管理器 ====================

import { Terminal } from '@xterm/xterm'

let termInstance = null
let socketInstance = null

export function create(mountEl, token, shellType = 'bash') {
    if (!mountEl) return

    // 清理已存在的实例
    destroy()

    // 创建新的终端实例
    termInstance = new Terminal({
        theme: { background: '#222' },
        fontSize: 15,
        cursorBlink: true,
    })

    termInstance.open(mountEl)

    // 创建 WebSocket 连接，传递shell类型参数
    const protocol = location.protocol === 'https:' ? 'wss://' : 'ws://'
    socketInstance = new WebSocket(protocol + location.host + '/ws/shell?token=' + token + '&shell=' + encodeURIComponent(shellType))

    termInstance.focus()

    // 设置事件监听
    termInstance.onData(data => {
        if (socketInstance && socketInstance.readyState === WebSocket.OPEN) {
            socketInstance.send(data)
        }
    })

    socketInstance.onopen = () => {
        termInstance.write('[等待终端连接...]\r\n')
    }

    socketInstance.onmessage = (e) => {
        termInstance.write(e.data)
    }

    socketInstance.onclose = () => {
        termInstance.write('\r\n[终端连接已关闭]\r\n')
    }

    socketInstance.onerror = (error) => {
        termInstance.write('\r\n[终端连接错误: ' + error.message + ']\r\n')
    }
}

export function destroy() {
    if (termInstance) {
        termInstance.dispose()
        termInstance = null
    }
    if (socketInstance) {
        socketInstance.close()
        socketInstance = null
    }
}
