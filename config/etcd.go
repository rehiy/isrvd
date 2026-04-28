package config

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rehiy/pango/logman"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type etcdStore struct {
	cli    *clientv3.Client
	prefix string
}

func newEtcdStore(cfg *EtcdConfig) (*etcdStore, error) {
	if cfg.Prefix == "" {
		cfg.Prefix = "/isrvd"
	}
	cfg.Prefix = strings.TrimRight(cfg.Prefix, "/")

	etcdCfg := clientv3.Config{
		Endpoints:   cfg.Endpoints,
		Username:    cfg.Username,
		Password:    cfg.Password,
		DialTimeout: 5 * time.Second,
	}

	if cfg.TLS != nil {
		tlsConfig, err := buildTLSConfig(cfg.TLS)
		if err != nil {
			return nil, fmt.Errorf("etcd tls: %w", err)
		}
		etcdCfg.TLS = tlsConfig
	}

	cli, err := clientv3.New(etcdCfg)
	if err != nil {
		return nil, fmt.Errorf("etcd connect: %w", err)
	}

	return &etcdStore{
		cli:    cli,
		prefix: cfg.Prefix,
	}, nil
}

func buildTLSConfig(tlsCfg *EtcdTLS) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(tlsCfg.CertFile, tlsCfg.KeyFile)
	if err != nil {
		return nil, err
	}
	caData, err := os.ReadFile(tlsCfg.CAFile)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caData) {
		return nil, fmt.Errorf("failed to append ca certs")
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
	}, nil
}

func (s *etcdStore) key(name string) string {
	return s.prefix + "/config/" + name
}

func (s *etcdStore) Load(ctx context.Context) (*RemoteConfig, int64, error) {
	keys := []string{"agent", "apisix", "marketplace", "links", "members", "docker", "server"}
	data := make(map[string][]byte)
	var maxRev int64

	for _, k := range keys {
		resp, err := s.cli.Get(ctx, s.key(k))
		if err != nil {
			return nil, 0, fmt.Errorf("etcd get %s: %w", k, err)
		}
		if len(resp.Kvs) > 0 {
			data[k] = resp.Kvs[0].Value
			if resp.Kvs[0].ModRevision > maxRev {
				maxRev = resp.Kvs[0].ModRevision
			}
		}
		if resp.Header.Revision > maxRev {
			maxRev = resp.Header.Revision
		}
	}

	rc, err := bytesToRemote(data)
	if err != nil {
		return nil, 0, err
	}
	return rc, maxRev, nil
}

func (s *etcdStore) Save(ctx context.Context, rc *RemoteConfig) error {
	items, err := remoteToBytes(rc)
	if err != nil {
		return err
	}
	for k, v := range items {
		_, err := s.cli.Put(ctx, s.key(k), string(v))
		if err != nil {
			return fmt.Errorf("etcd put %s: %w", k, err)
		}
	}
	return nil
}

func (s *etcdStore) Watch(ctx context.Context, rev int64, onChange func(key string, value []byte)) error {
	watchKey := s.prefix + "/config/"
	for {
		watcher := s.cli.Watch(ctx, watchKey, clientv3.WithPrefix(), clientv3.WithRev(rev))
		for resp := range watcher {
			if resp.CompactRevision > 0 {
				logman.Warn("etcd compacted", "compact_revision", resp.CompactRevision)
				onChange("_compacted", nil)
				rev = resp.CompactRevision
				break
			}
		if resp.Canceled {
			logman.Warn("etcd watch canceled", "error", resp.Err())
			onChange("_canceled", nil)
			break
		}
			for _, ev := range resp.Events {
				key := strings.TrimPrefix(string(ev.Kv.Key), watchKey)
				onChange(key, ev.Kv.Value)
				rev = ev.Kv.ModRevision + 1
			}
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		backoff := 1 * time.Second
		for {
			logman.Info("etcd watch reconnecting", "after", backoff)
			time.Sleep(backoff)
			resp, err := s.cli.Get(ctx, watchKey, clientv3.WithPrefix(), clientv3.WithLimit(1))
			if err == nil {
				if resp.Header.Revision > 0 {
					rev = resp.Header.Revision + 1
				}
				break
			}
			if backoff < 30*time.Second {
				backoff *= 2
			}
		}
	}
}

func (s *etcdStore) Close() error {
	if s.cli != nil {
		return s.cli.Close()
	}
	return nil
}
