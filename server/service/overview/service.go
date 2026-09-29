// Package overview 提供系统概览业务服务（统计 + 探测）。
package overview

// Service 概览业务服务
type Service struct{}

// NewService 创建概览业务服务
func NewService() *Service {
	return &Service{}
}
