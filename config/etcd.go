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
	prefix := cfg.Prefix
	if prefix == "" {
		prefix = "/isrvd"
	}
	prefix = strings.TrimRight(prefix, "/")

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
		prefix: prefix,
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

func (s *etcdStore) bootstrapKey() string {
	return s.prefix + "/meta/initialized"
}

func (s *etcdStore) revisionKey() string {
	return s.prefix + "/meta/revision"
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

	revResp, err := s.cli.Get(ctx, s.revisionKey())
	if err != nil {
		return nil, 0, fmt.Errorf("etcd get revision: %w", err)
	}
	if len(revResp.Kvs) > 0 && revResp.Kvs[0].ModRevision > maxRev {
		maxRev = revResp.Kvs[0].ModRevision
	}
	if revResp.Header.Revision > maxRev {
		maxRev = revResp.Header.Revision
	}

	rc, err := bytesToRemote(data)
	if err != nil {
		return nil, 0, err
	}
	return rc, maxRev, nil
}

func (s *etcdStore) Save(ctx context.Context, rc *RemoteConfig, expectedRevision int64) (int64, error) {
	items, err := remoteToBytes(rc)
	if err != nil {
		return 0, err
	}

	orderedKeys := []string{"agent", "apisix", "marketplace", "links", "members", "docker", "server"}
	ops := make([]clientv3.Op, 0, len(items)+1)
	for _, k := range orderedKeys {
		v, ok := items[k]
		if !ok {
			continue
		}
		ops = append(ops, clientv3.OpPut(s.key(k), string(v)))
	}
	ops = append(ops, clientv3.OpPut(s.revisionKey(), fmt.Sprintf("%d", time.Now().UnixNano())))

	cmpRevision := expectedRevision
	if cmpRevision <= 0 {
		resp, err := s.cli.Get(ctx, s.revisionKey())
		if err != nil {
			return 0, fmt.Errorf("etcd get revision: %w", err)
		}
		if len(resp.Kvs) > 0 {
			cmpRevision = resp.Kvs[0].ModRevision
		}
	}

	cmp := clientv3.Compare(clientv3.ModRevision(s.revisionKey()), "=", cmpRevision)
	if cmpRevision == 0 {
		cmp = clientv3.Compare(clientv3.Version(s.revisionKey()), "=", 0)
	}

	resp, err := s.cli.Txn(ctx).
		If(cmp).
		Then(ops...).
		Commit()
	if err != nil {
		return 0, fmt.Errorf("etcd save failed: %w", err)
	}
	if !resp.Succeeded {
		return 0, ErrRemoteConfigConflict
	}
	return resp.Header.Revision, nil
}

func (s *etcdStore) Bootstrap(ctx context.Context, rc *RemoteConfig) (bool, error) {
	items, err := remoteToBytes(rc)
	if err != nil {
		return false, err
	}

	orderedKeys := []string{"agent", "apisix", "marketplace", "links", "members", "docker", "server"}
	ops := make([]clientv3.Op, 0, len(items)+2)
	for _, k := range orderedKeys {
		v, ok := items[k]
		if !ok {
			continue
		}
		ops = append(ops, clientv3.OpPut(s.key(k), string(v)))
	}
	ops = append(ops, clientv3.OpPut(s.bootstrapKey(), fmt.Sprintf("%d", time.Now().UnixNano())))
	ops = append(ops, clientv3.OpPut(s.revisionKey(), fmt.Sprintf("%d", time.Now().UnixNano())))

	resp, err := s.cli.Txn(ctx).
		If(clientv3.Compare(clientv3.Version(s.bootstrapKey()), "=", 0)).
		Then(ops...).
		Commit()
	if err != nil {
		return false, fmt.Errorf("etcd bootstrap failed: %w", err)
	}
	return resp.Succeeded, nil
}

func (s *etcdStore) Watch(ctx context.Context, rev int64, onChange func(key string, value []byte, rev int64)) error {
	watchKey := s.prefix + "/config/"
	for {
		watcher := s.cli.Watch(ctx, watchKey, clientv3.WithPrefix(), clientv3.WithRev(rev))
		for resp := range watcher {
			if resp.CompactRevision > 0 {
				logman.Warn("etcd compacted", "compact_revision", resp.CompactRevision)
				onChange("_compacted", nil, resp.Header.Revision)
				rev = resp.CompactRevision
				break
			}
			if resp.Canceled {
				logman.Warn("etcd watch canceled", "error", resp.Err())
				onChange("_canceled", nil, resp.Header.Revision)
				break
			}
			for _, ev := range resp.Events {
				key := strings.TrimPrefix(string(ev.Kv.Key), watchKey)
				if ev.Type == clientv3.EventTypeDelete {
					onChange(key, nil, ev.Kv.ModRevision)
				} else {
					onChange(key, ev.Kv.Value, ev.Kv.ModRevision)
				}
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
