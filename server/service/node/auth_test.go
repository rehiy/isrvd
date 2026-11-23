package node

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestRateLimiterSlidingWindow(t *testing.T) {
	limiter := &rateLimiter{hits: make(map[string][]time.Time)}
	const source = "192.0.2.1"
	for i := 0; i < 10; i++ {
		if !limiter.allow(source) {
			t.Fatalf("blocked source after only %d failures", i)
		}
		limiter.record(source)
	}
	if limiter.allow(source) {
		t.Fatal("source should be rate limited after 10 failures")
	}

	// Expire the full window without sleeping and verify the source can retry.
	limiter.mu.Lock()
	limiter.hits[source] = []time.Time{time.Now().Add(-2 * time.Minute)}
	limiter.mu.Unlock()
	if !limiter.allow(source) {
		t.Fatal("source should recover after the one-minute failure window")
	}
	limiter.mu.Lock()
	_, retained := limiter.hits[source]
	limiter.mu.Unlock()
	if retained {
		t.Fatal("expired source should be removed from the map")
	}
}

func TestRateLimiterSourceCapacity(t *testing.T) {
	const maxSources = 4096
	limiter := &rateLimiter{hits: make(map[string][]time.Time)}
	for i := 0; i < maxSources; i++ {
		limiter.record(fmt.Sprintf("source-%d", i))
	}
	if len(limiter.hits) != maxSources {
		t.Fatalf("initial capacity = %d, want %d", len(limiter.hits), maxSources)
	}
	limiter.record("new-source")
	if len(limiter.hits) != maxSources {
		t.Fatalf("capacity increased to %d", len(limiter.hits))
	}
	if _, ok := limiter.hits["new-source"]; !ok {
		t.Fatal("new source should be admitted after oldest-source eviction")
	}
	if _, ok := limiter.hits["source-0"]; ok {
		t.Fatal("oldest source should be evicted to keep a bounded map")
	}
}

func TestRateLimiterConcurrentSourcesStayBounded(t *testing.T) {
	limiter := &rateLimiter{hits: make(map[string][]time.Time)}
	var wg sync.WaitGroup
	for worker := 0; worker < 16; worker++ {
		wg.Go(func() {
			for i := 0; i < 300; i++ {
				limiter.record(fmt.Sprintf("%d-%d", worker, i))
			}
		})
	}
	wg.Wait()
	limiter.mu.Lock()
	count := len(limiter.hits)
	limiter.mu.Unlock()
	if count > 4096 {
		t.Fatalf("concurrent failures exceeded source capacity: %d", count)
	}
}
