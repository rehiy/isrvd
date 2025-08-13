let __shellTerm = null;
let __shellSocket = null;

window.createShellTerminal = function (mountEl) {
    if (!mountEl) return;
    if (__shellTerm) {
        __shellTerm.dispose();
        __shellTerm = null;
    }
    if (__shellSocket) {
        __shellSocket.close();
        __shellSocket = null;
    }
    __shellTerm = new window.Terminal({
        theme: { background: '#222' },
        fontSize: 15,
        cursorBlink: true,
    });
    __shellTerm.open(mountEl);
    __shellSocket = new WebSocket((location.protocol === 'https:' ? 'wss://' : 'ws://') + location.host + '/ws/shell');
    __shellTerm.focus();
    __shellTerm.onData(data => {
        if (__shellSocket && __shellSocket.readyState === 1) {
            __shellSocket.send(data);
        }
    });
    __shellSocket.onopen = () => {
        __shellTerm.write('[等待终端连接...]\r\n');
    };
    __shellSocket.onmessage = e => {
        __shellTerm.write(e.data);
    };
    __shellSocket.onclose = () => {
        __shellTerm.write('\r\n[终端连接已关闭]\r\n');
    };
};

window.destroyShellTerminal = function () {
    if (__shellTerm) {
        __shellTerm.dispose();
        __shellTerm = null;
    }
    if (__shellSocket) {
        __shellSocket.close();
        __shellSocket = null;
    }
};
