package monitor

import "strings"

// 不应该显示的磁盘设备名前缀
var excludeDevicePrefixes = []string{
	"loop", // loop 设备
	"sr",   // 光驱设备
}

// 不应该显示的挂载点精确路径
var excludeMountExact = []string{
	"/etc/hosts",
	"/etc/resolv.conf",
	"/etc/hostname",
}

// 不应该显示的挂载点目录前缀（必须以 / 结尾）
var excludeMountPrefixes = []string{
	"/proc/",
	"/sys/",
	"/dev/pts/",
}

// 虚拟文件系统类型，不应该作为存储展示
var excludeFstypes = map[string]bool{
	"overlay":    true, // 容器层
	"tmpfs":      true, // 内存文件系统
	"devtmpfs":   true, // 设备文件系统
	"sysfs":      true,
	"proc":       true,
	"cgroup":     true,
	"cgroup2":    true,
	"pstore":     true,
	"securityfs": true,
	"debugfs":    true,
	"tracefs":    true,
	"hugetlbfs":  true,
	"mqueue":     true,
	"fusectl":    true,
	"bpf":        true,
}

// hasDevicePrefix 检查设备名是否匹配排除前缀
func hasDevicePrefix(name string) bool {
	for _, prefix := range excludeDevicePrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// isMountExcluded 检查挂载点是否应该被排除
func isMountExcluded(mountpoint, fstype string) bool {
	// 按文件系统类型过滤虚拟文件系统
	if excludeFstypes[fstype] {
		return true
	}
	// 精确路径匹配（用于 Docker 挂载的单个配置文件）
	for _, exact := range excludeMountExact {
		if mountpoint == exact {
			return true
		}
	}
	// 目录前缀匹配
	for _, prefix := range excludeMountPrefixes {
		if strings.HasPrefix(mountpoint, prefix) {
			return true
		}
	}
	return false
}
