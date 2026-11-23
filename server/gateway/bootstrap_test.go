package gateway

import (
	"testing"
)

func TestValidWebLink(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{"https://example.com/path?q=1#section", true},
		{"http://127.0.0.1:8080/dashboard", true},
		{"http://", false},
		{"https:///missing-host", false},
		{"javascript:alert(1)", false},
		{"https://user:pass@example.com/", false},
		{"https://example.com/\nnext", false},
		{"//example.com/path", false},
	}
	for _, tt := range tests {
		if got := validWebLink(tt.value); got != tt.want {
			t.Errorf("validWebLink(%q) = %v, want %v", tt.value, got, tt.want)
		}
	}
}

func TestApplyNodeBootstrapUntrustedValues(t *testing.T) {
	cfg := map[string]any{
		"title": "center",
		"marketplaceUrl": "https://center.example.com",
	}
	probe := map[string]any{"docker": false, "caddy": false}
	remoteCfg := map[string]any{
		"title": "node",
		"marketplaceUrl": "javascript:alert(1)",
	}
	remoteProbe := map[string]any{
		"docker": "false",
		"caddy": true,
		"swarm": 1,
	}
	applyNodeBootstrap(cfg, probe, nil, remoteCfg, remoteProbe)
	if cfg["title"] != "node" {
		t.Fatalf("node title was not applied: %v", cfg["title"])
	}
	if cfg["marketplaceUrl"] != "https://center.example.com" {
		t.Fatalf("invalid node marketplace URL was accepted: %v", cfg["marketplaceUrl"])
	}
	if probe["docker"] != false || probe["caddy"] != true {
		t.Fatalf("boolean probe merge mismatch: %#v", probe)
	}
	if _, ok := probe["swarm"]; ok {
		t.Fatal("non-boolean probe was accepted")
	}
}

func TestMergeLinksRejectsInvalidURLs(t *testing.T) {
	center := []any{
		map[string]any{"label": "docs", "url": "https://example.com/docs", "icon": "book"},
	}
	remote := []any{
		map[string]any{"label": "duplicate", "url": "https://example.com/docs", "icon": "duplicate"},
		map[string]any{"label": "invalid", "url": "http://", "icon": "alert"},
		map[string]any{"label": "credentials", "url": "https://user:pass@example.com/", "icon": "alert"},
		map[string]any{"label": "safe", "url": "https://node.example.com/", "icon": "server"},
	}
	got := mergeLinks(center, remote)
	if len(got) != 2 {
		t.Fatalf("mergeLinks yielded %d links, want 2: %#v", len(got), got)
	}
	first := got[0].(map[string]any)
	second := got[1].(map[string]any)
	if first["label"] != "docs" || second["label"] != "safe" {
		t.Fatalf("link order or filtering mismatch: %#v", got)
	}
}
