package cstore

import "testing"

func TestEtcdStoreUsesURIPathAsConfigKey(t *testing.T) {
	store, key, err := OpenWithKey("etcd://user:pass@127.0.0.1:2379/isrvd/config?scheme=http&fallback=/data/conf/isrvd.yml")
	if err != nil {
		t.Fatalf("OpenWithKey returned error: %v", err)
	}
	etcdStore, ok := store.(*EtcdStore)
	if !ok {
		t.Fatalf("store type = %T, want *EtcdStore", store)
	}
	if key != "" {
		t.Fatalf("key = %q, want empty key for full etcd path", key)
	}
	if got := etcdStore.etcdKey(key); got != "/isrvd/config" {
		t.Fatalf("etcd key = %q, want /isrvd/config", got)
	}
	if got := etcdStore.fallbackPath(key); got != "/data/conf/isrvd.yml" {
		t.Fatalf("fallback path = %q, want /data/conf/isrvd.yml", got)
	}
}

func TestEtcdStoreRequiresConfigKey(t *testing.T) {
	if _, _, err := OpenWithKey("etcd://127.0.0.1:2379"); err == nil {
		t.Fatal("OpenWithKey returned nil error for etcd URI without config key")
	}
	if _, _, err := OpenWithKey("etcd://127.0.0.1:2379/"); err == nil {
		t.Fatal("OpenWithKey returned nil error for etcd URI with root path only")
	}
}
