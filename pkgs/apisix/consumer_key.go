package apisix

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	ErrConsumerNotFound        = errors.New("消费者不存在")
	ErrConsumerKeyMissing      = errors.New("消费者未配置 key-auth Token")
	ErrConsumerKeyInvalid      = errors.New("消费者账号或 Token 参数无效")
	ErrConsumerResponseInvalid = errors.New("消费者响应无效")
)

// ConsumerKey 消费者完整 key-auth Token 信息（仅含已授权的 key-auth 凭据，不暴露其他插件密钥）
type ConsumerKey struct {
	Username   string `json:"username"`    // Consumer 用户名
	Key        string `json:"key"`         // 完整未脱敏的 key-auth Token
	Desc       string `json:"desc"`        // Consumer 描述
	CreateTime int64  `json:"create_time"` // 创建时间（Unix 时间戳，只读）
	UpdateTime int64  `json:"update_time"` // 更新时间（Unix 时间戳，只读）
}

// ConsumerKeyInspect 读取指定 Consumer 的完整 key-auth Token
func (c *Client) ConsumerKeyInspect(ctx context.Context, username string) (*ConsumerKey, error) {
	value, err := c.consumerKeyValue(ctx, username)
	if err != nil {
		return nil, err
	}
	return consumerKey(value)
}

// ConsumerKeyUpdate 签发或更新 Consumer 的 key-auth Token，保留其他 Consumer 字段与非 key-auth 插件配置。
// 调用方负责在同一服务实例内与其他 Consumer 写操作按账号串行化；APISIX PUT 本身为 upsert，不覆盖外部直接写入。
func (c *Client) ConsumerKeyUpdate(ctx context.Context, username, key string, desc *string, createOnly bool) (*ConsumerKey, error) {
	if !ValidConsumerName(username) || !validConsumerKey(key) || !validConsumerDesc(desc) {
		return nil, ErrConsumerKeyInvalid
	}
	value, err := c.consumerKeyValue(ctx, username)
	if err != nil && !errors.Is(err, ErrConsumerNotFound) {
		return nil, err
	}
	if err == nil && createOnly {
		// 已有账号若无启用状态的 key-auth 视为冲突（409），绝不隐式添加或重置认证方式。
		return consumerKey(value)
	}
	if value == nil {
		value = map[string]any{"username": username}
	}
	if rawDesc := value["desc"]; rawDesc != nil {
		if _, ok := rawDesc.(string); !ok {
			return nil, ErrConsumerResponseInvalid
		}
	}
	plugins, ok := value["plugins"].(map[string]any)
	if !ok {
		if value["plugins"] != nil {
			return nil, ErrConsumerResponseInvalid
		}
		plugins = map[string]any{}
		value["plugins"] = plugins
	}
	auth, ok := plugins["key-auth"].(map[string]any)
	if !ok {
		if plugins["key-auth"] != nil {
			return nil, ErrConsumerResponseInvalid
		}
		auth = map[string]any{}
		plugins["key-auth"] = auth
	}
	disabled, err := pluginDisabled(auth)
	if err != nil {
		return nil, err
	}
	if disabled {
		return nil, ErrConsumerKeyMissing
	}
	auth["key"] = key
	if desc != nil {
		value["desc"] = *desc
	}
	// 移除只读时间戳，交由 APISIX 保留原 create_time 并自动刷新 update_time。
	delete(value, "create_time")
	delete(value, "update_time")
	if _, err := c.doRequest(ctx, http.MethodPut, "/consumers/"+url.PathEscape(username), value); err != nil {
		return nil, err
	}
	// 回读上游实际保存的密钥，兼容写成功但响应丢失后的重试语义。
	return c.ConsumerKeyInspect(ctx, username)
}

// ValidConsumerName 校验 Consumer 用户名合法性
func ValidConsumerName(username string) bool {
	return username != "" && len(username) <= 100 && utf8.ValidString(username) &&
		username != "." && username != ".." &&
		!strings.ContainsAny(username, "/\\%?#") &&
		!strings.ContainsFunc(username, func(r rune) bool { return unicode.IsControl(r) || unicode.IsSpace(r) })
}

// ─── 辅助函数 ───

