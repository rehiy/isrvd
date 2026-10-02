package compose

import (
	"context"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rehiy/libgo/logman"
	"github.com/rehiy/libgo/secure"
	"github.com/rehiy/libgo/strutil"

	"isrvd/pkgs/compose"
	"isrvd/pkgs/cstore"
	"isrvd/server/config"
)

const (
	historyLimit         = 10
	historySnapshotBytes = 512 << 10
	historyStorageBytes  = 1 << 20
)

// HistoryRecord 的配置快照只加密落盘，不随列表返回。
type HistoryRecord struct {
	ID          string    `json:"id" yaml:"id"`
	Time        time.Time `json:"time" yaml:"time"`
	Action      string    `json:"action" yaml:"action"`
	Success     bool      `json:"success" yaml:"success"`
	Snapshot    string    `json:"-" yaml:"snapshot,omitempty"`
	HasSnapshot bool      `json:"hasSnapshot" yaml:"hasSnapshot"`
}

func (s *Service) HistoryList(ctx context.Context, target, name string) ([]HistoryRecord, error) {
	name, err := s.historyProjectName(ctx, target, name)
	if err != nil {
		return nil, err
	}
	records, err := s.historyList(target, name)
	for i := range records {
		records[i].Snapshot = ""
	}
	return records, err
}

func (s *Service) historyList(target, name string) ([]HistoryRecord, error) {
	s.historyMu.Lock()
	defer s.historyMu.Unlock()
	_, records, err := s.historyStore(target, name)
	if records == nil {
		records = []HistoryRecord{}
	}
	return records, err
}

func (s *Service) HistoryInspect(ctx context.Context, target, name, id string) (*ConfigDetail, error) {
	name, err := s.historyProjectName(ctx, target, name)
	if err != nil {
		return nil, err
	}
	records, err := s.historyList(target, name)
	if err != nil {
		return nil, err
	}
	for _, record := range records {
		if record.ID != id || record.Snapshot == "" {
			continue
		}
		return historyOpen(target, name, record)
	}
	return nil, fmt.Errorf("历史配置不存在或无可用快照")
}

// 先保存当前配置，再允许破坏性重部署；记录写入失败时保留现有实例。
func (s *Service) historyBegin(target, name string, current *ConfigDetail) error {
	if current == nil || current.Content == "" {
		return fmt.Errorf("无法保存重部署前的配置快照")
	}
	return s.historyAppend(target, name, "snapshot", true, current)
}

func (s *Service) historyFinish(ctx context.Context, target, name, action string, success bool) {
	var detail *ConfigDetail
	var err error
	if success {
		if target == "docker" {
			detail, err = s.DockerInspect(ctx, name, false)
		} else {
			detail, err = s.SwarmInspect(ctx, name, false)
		}
	}
	if err != nil {
		logman.Warn("Compose history snapshot failed", "name", name, "error", err)
		detail = nil
	}
	if err := s.historyAppend(target, name, action, success, detail); err != nil {
		logman.Warn("Compose history save failed", "name", name, "error", err)
		if detail != nil {
			if err := s.historyAppend(target, name, action, success, nil); err != nil {
				logman.Warn("Compose history metadata save failed", "name", name, "error", err)
			}
		}
	}
}

