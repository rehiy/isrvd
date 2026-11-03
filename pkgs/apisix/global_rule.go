package apisix

import (
	"context"
	"net/http"
	"net/url"
)

// GlobalRule Apisix Global Rule 信息
type GlobalRule struct {
	ID         string         `json:"id,omitempty"`      // Global Rule ID
	Plugins    map[string]any `json:"plugins,omitempty"` // 对所有路由生效的插件配置
	CreateTime int64          `json:"create_time"`       // 创建时间（Unix 时间戳，只读）
	UpdateTime int64          `json:"update_time"`       // 更新时间（Unix 时间戳，只读）
}

// GlobalRuleList 获取所有 Global Rule
func (c *Client) GlobalRuleList(ctx context.Context) ([]GlobalRule, error) {
	return c.requestResourceList[GlobalRule](ctx, "/global_rules", "解析 Global Rule 列表失败")
}

// GlobalRuleSave 创建或覆盖指定 ID 的 Global Rule
func (c *Client) GlobalRuleSave(ctx context.Context, id string, plugins map[string]any) error {
	_, err := c.doRequest(ctx, http.MethodPut, "/global_rules/"+url.PathEscape(id), map[string]any{"plugins": plugins})
	return err
}

// GlobalRuleDelete 删除 Global Rule
func (c *Client) GlobalRuleDelete(ctx context.Context, id string) error {
	_, err := c.doRequest(ctx, http.MethodDelete, "/global_rules/"+url.PathEscape(id), nil)
	return err
}
