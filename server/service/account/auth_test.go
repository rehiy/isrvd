package account

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExtractJWTQueryTokenOnlyForGET(t *testing.T) {
	svc := &Service{}
	tests := []struct {
		name       string
		method     string
		allowed    bool
		websocket  bool
		want       string
	}{
		{name: "get query-enabled", method: http.MethodGet, allowed: true, want: "query-secret"},
		{name: "get query-disabled", method: http.MethodGet, want: ""},
		{name: "get websocket", method: http.MethodGet, websocket: true, want: "query-secret"},
		{name: "post query-enabled", method: http.MethodPost, allowed: true, want: ""},
		{name: "post spoofed websocket", method: http.MethodPost, allowed: true, websocket: true, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/copilot/agui?token=query-secret", nil)
			if tt.websocket {
				req.Header.Set("Connection", "Upgrade")
				req.Header.Set("Upgrade", "websocket")
			}
			if got := svc.extractJWT(req, tt.allowed); got != tt.want {
				t.Errorf("extractJWT() = %q, want %q", got, tt.want)
			}
			req.Header.Set("Authorization", "Bearer header-secret")
			if got := svc.extractJWT(req, tt.allowed); got != "header-secret" {
				t.Errorf("header auth was not preserved: %q", got)
			}
		})
	}
}