func (s *Service) historyAppend(target, name, action string, success bool, detail *ConfigDetail) error {
	s.historyMu.Lock()
	defer s.historyMu.Unlock()
	store, records, err := s.historyStore(target, name)
	if err != nil {
		return err
	}
	if action == "snapshot" && len(records) > 0 && detail != nil {
		previous, err := historyOpen(target, name, records[0])
		if err == nil && previous.Content == detail.Content && previous.EnvContent == detail.EnvContent {
			return nil
		}
	}
	record := HistoryRecord{ID: strutil.NewString(), Time: time.Now(), Action: action, Success: success}
	if detail != nil {
		plain, err := json.Marshal(detail)
		if err != nil {
			return err
		}
		if len(plain) > historySnapshotBytes {
			return fmt.Errorf("历史配置快照不能超过 512 KiB")
		}
		aead, err := historyAEAD()
		if err != nil {
			return err
		}
		raw, err := secure.AEADSeal(aead, plain, []byte(historyKey(target, name)+record.ID))
		if err != nil {
			return err
		}
		record.Snapshot = base64.RawStdEncoding.EncodeToString(raw)
		record.HasSnapshot = true
	}
	records = append([]HistoryRecord{record}, records...)
	bytes := 0
	for i, record := range records {
		bytes += len(record.Snapshot) + 512
		if i >= historyLimit || bytes > historyStorageBytes {
			records = records[:i]
			break
		}
	}
	return s.historySave(store, target, name, records)
}

func (s *Service) historyProjectName(ctx context.Context, target, name string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := compose.ValidateProjectName(name); err != nil {
		return "", err
	}
	if target != "docker" && target != "swarm" {
		return "", fmt.Errorf("未知部署目标")
	}
	if target == "docker" {
		name = s.dockerProjectName(ctx, name, s.docker.ContainerRoot())
	}
	return name, nil
}

// 调用方持有 historyMu；旧记录只在项目历史文件不存在时复制，不删除源数据。
func (s *Service) historyStore(target, name string) (*cstore.TypedStore[[]HistoryRecord], []HistoryRecord, error) {
	if err := compose.ValidateProjectName(name); err != nil {
		return nil, nil, err
	}
	if target != "docker" && target != "swarm" {
		return nil, nil, fmt.Errorf("未知部署目标")
	}
	if s.docker == nil || s.docker.ContainerRoot() == "" {
		return nil, nil, fmt.Errorf("未配置容器数据根目录")
	}
	dir := filepath.Join(s.docker.ContainerRoot(), name)
	file := ".isrvd-history-" + target + ".yml"
	local, err := historyFileStore(dir, file)
	if err != nil {
		return nil, nil, err
	}
	store := cstore.NewTypedWith[[]HistoryRecord](local, file)
	records, err := store.Get()
	if err != nil {
		return nil, nil, err
	}
	migrate := false
	if _, err := os.Lstat(filepath.Join(dir, file)); os.IsNotExist(err) {
		legacy, err := config.OpenData[[]HistoryRecord](historyKey(target, name))
		if err != nil {
			return nil, nil, err
		}
		records, err = legacy.Get()
		if err != nil {
			return nil, nil, err
		}
		migrate = len(records) > 0
	} else if err != nil {
		return nil, nil, err
	}
	for i := range records {
		if records[i].Snapshot != "" {
			migrate = true
		} else if records[i].HasSnapshot {
			_, stored, err := s.historyVersionStore(target, name, records[i].ID)
			if err != nil {
				logman.Warn("Compose history version unavailable", "name", name, "id", records[i].ID, "error", err)
			} else if stored == nil || stored.Snapshot == "" {
				logman.Warn("Compose history version missing or invalid", "name", name, "id", records[i].ID)
			} else {
				records[i].Snapshot = stored.Snapshot
			}
		}
		records[i].HasSnapshot = records[i].Snapshot != ""
	}
	if migrate {
		if err := s.historySave(store, target, name, records); err != nil {
			return nil, nil, err
		}
	}
	return store, records, nil
}

