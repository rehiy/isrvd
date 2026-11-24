package apisix

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestConsumerKeyPreservesExistingData(t *testing.T) {
	const exactLargeID = "9007199254740993"
	rawDoc := `{"username":"alice","desc":"original","group_id":"team","create_time":1700000000,"update_time":1700000010,"labels":{"owner":"ops"},"plugins":{"key-auth":{"key":"original-key","_meta":{"disable":false}},"jwt-auth":{"secret":"jwt-secret","snowflake":` + exactLargeID + `}}}`
	var value map[string]any
	decodeUseNumber := func(r io.Reader, dst any) error {
		dec := json.NewDecoder(r)
		dec.UseNumber()
		return dec.Decode(dst)
	}
	if err := decodeUseNumber(strings.NewReader(rawDoc), &value); err != nil {
		t.Fatal(err)
	}

	writes := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/consumers/alice" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method == http.MethodPut {
			writes++
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
				return
			}
			var putBody map[string]any
			if err := decodeUseNumber(bytes.NewReader(bodyBytes), &putBody); err != nil {
				t.Error(err)
				return
			}
			if _, hasCreate := putBody["create_time"]; hasCreate {
				t.Error("PUT unexpectedly sent read-only create_time")
			}
			if _, hasUpdate := putBody["update_time"]; hasUpdate {
				t.Error("PUT unexpectedly sent read-only update_time")
			}
			putBody["create_time"] = json.Number("1700000000")
			putBody["update_time"] = json.Number("1700000099")
			value = putBody
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"value": value})
	}))
	defer upstream.Close()

	c := NewClient(upstream.URL, "admin-fixture")
	desc := "do-not-apply"
	key, err := c.ConsumerKeyUpdate(t.Context(), "alice", "proposed-key", &desc, true)
	if err != nil || key.Key != "original-key" || key.Desc != "original" || key.UpdateTime != 1700000010 || writes != 0 {
		t.Fatalf("existing Token was changed: %+v, %v", key, err)
	}

	key, err = c.ConsumerKeyUpdate(t.Context(), "alice", "rotated-key", nil, false)
	if err != nil || key.Key != "rotated-key" || key.Desc != "original" || key.CreateTime != 1700000000 || key.UpdateTime != 1700000099 || writes != 1 {
		t.Fatalf("rotation failed or timestamps not refreshed: %+v, %v", key, err)
	}

	plugins := value["plugins"].(map[string]any)
	jwtPlugin := plugins["jwt-auth"].(map[string]any)
	if value["group_id"] != "team" || value["labels"].(map[string]any)["owner"] != "ops" ||
		jwtPlugin["secret"] != "jwt-secret" || jwtPlugin["snowflake"] != json.Number(exactLargeID) ||
		plugins["key-auth"].(map[string]any)["_meta"] == nil {
		t.Fatalf("unrelated Consumer data or 64-bit integer changed: %+v", value)
	}
	encoded, _ := json.Marshal(key)
	if strings.Contains(string(encoded), "jwt-secret") {
		t.Fatal("response leaked another plugin's secret")
	}

	// Disabled key-auth must not be returned as active or silently re-enabled.
	plugins["key-auth"].(map[string]any)["_meta"] = map[string]any{"disable": true}
	if _, err := c.ConsumerKeyInspect(t.Context(), "alice"); !errors.Is(err, ErrConsumerKeyMissing) {
		t.Fatalf("inspect returned disabled key-auth: %v", err)
	}
	if _, err := c.ConsumerKeyUpdate(t.Context(), "alice", "new-key", nil, true); !errors.Is(err, ErrConsumerKeyMissing) || writes != 1 {
		t.Fatalf("create-only mutated disabled key-auth: %v", err)
	}
	if _, err := c.ConsumerKeyUpdate(t.Context(), "alice", "new-key", nil, false); !errors.Is(err, ErrConsumerKeyMissing) || writes != 1 {
		t.Fatalf("update bypassed disabled key-auth: %v", err)
	}

	delete(plugins, "key-auth")
	if _, err := c.ConsumerKeyUpdate(t.Context(), "alice", "new-key", nil, true); !errors.Is(err, ErrConsumerKeyMissing) || writes != 1 {
		t.Fatalf("create-only changed account without key-auth: %v", err)
	}
}

