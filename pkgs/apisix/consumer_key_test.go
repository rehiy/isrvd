package apisix

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestConsumerKeyPreservesExistingData(t *testing.T) {
	value := map[string]any{"username": "alice", "desc": "original", "group_id": "team", "labels": map[string]any{"owner": "ops"},
		"plugins": map[string]any{"key-auth": map[string]any{"key": "original-key", "_meta": map[string]any{"disable": false}}, "jwt-auth": map[string]any{"secret": "jwt-secret"}}}
	writes := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/consumers/alice" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method == http.MethodPut {
			writes++
			if err := json.NewDecoder(r.Body).Decode(&value); err != nil {
				t.Error(err)
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"value": value})
	}))
	defer upstream.Close()
	c := NewClient(upstream.URL, "admin-fixture")
	desc := "do-not-apply"
	key, err := c.ConsumerKeyUpdate(t.Context(), "alice", "proposed-key", &desc, true)
	if err != nil || key.Key != "original-key" || key.Desc != "original" || writes != 0 {
		t.Fatalf("existing Token was changed: %+v, %v", key, err)
	}
	key, err = c.ConsumerKeyUpdate(t.Context(), "alice", "rotated-key", nil, false)
	if err != nil || key.Key != "rotated-key" || key.Desc != "original" || writes != 1 {
		t.Fatalf("rotation failed: %+v, %v", key, err)
	}
	plugins := value["plugins"].(map[string]any)
	if value["group_id"] != "team" || value["labels"].(map[string]any)["owner"] != "ops" ||
		plugins["jwt-auth"].(map[string]any)["secret"] != "jwt-secret" || plugins["key-auth"].(map[string]any)["_meta"] == nil {
		t.Fatal("unrelated Consumer data changed")
	}
	encoded, _ := json.Marshal(key)
	if strings.Contains(string(encoded), "jwt-secret") {
		t.Fatal("response leaked another plugin's secret")
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
		{"wrong-account", 200, `{"value":{"username":"bob","plugins":{"key-auth":{"key":"secret-fixture"}}}}`, ErrConsumerResponseInvalid},
		{"masked", 200, `{"value":{"username":"alice","plugins":{"key-auth":{"key":"abcde******xyz"}}}}`, ErrConsumerResponseInvalid},
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

func TestConsumerKeyValidatesBeforeRequest(t *testing.T) {
	c := NewClient("http://unused.invalid", "unused")
	for _, name := range []string{"", ".", "..", "a/b", "a%2fb", " a", "a?b", "a#b"} {
		if _, err := c.ConsumerKeyInspect(t.Context(), name); !errors.Is(err, ErrConsumerKeyInvalid) {
			t.Fatalf("accepted %q: %v", name, err)
		}
	}
	for _, key := range []string{"", " ", "abc******def", "key\nvalue", strings.Repeat("a", 4097)} {
		if _, err := c.ConsumerKeyUpdate(t.Context(), "alice", key, nil, true); !errors.Is(err, ErrConsumerKeyInvalid) {
			t.Fatalf("invalid key accepted: %v", err)
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