func (s *Service) historySave(store *cstore.TypedStore[[]HistoryRecord], target, name string, records []HistoryRecord) error {
	created := []string{}
	committed := false
	defer func() {
		if !committed {
			for _, path := range created {
				if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
					logman.Warn("Compose history version rollback failed", "name", name, "error", err)
				}
			}
		}
	}()
	metadata := append([]HistoryRecord{}, records...)
	for i, record := range records {
		if record.Snapshot != "" {
			version, stored, err := s.historyVersionStore(target, name, record.ID)
			if err != nil {
				return err
			}
			if stored == nil {
				if err := version.Set(&record); err != nil {
					return err
				}
				created = append(created, filepath.Join(s.docker.ContainerRoot(), name, historyVersionFile(target, record.ID)))
			} else if stored.Snapshot != record.Snapshot {
				return fmt.Errorf("历史版本文件已存在且内容不一致")
			}
		}
		metadata[i].HasSnapshot = record.Snapshot != ""
		metadata[i].Snapshot = ""
	}
	if err := store.Set(metadata); err != nil {
		return err
	}
	committed = true
	s.historyCleanup(target, name, records)
	return nil
}

// 仅在索引提交后清理本目标的 UUID 版本文件，不删除其他文件或符号链接。
func (s *Service) historyCleanup(target, name string, records []HistoryRecord) {
	dir := filepath.Join(s.docker.ContainerRoot(), name)
	entries, err := os.ReadDir(dir)
	if err != nil {
		logman.Warn("Compose history cleanup failed", "name", name, "error", err)
		return
	}
	retained := make(map[string]bool, len(records))
	for _, record := range records {
		retained[historyVersionFile(target, record.ID)] = true
	}
	for _, entry := range entries {
		if !entry.Type().IsRegular() || retained[entry.Name()] {
			continue
		}
		id, ok := strings.CutPrefix(entry.Name(), "compose."+target+"-")
		parsed, err := uuid.Parse(id)
		if !ok || err != nil || parsed.String() != id {
			continue
		}
		if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil && !os.IsNotExist(err) {
			logman.Warn("Compose history version cleanup failed", "name", name, "error", err)
		}
	}
}

func (s *Service) historyVersionStore(target, name, id string) (*cstore.TypedStore[*HistoryRecord], *HistoryRecord, error) {
	if err := compose.ValidateProjectName(id); err != nil {
		return nil, nil, err
	}
	dir := filepath.Join(s.docker.ContainerRoot(), name)
	file := historyVersionFile(target, id)
	local, err := historyFileStore(dir, file)
	if err != nil {
		return nil, nil, err
	}
	store := cstore.NewTypedWith[*HistoryRecord](local, file)
	record, err := store.Get()
	if err == nil && record != nil && record.ID != id {
		err = fmt.Errorf("历史版本文件 ID 不匹配")
	}
	return store, record, err
}

// ─── 辅助函数 ───

func historyFileStore(dir, file string) (cstore.Store, error) {
	for _, path := range []string{dir, filepath.Join(dir, file)} {
		info, err := os.Lstat(path)
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("历史配置路径不能为符号链接")
		}
	}
	return cstore.Open(dir)
}

func historyVersionFile(target, id string) string {
	return "compose." + target + "-" + id
}

func historyOpen(target, name string, record HistoryRecord) (*ConfigDetail, error) {
	aead, err := historyAEAD()
	if err != nil {
		return nil, err
	}
	raw, err := base64.RawStdEncoding.DecodeString(record.Snapshot)
	if err != nil {
		return nil, fmt.Errorf("历史配置格式错误")
	}
	plain, err := secure.AEADOpen(aead, raw, []byte(historyKey(target, name)+record.ID))
	if err != nil {
		return nil, fmt.Errorf("历史配置解密失败，JWT 密钥可能已变更")
	}
	var detail ConfigDetail
	if err := json.Unmarshal(plain, &detail); err != nil {
		return nil, fmt.Errorf("历史配置解析失败: %w", err)
	}
	return &detail, nil
}

func historyKey(target, name string) string {
	return fmt.Sprintf("compose-%x.yml", sha256.Sum256([]byte(target+":"+name)))
}

func historyAEAD() (cipher.AEAD, error) {
	key := sha256.Sum256([]byte("isrvd-compose:" + config.Current().Server.JWTSecret))
	return secure.NewAESGCM(key[:])
}
