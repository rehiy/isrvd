package helper

import (
	"net/http"

	"github.com/gorilla/websocket"
)

// WsUpgrader WebSocket 升级器（全局共享）
var WsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}
