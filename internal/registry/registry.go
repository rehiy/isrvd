// Package registry 提供服务注册和管理功能
package registry

func Init() {
	initApisix()
	initDocker()
}
