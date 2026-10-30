package gateway

import (
	"encoding/json"
	"net/http"

	"isrvd/server/i18n"
)

// ─── 辅助函数 ───

func writeRaw(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// 网关是原生 http.Handler，没有 gin 上下文，提示文案按请求语言直接翻译。

func success(w http.ResponseWriter, r *http.Request, message string, payload any) {
	data, _ := json.Marshal(map[string]any{"success": true, "message": i18n.TR(r, message), "payload": payload})
	writeRaw(w, http.StatusOK, data)
}

func fail(w http.ResponseWriter, r *http.Request, status int, message string) {
	data, _ := json.Marshal(map[string]any{"success": false, "message": i18n.TR(r, message)})
	writeRaw(w, status, data)
}
