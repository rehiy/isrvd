package apisix

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"unicode"
)

var (
	ErrConsumerNotFound        = errors.New("消费者不存在")
	ErrConsumerKeyMissing      = errors.New("消费者未配置 key-auth Token")
	ErrConsumerKeyInvalid      = errors.New("消费者账号或 Token 参数无效")
	ErrConsumerResponseInvalid = errors.New("消费者响应无效")
)

// ConsumerKey contains only the authorized key-auth credential, not other plugin secrets.
type ConsumerKey struct {
	Username   string `json:"username"`
	Key        string `json:"key"`
	Desc       string `json:"desc"`
	CreateTime int64  `json:"create_time"`
	UpdateTime int64  `json:"update_time"`
}

func (c *Client) ConsumerKeyInspect(ctx context.Context, username string) (*ConsumerKey, error) {
	value, err := c.consumerKeyValue(ctx, username)
	if err != nil {
		return nil, err
	}
	return consumerKey(value)
}

// ConsumerKeyUpdate preserves all Consumer fields and non-key-auth plugin settings.
// Callers serialize the read/write cycle with other Consumer writers. APISIX PUT
// is an upsert; direct Admin writers and other isrvd processes are not covered.
func (c *Client) ConsumerKeyUpdate(ctx context.Context, username, key string, desc *string, createOnly bool) (*ConsumerKey, error) {
	if !validConsumerName(username) || strings.TrimSpace(key) == "" || len(key) > 4096 ||
		strings.Contains(key, "******") || strings.ContainsFunc(key, unicode.IsControl) {
		return nil, ErrConsumerKeyInvalid
	}
	value, err := c.consumerKeyValue(ctx, username)
	if err != nil && !errors.Is(err, ErrConsumerNotFound) {
		return nil, err
	}
	if err == nil && createOnly {
		// An existing account without key-auth is a conflict, never implicit rotation.
		return consumerKey(value)
	}
	if value == nil {
		value = map[string]any{"username": username}
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
	auth["key"] = key
	if desc != nil {
		value["desc"] = *desc
	}
	if _, err := c.doRequest(ctx, http.MethodPut, "/consumers/"+url.PathEscape(username), value); err != nil {
		return nil, err
	}
	// Return the actual persisted key, including after an ambiguous retry.
	return c.ConsumerKeyInspect(ctx, username)
}

func (c *Client) consumerKeyValue(ctx context.Context, username string) (map[string]any, error) {
	if !validConsumerName(username) {
		return nil, ErrConsumerKeyInvalid
	}
	value, err := c.requestResource[map[string]any](ctx, http.MethodGet, "/consumers/"+url.PathEscape(username), nil, "解析 Consumer 失败")
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			return nil, ErrConsumerNotFound
		}
		return nil, err
	}
	if value == nil || (*value)["username"] != username {
		return nil, ErrConsumerResponseInvalid
	}
	return *value, nil
}

func consumerKey(value map[string]any) (*ConsumerKey, error) {
	plugins, _ := value["plugins"].(map[string]any)
	auth, _ := plugins["key-auth"].(map[string]any)
	key, _ := auth["key"].(string)
	if key == "" {
		return nil, ErrConsumerKeyMissing
	}
	if strings.Contains(key, "******") {
		return nil, ErrConsumerResponseInvalid
	}
	username, _ := value["username"].(string)
	desc, _ := value["desc"].(string)
	created, _ := value["create_time"].(float64)
	updated, _ := value["update_time"].(float64)
	return &ConsumerKey{Username: username, Key: key, Desc: desc, CreateTime: int64(created), UpdateTime: int64(updated)}, nil
}

func validConsumerName(username string) bool {
	return username != "" && len(username) <= 100 && username != "." && username != ".." &&
		!strings.ContainsAny(username, "/\\%?#") && !strings.ContainsFunc(username, func(r rune) bool { return unicode.IsControl(r) || unicode.IsSpace(r) })
}
