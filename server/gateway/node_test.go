package gateway

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"isrvd/server/service/node"
)

func TestDecodeNodeBody(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		status int
		want   string
	}{
		{"one object", "{\"name\":\"demo\"}", http.StatusOK, "demo"},
		{"whitespace suffix", " \n{\"name\":\"demo\"}\n \t", http.StatusOK, "demo"},
		{"multiple JSON values", "{\"name\":\"demo\"}{\"name\":\"other\"}", http.StatusBadRequest, ""},
		{"trailing garbage", "{\"name\":\"demo\"}garbage", http.StatusBadRequest, ""},
		{"partial JSON", "{\"name\":", http.StatusBadRequest, ""},
		{"empty body", "", http.StatusBadRequest, ""},
		{"exceeds limit after JSON", "{\"name\":\"demo\"}" + strings.Repeat(" ", 1<<20), http.StatusRequestEntityTooLarge, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/node", strings.NewReader(tt.body))
			w := httptest.NewRecorder()
			var value node.NodeRequest
			ok := decode(w, req, &value)
			if ok != (tt.status == http.StatusOK) {
				t.Fatalf("decode success=%v, expected status=%d, body=%s", ok, tt.status, w.Body.String())
			}
			if w.Code != tt.status {
				t.Fatalf("status=%d, want=%d, body=%s", w.Code, tt.status, w.Body.String())
			}
			if ok && value.Name != tt.want {
				t.Fatalf("name=%q, want=%q", value.Name, tt.want)
			}
		})
	}
}
