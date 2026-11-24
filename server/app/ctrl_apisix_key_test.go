package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"isrvd/server/config"
	"isrvd/server/service/account"
	"isrvd/server/service/apisix"
	"isrvd/server/service/system"
)

func TestConsumerKeyPermissionsResponsesAndAudit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reads, writes := 0, 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/routes" {
			_, _ = w.Write([]byte(`{"list":[]}`))
			return
		}
		if r.Method == http.MethodPut {
			writes++
		}
		reads++
		switch r.URL.Path {
		case "/consumers/missing":
			http.NotFound(w, r)
		case "/consumers/no-key":
			_, _ = w.Write([]byte(`{"value":{"username":"no-key"}}`))
		case "/consumers/disabled":
			_, _ = w.Write([]byte(`{"value":{"username":"disabled","plugins":{"key-auth":{"key":"disabled-secret-fixture","_meta":{"disable":true}}}}}`))
		case "/consumers/broken":
			http.Error(w, "upstream-secret-fixture", 403)
		case "/consumers/alice":
			_, _ = w.Write([]byte(`{"value":{"username":"alice","plugins":{"key-auth":{"key":"alice-original-token"},"jwt-auth":{"secret":"jwt-secret-fixture"}}}}`))
		default:
			t.Errorf("unexpected upstream: %s", r.URL.Path)
		}
	}))
	defer upstream.Close()
	previous := config.Current()
	cfg := *previous
	serverConfig := *cfg.Server
	serverConfig.RootDirectory = t.TempDir()
	cfg.Server = &serverConfig
	cfg.THA = &config.THAConfig{Enabled: true, HeaderName: "X-Test-User", TrustedCIDRs: []string{"127.0.0.1/32"}}
	cfg.Apisix = &config.ApisixConfig{AdminURL: upstream.URL, AdminKey: "admin-fixture"}
	readPermission := "GET /api/apisix/consumer/:username/key"
	writePermission := "PUT /api/apisix/consumer/:username/key"
	cfg.Members = map[string]*config.MemberConfig{
		"viewer": {Username: "viewer", Permissions: []string{"GET /api/apisix/consumers"}},
		"reader": {Username: "reader", Permissions: []string{readPermission}},
		"writer": {Username: "writer", Permissions: []string{writePermission}},
		"appkey": {Username: "appkey", Permissions: []string{readPermission, writePermission}},
	}
	config.Publish(&cfg)
	t.Cleanup(func() { config.Publish(previous) })
	svc, err := apisix.NewService(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	app := &App{Engine: gin.New(), routeIndex: map[string]Route{}, accountSvc: &account.Service{}, auditSvc: system.NewAuditService()}
	t.Cleanup(func() { _ = app.auditSvc.Close() })
	app.availability.Store(&availableServices{apisixSvc: svc})
	group := app.Group("/api")
	group.Use(app.authMiddleware(app.routeIndex), app.permMiddleware(app.routeIndex), app.auditMiddleware(app.routeIndex))
	for _, route := range app.defineApisixRoutes() {
		app.registerRoute(group, route)
	}
	// The member editor builds its checkboxes from this catalog, not a static list.
	catalogResponse := httptest.NewRecorder()
	catalogContext, _ := gin.CreateTestContext(catalogResponse)
	app.accountRouteList(catalogContext)
	var catalog struct {
		Payload []Route `json:"payload"`
	}
	if err := json.Unmarshal(catalogResponse.Body.Bytes(), &catalog); err != nil {
		t.Fatal(err)
	}
	for permission, label := range map[string]string{readPermission: "读取 APISIX 消费者完整 Token", writePermission: "签发 APISIX 消费者 Token"} {
		found := false
		for _, route := range catalog.Payload {
			if route.Key == permission && route.Label == label && route.Module == "apisix" && route.Access == AccessPerm && route.ExplicitGrant {
				found = true
			}
		}
		if !found {
			t.Fatalf("permission checkbox missing: %s", permission)
		}
	}
	call := func(user, method, name, body string, expected int) string {
		t.Helper()
		request := httptest.NewRequest(method, "/api/apisix/consumer/"+name+"/key", strings.NewReader(body))
		request.RemoteAddr = "127.0.0.1:12345"
		request.Header.Set("X-Test-User", user)
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		app.ServeHTTP(response, request)
		if response.Code != expected {
			t.Fatalf("%s %s as %s: HTTP %d: %s", method, name, user, response.Code, response.Body.String())
		}
		if expected == 200 && response.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatal("Token response may be cached")
		}
		for _, secret := range []string{"jwt-secret-fixture", "upstream-secret-fixture", "disabled-secret-fixture"} {
			if strings.Contains(response.Body.String(), secret) {
				t.Fatalf("response leaked secret %q", secret)
			}
		}
		return response.Body.String()
	}
	call("", "GET", "alice", "", 401)
	call("viewer", "GET", "alice", "", 403)
	call("writer", "PUT", "alice", `{"key":"proposed-fixture","create_only":true}`, 403)
	call("reader", "PUT", "alice", `{"key":"proposed-fixture","create_only":true}`, 403)
	if reads != 0 || writes != 0 {
		t.Fatal("unauthorized request reached upstream")
	}
	if len(app.auditSvc.LogList("writer", 0)) != 0 {
		t.Fatal("missing extra read permission was not rejected before audit middleware")
	}
	if !strings.Contains(call("reader", "GET", "alice", "", 200), "alice-original-token") {
		t.Fatal("read did not return original Token")
	}
	if !strings.Contains(call("appkey", "PUT", "alice", `{"key":"proposed-fixture","create_only":true}`, 200), "alice-original-token") || writes != 0 {
		t.Fatal("create-only failed to preserve existing Token")
	}
	// Reject camelCase typo createOnly instead of silently overwriting alice's Token as create_only=false.
	call("appkey", "PUT", "alice", `{"key":"proposed-fixture","createOnly":true}`, 400)
	call("appkey", "PUT", "alice", `null`, 400)
	call("appkey", "PUT", "alice", `{"key":"proposed-fixture"}{"key":"second"}`, 400)
	call("appkey", "PUT", "%ff", `{"key":"proposed-fixture","create_only":true}`, 400)
	if writes != 0 {
		t.Fatalf("rejected request mutated upstream %d times", writes)
	}
	tooLarge := call("appkey", "PUT", "alice", `{"key":"proposed-fixture","desc":"`+strings.Repeat("x", 70<<10)+`"}`, 413)
	if !strings.Contains(tooLarge, "请求体过大") {
		t.Fatalf("unexpected 413 message: %s", tooLarge)
	}
	call("reader", "GET", "missing", "", 404)
	call("reader", "GET", "no-key", "", 409)
	call("reader", "GET", "disabled", "", 409)
	call("reader", "GET", "broken", "", 502)
	call("appkey", "PUT", "alice", `{"key":""}`, 400)
	logs := app.auditSvc.LogList("", 0)
	encoded, _ := json.Marshal(logs)
	for _, secret := range []string{"alice-original-token", "proposed-fixture", "jwt-secret-fixture", "disabled-secret-fixture"} {
		if strings.Contains(string(encoded), secret) {
			t.Fatal("audit leaked a Token")
		}
	}
	found := false
	for _, entry := range logs {
		if entry.Method == "GET" && entry.Username == "reader" && strings.HasSuffix(entry.URI, "/alice/key") && entry.Success {
			found = true
		}
	}
	if !found {
		t.Fatal("successful Token read was not audited")
	}
	// A node request must check the original caller, not the receiving agent's founder role.
	app.lifecycleCtx = t.Context()
	app.resetRequestContext()
	defer app.requestsCancel()
	app.initGatewayPermissions()
	before := running.Load()
	running.Store(app)
	defer running.Store(before)
	for _, user := range []string{"writer", "appkey"} {
		r := httptest.NewRequest(http.MethodPut, "/api/apisix/consumer/alice/key", nil)
		r.RemoteAddr = "127.0.0.1:12345"
		r.Header.Set("X-Test-User", user)
		account.TrustGatewayIdentity(r)
		_, _, status, _ := GatewayAuthorize(r, false)
		want := http.StatusForbidden
		if user == "appkey" {
			want = http.StatusOK
		}
		if status != want {
			t.Fatalf("node authorization as %s returned %d, want %d", user, status, want)
		}
	}

	// Upstream network failure must not expose internal AdminURL to non-system callers.
	upstream.Close()
	netFail := call("reader", "GET", "alice", "", 502)
	if strings.Contains(netFail, upstream.URL) {
		t.Fatalf("502 response leaked internal AdminURL: %s", netFail)
	}
}
