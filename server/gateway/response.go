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
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func success(w http.ResponseWriter, r *http.Request, message string, payload any) {
	// 与 fail 一致：按请求语言翻译提示文案
	respond(w, http.StatusOK, map[string]any{
		"success": true,
		"message": i18n.Translate(i18n.Parse(r.Header.Get("Accept-Language")), message),
		"payload": payload,
	})
}

func fail(w http.ResponseWriter, r *http.Request, status int, message string) {
	// 网关是原生 http.Handler，没有 gin 上下文，按请求头直接协商语言
	lang := i18n.Parse(r.Header.Get("Accept-Language"))
	respond(w, status, map[string]any{"success": false, "message": i18n.Translate(lang, message)})
}
