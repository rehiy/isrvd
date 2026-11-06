package monitor

import (
	"os"
	"path/filepath"
	"strings"
)

// ─── 辅助函数 ───

// 非物理存储的块设备名前缀（虚拟、只读镜像、可移动介质）
// 注意：dm-/md 等逻辑卷、软 RAID 承载真实数据，不在此列
var virtualBlockPrefixes = []string{
	"loop", // loop 设备（snap、镜像挂载）
	"ram",  // 内存盘
	"zram", // 压缩内存盘
	"sr",   // 光驱
	"fd",   // 软驱
}

// isVirtualBlock 判断设备（/dev/xxx 或 xxx）是否为非物理存储块设备
func isVirtualBlock(device string) bool {
	name := filepath.Base(device)
	for _, prefix := range virtualBlockPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// isMountExcluded 判断挂载是否不应作为存储展示。
// 虚拟文件系统（tmpfs/overlay/proc/cgroup/fuse/nfs 等 nodev 类型）已由
// psutil 依据内核 /proc/filesystems 过滤，这里只处理其余情形：
//  1. 底层是虚拟块设备（loop/ram/sr 等）
//  2. 挂载点不是目录（如容器内单文件 bind mount：/etc/hosts、/etc/resolv.conf）
func isMountExcluded(device, mountpoint string) bool {
	if isVirtualBlock(device) {
		return true
	}
	if fi, err := os.Stat(mountpoint); err != nil || !fi.IsDir() {
		return true
	}
	return false
}
