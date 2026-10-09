package config

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"reflect"
	"sort"
	"sync"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/rehiy/libgo/logman"

	"isrvd/pkgs/cstore"
)

// ReloadCh 配置变更通知通道，存储变更时触发服务重载
var ReloadCh = make(chan struct{}, 1)

var (
	store       cstore.Store
	storeKey    string
	document    *cstore.Document
	watchCancel context.CancelFunc
	watchWG     sync.WaitGroup
	watchMu     sync.Mutex
	watching    bool
	localWrites = map[[sha256.Size]byte]int{}
)

func Init() error {
	uri := EnvOrDefault("CONFIG_PATH", "config.yml")

	var err error
	// etcd 认证：ETCD_USERNAME / ETCD_PASSWORD 优先于 URI userinfo
	store, storeKey, err = cstore.OpenWithKey(uri, cstore.WithEtcdCredentials(
		os.Getenv("ETCD_USERNAME"),
		os.Getenv("ETCD_PASSWORD"),
	))
	if err != nil {
		return err
	}

	document = cstore.NewDocument(store, storeKey)

	logman.Info("load config", "path", uri)
	if err := Load(); err != nil {
		return err
	}

	watchCtx, cancel := context.WithCancel(context.Background())
	watchCancel = cancel
	watchConfigChanges(watchCtx)
	return nil
}

// Close 停止配置监听并释放底层存储资源。
func Close() error {
	if watchCancel != nil {
		watchCancel()
		watchWG.Wait()
	}
	watchMu.Lock()
	watching = false
	clear(localWrites)
	watchMu.Unlock()
	if store != nil {
		return store.Close()
	}
	return nil
}

// OpenData 按当前数据目录派生业务数据存储，后端选择与迁移由 cstore 负责。
func OpenData[T any](file string) (*cstore.TypedStore[T], error) {
	dataStore, err := cstore.OpenData(store, Current().Server.RootDirectory)
	if err != nil {
		return nil, err
	}
	return cstore.NewTypedWith[T](dataStore, file), nil
}

// ReadStored 读取并验证持久化配置，但不发布运行时快照。
func ReadStored() (*Snapshot, error) {
	updateMu.Lock()
	defer updateMu.Unlock()
	return readSnapshot()
}

// Publish 发布已经完整校验的配置快照。
func Publish(snapshot *Snapshot) {
	if snapshot != nil {
		current.Store(snapshot)
	}
}

// Load 从 store 加载配置，完整构造后一次性发布。
func Load() error {
	snapshot, err := ReadStored()
	if err != nil {
		return err
	}
	Publish(snapshot)
	return nil
}

// Update 在配置副本上串行修改，持久化成功后发布新快照；可选回调用于同步运行态派生数据。
func Update(mutator func(*Snapshot) error, afterPublish ...func(*Snapshot)) error {
	updateMu.Lock()
	defer updateMu.Unlock()

	base := current.Load()
	stored, err := readSnapshot()
	if err != nil {
		return err
	}
	if stored != nil {
		if !reflect.DeepEqual(stored, base) {
			return fmt.Errorf("配置正在重载，请稍后重试")
		}
		base = stored
	}
	candidate, err := updateSnapshot(base, mutator)
	if err != nil {
		return err
	}
	current.Store(candidate)
	for _, apply := range afterPublish {
		if apply != nil {
			apply(candidate)
		}
	}
	return nil
}

// UpdateStored 更新持久化配置但不发布运行时快照，由服务重载统一生效。
func UpdateStored(mutator func(*Snapshot) error) error {
	updateMu.Lock()
	defer updateMu.Unlock()

	runtimeSnapshot := current.Load()
	base, err := readSnapshot()
	if err != nil {
		return err
	}
	if base == nil {
		base = runtimeSnapshot
	} else if !reflect.DeepEqual(base, runtimeSnapshot) {
		return fmt.Errorf("配置正在重载，请稍后重试")
	}
	_, err = updateSnapshot(base, mutator)
	return err
}

// ─── 辅助函数 ───

