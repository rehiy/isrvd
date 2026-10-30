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

func respond(w http.ResponseWriter, status int, body map[string]any) {
	data, _ := json.Marshal(body)
	writeRaw(w, status, data)
}

// translate 网关是原生 http.Handler，没有 gin 上下文，按请求头直接协商语言翻译提示文案
func translate(r *http.Request, message string) string {
	return i18n.Translate(i18n.Parse(r.Header.Get("Accept-Language")), message)
}

func success(w http.ResponseWriter, r *http.Request, message string, payload any) {
	respond(w, http.StatusOK, map[string]any{"success": true, "message": translate(r, message), "payload": payload})
}

func fail(w http.ResponseWriter, r *http.Request, status int, message string) {
	respond(w, status, map[string]any{"success": false, "message": translate(r, message)})
}
