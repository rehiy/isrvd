package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAllowQueryTokenOnlyForGET(t *testing.T) {
	g := &gateway{queryRoutes: [][]string{{"", "api", "demo"}}}
	tests := []struct {
		name      string
		method    string
		path      string
		websocket bool
		want      bool
	}{
		{name: "declared get route", method: http.MethodGet, path: "/api/demo", want: true},
		{name: "other get route", method: http.MethodGet, path: "/api/other"},
		{name: "declared post route", method: http.MethodPost, path: "/api/demo"},
		{name: "get websocket", method: http.MethodGet, path: "/api/other", websocket: true, want: true},
		{name: "post spoofed websocket", method: http.MethodPost, path: "/api/demo", websocket: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path+"?token=secret", nil)
			if tt.websocket {
				req.Header.Set("Connection", "Upgrade")
				req.Header.Set("Upgrade", "websocket")
			}
			if got := g.allowQueryToken(req); got != tt.want {
				t.Errorf("allowQueryToken() = %v, want %v", got, tt.want)
			}
		})
	}
}