func TestConsumerKeyRejectsUpstreamFailures(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{"missing", 404, "secret-fixture", ErrConsumerNotFound},
		{"forbidden", 403, "secret-fixture", nil},
		{"unavailable", 503, "secret-fixture", nil},
		{"bad-json", 200, "secret-fixture", nil},
		{"null-value", 200, `{"value":null}`, ErrConsumerResponseInvalid},
		{"trailing-json", 200, `{"value":{"username":"alice","plugins":{"key-auth":{"key":"secret-fixture"}}}}{}`, ErrConsumerResponseInvalid},
		{"wrong-account", 200, `{"value":{"username":"bob","plugins":{"key-auth":{"key":"secret-fixture"}}}}`, ErrConsumerResponseInvalid},
		{"masked", 200, `{"value":{"username":"alice","plugins":{"key-auth":{"key":"abcde******xyz"}}}}`, ErrConsumerResponseInvalid},
		{"bad-plugins-type", 200, `{"value":{"username":"alice","plugins":"invalid"}}`, ErrConsumerResponseInvalid},
		{"bad-meta-type", 200, `{"value":{"username":"alice","plugins":{"key-auth":{"key":"secret-fixture","_meta":"invalid"}}}}`, ErrConsumerResponseInvalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			writes := 0
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					writes++
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer upstream.Close()
			c := NewClient(upstream.URL, "admin-fixture")
			_, err := c.ConsumerKeyInspect(t.Context(), "alice")
			if err == nil || (tc.want != nil && !errors.Is(err, tc.want)) || strings.Contains(err.Error(), "secret-fixture") {
				t.Fatalf("unexpected or leaking error: %v", err)
			}
			if tc.status != 404 {
				if _, err := c.ConsumerKeyUpdate(t.Context(), "alice", "new-key", nil, true); err == nil || writes != 0 {
					t.Fatalf("upstream error caused a write: %v", err)
				}
			}
		})
	}
}

func TestConsumerErrorRedactionPreservesRouteDiagnostics(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		if strings.HasPrefix(r.URL.Path, "/consumers") {
			_, _ = w.Write([]byte("consumer-secret-echo"))
			return
		}
		_, _ = w.Write([]byte(`{"error_msg":"invalid route uri"}`))
	}))
	defer upstream.Close()
	c := NewClient(upstream.URL, "admin-fixture")

	if _, err := c.ConsumerKeyInspect(t.Context(), "alice"); err == nil || strings.Contains(err.Error(), "consumer-secret-echo") {
		t.Fatalf("consumer error leaked body: %v", err)
	}
	if _, err := c.RouteInspect(t.Context(), "r1"); err == nil || !strings.Contains(err.Error(), "invalid route uri") {
		t.Fatalf("non-consumer error lost APISIX diagnostic message: %v", err)
	}
}

func TestConsumerKeyValidatesBeforeRequest(t *testing.T) {
	c := NewClient("http://unused.invalid", "unused")
	for _, name := range []string{"", ".", "..", "a/b", "a%2fb", " a", "a?b", "a#b", "\xff"} {
		if _, err := c.ConsumerKeyInspect(t.Context(), name); !errors.Is(err, ErrConsumerKeyInvalid) {
			t.Fatalf("accepted %q: %v", name, err)
		}
	}
	for _, key := range []string{"", " ", " leading", "trailing ", "a b", "\xff", "abc******def", "key\nvalue", strings.Repeat("a", 4097)} {
		if _, err := c.ConsumerKeyUpdate(t.Context(), "alice", key, nil, true); !errors.Is(err, ErrConsumerKeyInvalid) {
			t.Fatalf("invalid key %q accepted: %v", key, err)
		}
	}
	for _, badDesc := range []string{"\xff", "bad\x00desc"} {
		if _, err := c.ConsumerKeyUpdate(t.Context(), "alice", "valid-key", &badDesc, true); !errors.Is(err, ErrConsumerKeyInvalid) {
			t.Fatalf("invalid desc %q accepted: %v", badDesc, err)
		}
	}
}

func TestConsumerListRemainsMasked(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"list":[{"value":{"username":"alice","plugins":{"key-auth":{"key":"original-secret-token"}}}}]}`))
	}))
	defer upstream.Close()
	rows, err := NewClient(upstream.URL, "admin-fixture").ConsumerList(t.Context())
	encoded, _ := json.Marshal(rows)
	if err != nil || strings.Contains(string(encoded), "original-secret-token") || !strings.Contains(string(encoded), "******") {
		t.Fatalf("list masking changed: %v", err)
	}
}
