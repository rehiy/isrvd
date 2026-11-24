package apisix

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	pkgApisix "isrvd/pkgs/apisix"
)

func TestConsumerKeyConcurrentIssuanceAndRetry(t *testing.T) {
	var mu sync.Mutex
	var value map[string]any
	writes := 0
	failWriteResponse := true
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.Method == http.MethodPut {
			writes++
			if err := json.NewDecoder(r.Body).Decode(&value); err != nil {
				t.Error(err)
			}
			if failWriteResponse {
				failWriteResponse = false
				http.Error(w, "lost response", 502)
				return
			}
		}
		if value == nil {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"value": value})
	}))
	defer upstream.Close()
	s := &Service{client: pkgApisix.NewClient(upstream.URL, "admin-fixture")}
	if _, err := s.ConsumerKeyUpdate(t.Context(), "alice", ConsumerKeyUpdateRequest{Key: "first-key", CreateOnly: true}); err == nil {
		t.Fatal("ambiguous write reported success")
	}
	var group sync.WaitGroup
	for i := range 16 {
		group.Go(func() {
			key, err := s.ConsumerKeyUpdate(t.Context(), "alice", ConsumerKeyUpdateRequest{Key: fmt.Sprintf("proposal-%d", i), CreateOnly: true})
			if err != nil || key.Key != "first-key" {
				t.Errorf("retry rotated Token: %+v %v", key, err)
			}
		})
	}
	group.Wait()
	if writes != 1 {
		t.Fatalf("issued %d times", writes)
	}
	// A new service instance after restart still retrieves the original upstream key.
	s = &Service{client: pkgApisix.NewClient(upstream.URL, "admin-fixture")}
	key, err := s.ConsumerKeyUpdate(t.Context(), "alice", ConsumerKeyUpdateRequest{Key: "restart-key", CreateOnly: true})
	if err != nil || key.Key != "first-key" || writes != 1 {
		t.Fatalf("restart rotated Token: %+v %v", key, err)
	}
	mu.Lock()
	value = nil
	writes = 0
	mu.Unlock()
	keys := make(chan string, 16)
	for i := range 16 {
		group.Go(func() {
			key, err := s.ConsumerKeyUpdate(t.Context(), "alice", ConsumerKeyUpdateRequest{Key: fmt.Sprintf("fresh-%d", i), CreateOnly: true})
			if err != nil {
				t.Error(err)
				return
			}
			keys <- key.Key
		})
	}
	group.Wait()
	close(keys)
	first := ""
	for key := range keys {
		if first == "" {
			first = key
		}
		if key != first {
			t.Fatal("concurrent creation returned different Tokens")
		}
	}
	if writes != 1 || first == "" {
		t.Fatalf("concurrent creation wrote %d times", writes)
	}
}