func readSnapshot() (*Snapshot, error) {
	conf := &Config{}
	raw, err := document.Read(func(data []byte) error {
		if data == nil {
			return nil
		}
		return yaml.Unmarshal(data, conf)
	})
	if err != nil {
		return nil, fmt.Errorf("读取配置失败: %w", err)
	}
	if raw == nil {
		logman.Warn("未找到配置文件", "key", storeKey)
		return nil, nil
	}

	migrated := migrate(conf, raw)
	snapshot, err := buildSnapshot(conf)
	if err != nil {
		return nil, fmt.Errorf("构建配置快照失败: %w", err)
	}
	if conf.Server == nil || conf.Server.JWTSecret != snapshot.Server.JWTSecret {
		if conf.Server == nil {
			conf.Server = &ServerConfig{}
		}
		conf.Server.JWTSecret = snapshot.Server.JWTSecret
		migrated = true
	}
	if migrated {
		data, marshalErr := yaml.Marshal(conf)
		if marshalErr != nil {
			return nil, fmt.Errorf("配置迁移序列化失败: %w", marshalErr)
		}
		if err := setConfigBytes(data); err != nil {
			return nil, fmt.Errorf("配置迁移保存失败: %w", err)
		}
		logman.Info("配置已自动更新（配置迁移）")
	}
	return snapshot, nil
}

func updateSnapshot(base *Snapshot, mutator func(*Snapshot) error) (*Snapshot, error) {
	if mutator == nil {
		return nil, fmt.Errorf("配置更新函数不能为空")
	}
	candidate, err := cloneSnapshot(base)
	if err != nil {
		return nil, fmt.Errorf("复制配置快照失败: %w", err)
	}
	if err := mutator(candidate); err != nil {
		return nil, err
	}
	candidate, err = cloneSnapshot(candidate)
	if err != nil {
		return nil, fmt.Errorf("隔离配置候选失败: %w", err)
	}
	if err := normalizeSnapshot(candidate); err != nil {
		return nil, err
	}
	if err := validateSnapshot(candidate); err != nil {
		return nil, err
	}
	if err := persistSnapshot(candidate); err != nil {
		return nil, err
	}
	return candidate, nil
}

func persistSnapshot(snapshot *Snapshot) error {
	members := make([]*MemberConfig, 0, len(snapshot.Members))
	for _, member := range snapshot.Members {
		if member != nil {
			members = append(members, member)
		}
	}
	sort.Slice(members, func(i, j int) bool {
		return members[i].Username < members[j].Username
	})

	conf := &Config{
		Schema:      schema,
		Server:      snapshot.Server,
		Password:    snapshot.Password,
		Passkey:     snapshot.Passkey,
		OIDC:        snapshot.OIDC,
		THA:         snapshot.THA,
		Copilot:     snapshot.Copilot,
		Notify:      snapshot.Notify,
		Apisix:      snapshot.Apisix,
		Caddy:       snapshot.Caddy,
		Docker:      snapshot.Docker,
		Monitor:     snapshot.Monitor,
		Marketplace: snapshot.Marketplace,
		Links:       snapshot.Links,
		Members:     members,
	}

	buf, err := yaml.Marshal(conf)
	if err != nil {
		return err
	}
	var persistent Config
	if err := yaml.Unmarshal(buf, &persistent); err != nil {
		return err
	}
	denormalizePaths(&persistent)
	data, err := yaml.Marshal(&persistent)
	if err != nil {
		return err
	}
	if err := setConfigBytes(data); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}
	return nil
}

func setConfigBytes(data []byte) error {
	fingerprint := sha256.Sum256(data)
	watchMu.Lock()
	tracked := watching
	if tracked {
		localWrites[fingerprint]++
	}
	watchMu.Unlock()
	if err := document.Set(data); err != nil {
		forgetLocalWrite(fingerprint)
		return err
	}
	if tracked {
		time.AfterFunc(30*time.Second, func() { forgetLocalWrite(fingerprint) })
	}
	return nil
}

func forgetLocalWrite(fingerprint [sha256.Size]byte) bool {
	watchMu.Lock()
	defer watchMu.Unlock()
	count := localWrites[fingerprint]
	if count == 0 {
		return false
	}
	if count == 1 {
		delete(localWrites, fingerprint)
	} else {
		localWrites[fingerprint] = count - 1
	}
	return true
}

func watchConfigChanges(ctx context.Context) {
	ch := store.Watch(ctx, storeKey)
	if ch == nil {
		return
	}
	watchMu.Lock()
	watching = true
	watchMu.Unlock()
	watchWG.Add(1)
	go func() {
		defer watchWG.Done()
		for ev := range ch {
			switch ev.Type {
			case cstore.EventPut:
				if forgetLocalWrite(sha256.Sum256(ev.Value)) {
					continue
				}
				logman.Info("Config changed, triggering reload", "key", ev.Key)
				select {
				case ReloadCh <- struct{}{}:
				default:
				}
			case cstore.EventDelete:
				watchMu.Lock()
				clear(localWrites)
				watchMu.Unlock()
				logman.Warn("Config deleted", "key", ev.Key)
			}
		}
	}()
}

// EnvOrDefault 返回环境变量的值，未设置或为空时返回 fallback。
func EnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
