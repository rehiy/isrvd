package node

// PermissionRoute 描述中控节点管理接口及其可分配权限。
type PermissionRoute struct {
	Method string
	Path   string
	Label  string
}

// PermissionRoutes 返回节点管理路由，路径使用与账户权限一致的 :param 格式。
// 受管机隧道入口使用节点令牌认证，不参与成员权限分配。
func PermissionRoutes() []PermissionRoute {
	return []PermissionRoute{
		{"GET", "/api/node/nodes", "查询节点列表"},
		{"POST", "/api/node", "创建节点"},
		{"PUT", "/api/node/:id", "更新节点"},
		{"POST", "/api/node/:id/revoke", "吊销节点"},
		{"DELETE", "/api/node/:id", "删除节点"},
		{"PUT", "/api/node/:id/portmap", "配置节点端口转发"},
	}
}
