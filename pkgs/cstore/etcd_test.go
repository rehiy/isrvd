package cstore

import (
	"strings"
	"testing"
)

func TestEtcdStoreRejectsInvalidConnectionSettings(t *testing.T) {
	tests := []struct {
		uri  string
		want string
	}{
		{"etcd://127.0.0.1:2379/app/config?scheme=ftp", "scheme"},
		{"etcd://127.0.0.1:2379/app/config?scheme=file", "scheme"},
		{"etcd://127.0.0.1:2379/app/config?timeout=0s", "timeout"},
		{"etcd://127.0.0.1:2379/app/config?timeout=-1s", "timeout"},
		{"etcd://,,/app/config", "endpoints"},
	}
	for _, tt := range tests {
		t.Run(tt.uri, func(t *testing.T) {
			_, err := newEtcdStore(tt.uri, options{})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("newEtcdStore(%q): expected %s validation error, got %v", tt.uri, tt.want, err)
			}
		})
	}
}
