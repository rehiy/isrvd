package model

// PatchRouteStatusRequest 更新路由状态请求
type PatchRouteStatusRequest struct {
	Status int `json:"status"`
}

// CreateConsumerRequest 创建 Consumer 请求
type CreateConsumerRequest struct {
	Username string `json:"username" binding:"required"`
	Desc     string `json:"desc"`
}

// UpdateConsumerRequest 更新 Consumer 请求
type UpdateConsumerRequest struct {
	Desc string `json:"desc"`
}

// RevokeWhitelistRequest 移除白名单请求
type RevokeWhitelistRequest struct {
	RouteID      string `json:"route_id" binding:"required"`
	ConsumerName string `json:"consumer_name" binding:"required"`
}
