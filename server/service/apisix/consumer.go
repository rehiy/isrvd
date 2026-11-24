package apisix

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"slices"
	"strings"
	"sync"

	"isrvd/pkgs/apisix"
)

// ConsumerList 获取 Consumer 列表
func (s *Service) ConsumerList(ctx context.Context) ([]apisix.Consumer, error) {
	list, err := s.client.ConsumerList(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取消费者列表失败: %w", err)
	}
	return list, nil
}

// ConsumerCreate 创建 Consumer，支持传入完整 plugins
func (s *Service) ConsumerCreate(ctx context.Context, username, desc string, plugins map[string]any) (*apisix.Consumer, error) {
	if username == "" {
		return nil, fmt.Errorf("用户名不能为空")
	}
	unlock := s.lockConsumer(username)
	defer unlock()
	consumer, err := s.client.ConsumerCreate(ctx, username, desc, plugins)
	if err != nil {
		return nil, fmt.Errorf("创建消费者失败: %w", err)
	}
	return consumer, nil
}

// ConsumerUpdate 更新 Consumer（支持 plugins，自动替换脱敏值）
func (s *Service) ConsumerUpdate(ctx context.Context, username, desc string, plugins map[string]any) error {
	if username == "" {
		return fmt.Errorf("用户名不能为空")
	}
	unlock := s.lockConsumer(username)
	defer unlock()
	if err := s.client.ConsumerUpdate(ctx, username, desc, plugins); err != nil {
		return fmt.Errorf("更新消费者失败: %w", err)
	}
	return nil
}

// ConsumerDelete 删除 Consumer；仍被路由访问授权引用时拒绝删除，
// 避免白名单中残留用户名，日后同名 Consumer 被重建时自动获得访问权限
func (s *Service) ConsumerDelete(ctx context.Context, username string) error {
	if username == "" {
		return fmt.Errorf("用户名不能为空")
	}
	unlock := s.lockConsumer(username)
	defer unlock()
	routes, err := s.client.RouteWhitelistInspect(ctx)
	if err != nil {
		return fmt.Errorf("获取访问授权失败: %w", err)
	}
	var refs []string
	for _, r := range routes {
		if slices.Contains(r.Consumers, username) {
			refs = append(refs, cmp.Or(r.Name, r.ID))
		}
	}
	if len(refs) > 0 {
		return fmt.Errorf("消费者 %s 仍被路由授权引用（%s）", username, strings.Join(refs, ", "))
	}
	if err := s.client.ConsumerDelete(ctx, username); err != nil {
		return fmt.Errorf("删除消费者失败: %w", err)
	}
	return nil
}

// ConsumerKeyInspect 读取指定 Consumer 的完整 key-auth Token
func (s *Service) ConsumerKeyInspect(ctx context.Context, username string) (*apisix.ConsumerKey, error) {
	if !apisix.ValidConsumerName(username) {
		return nil, apisix.ErrConsumerKeyInvalid
	}
	unlock := s.rlockConsumer(username)
	defer unlock()
	return s.client.ConsumerKeyInspect(ctx, username)
}

// ConsumerKeyUpdateRequest 签发或更新 Consumer key-auth Token 请求（与通用插件编辑权限隔离）
type ConsumerKeyUpdateRequest struct {
	Key        string  `json:"key"`         // 待签发的完整 key-auth Token
	Desc       *string `json:"desc"`        // 不传或 null 时保留已有描述，空字符串表示清空
	CreateOnly bool    `json:"create_only"` // true 时只新建，已存在则返回原 Token
}

// UnmarshalJSON 拒绝非对象请求体与未知字段，避免将拼写错误的 createOnly 静默当作覆盖更新执行。
func (r *ConsumerKeyUpdateRequest) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return errors.New("invalid json object")
	}
	type fields ConsumerKeyUpdateRequest
	dec := json.NewDecoder(bytes.NewReader(trimmed))
	dec.DisallowUnknownFields()
	if err := dec.Decode((*fields)(r)); err != nil {
		return err
	}
	if err := dec.Decode(new(any)); !errors.Is(err, io.EOF) {
		return errors.New("trailing json data")
	}
	return nil
}

// ConsumerKeyUpdate 签发或更新指定 Consumer 的 key-auth Token
func (s *Service) ConsumerKeyUpdate(ctx context.Context, username string, req ConsumerKeyUpdateRequest) (*apisix.ConsumerKey, error) {
	if !apisix.ValidConsumerName(username) {
		return nil, apisix.ErrConsumerKeyInvalid
	}
	unlock := s.lockConsumer(username)
	defer unlock()
	return s.client.ConsumerKeyUpdate(ctx, username, req.Key, req.Desc, req.CreateOnly)
}

// lockConsumer 对同一服务实例内的 Consumer 写操作（含联合白名单创建）按账号加写锁。
func (s *Service) lockConsumer(username string) func() {
	lock := s.consumerMutex(username)
	lock.Lock()
	return lock.Unlock
}

// rlockConsumer 对 Consumer 读取加读锁，避免读到并发写过程中的中间态，同时不阻塞并发读取。
func (s *Service) rlockConsumer(username string) func() {
	lock := s.consumerMutex(username)
	lock.RLock()
	return lock.RUnlock
}

func (s *Service) consumerMutex(username string) *sync.RWMutex {
	h := fnv.New32a()
	_, _ = h.Write([]byte(username))
	return &s.consumerLocks[h.Sum32()%uint32(len(s.consumerLocks))]
}