func (c *Client) consumerKeyValue(ctx context.Context, username string) (map[string]any, error) {
	if !ValidConsumerName(username) {
		return nil, ErrConsumerKeyInvalid
	}
	data, err := c.doRequest(ctx, http.MethodGet, "/consumers/"+url.PathEscape(username), nil)
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			return nil, ErrConsumerNotFound
		}
		return nil, err
	}
	var raw struct {
		Value map[string]any `json:"value"`
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(&raw); err != nil {
		return nil, fmt.Errorf("解析 Consumer 失败: %w", err)
	}
	if err := dec.Decode(new(any)); !errors.Is(err, io.EOF) {
		return nil, ErrConsumerResponseInvalid
	}
	if raw.Value == nil || raw.Value["username"] != username {
		return nil, ErrConsumerResponseInvalid
	}
	return raw.Value, nil
}

func consumerKey(value map[string]any) (*ConsumerKey, error) {
	username, ok := value["username"].(string)
	if !ok || username == "" {
		return nil, ErrConsumerResponseInvalid
	}
	desc := ""
	if rawDesc := value["desc"]; rawDesc != nil {
		desc, ok = rawDesc.(string)
		if !ok {
			return nil, ErrConsumerResponseInvalid
		}
	}
	var plugins map[string]any
	if rawPlugins := value["plugins"]; rawPlugins != nil {
		plugins, ok = rawPlugins.(map[string]any)
		if !ok {
			return nil, ErrConsumerResponseInvalid
		}
	}
	var auth map[string]any
	if rawAuth := plugins["key-auth"]; rawAuth != nil {
		auth, ok = rawAuth.(map[string]any)
		if !ok {
			return nil, ErrConsumerResponseInvalid
		}
	}
	disabled, err := pluginDisabled(auth)
	if err != nil {
		return nil, err
	}
	if disabled {
		return nil, ErrConsumerKeyMissing
	}
	key := ""
	if rawKey := auth["key"]; rawKey != nil {
		key, ok = rawKey.(string)
		if !ok {
			return nil, ErrConsumerResponseInvalid
		}
	}
	if key == "" {
		return nil, ErrConsumerKeyMissing
	}
	if strings.Contains(key, "******") {
		return nil, ErrConsumerResponseInvalid
	}
	created, err := jsonTimestamp(value["create_time"])
	if err != nil {
		return nil, err
	}
	updated, err := jsonTimestamp(value["update_time"])
	if err != nil {
		return nil, err
	}
	return &ConsumerKey{
		Username:   username,
		Key:        key,
		Desc:       desc,
		CreateTime: created,
		UpdateTime: updated,
	}, nil
}

func pluginDisabled(plugin map[string]any) (bool, error) {
	if plugin == nil {
		return false, nil
	}
	rawMeta, exists := plugin["_meta"]
	if !exists || rawMeta == nil {
		return false, nil
	}
	meta, ok := rawMeta.(map[string]any)
	if !ok {
		return false, ErrConsumerResponseInvalid
	}
	rawDisable, exists := meta["disable"]
	if !exists || rawDisable == nil {
		return false, nil
	}
	disabled, ok := rawDisable.(bool)
	if !ok {
		return false, ErrConsumerResponseInvalid
	}
	return disabled, nil
}

func jsonTimestamp(raw any) (int64, error) {
	switch v := raw.(type) {
	case nil:
		return 0, nil
	case json.Number:
		if n, err := v.Int64(); err == nil {
			return n, nil
		}
		f, err := v.Float64()
		if err != nil {
			return 0, ErrConsumerResponseInvalid
		}
		return int64(f), nil
	case float64:
		return int64(v), nil
	case int64:
		return v, nil
	default:
		return 0, ErrConsumerResponseInvalid
	}
}

func validConsumerKey(key string) bool {
	return key != "" && len(key) <= 4096 && utf8.ValidString(key) &&
		!strings.Contains(key, "******") &&
		!strings.ContainsFunc(key, func(r rune) bool { return unicode.IsControl(r) || unicode.IsSpace(r) })
}

func validConsumerDesc(desc *string) bool {
	if desc == nil {
		return true
	}
	return utf8.ValidString(*desc) && !strings.ContainsFunc(*desc, func(r rune) bool {
		return unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t'
	})
}
