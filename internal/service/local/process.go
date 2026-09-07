// Package local 提供本机管理能力（进程查看与终止等）。
package local

import (
	"errors"
	"os"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

// ErrProtectedProcess 受保护的进程不允许终止
var ErrProtectedProcess = errors.New("该进程受保护，不允许终止")

// ProcessInfo 进程信息
type ProcessInfo struct {
	PID           int32   `json:"pid"`               // 进程 ID
	PPID          int32   `json:"ppid"`              // 父进程 ID
	Name          string  `json:"name"`              // 进程名
	Username      string  `json:"username"`          // 运行用户
	Status        string  `json:"status"`            // 运行状态
	CPUPercent    float64 `json:"cpuPercent"`        // CPU 占用（%，自启动以来的均值，可超过 100）
	MemoryPercent float32 `json:"memoryPercent"`     // 内存占用（%）
	MemoryRSS     uint64  `json:"memoryRss"`         // 常驻内存（字节）
	CreateTime    int64   `json:"createTime"`        // 启动时间（Unix 毫秒）
	Cmdline       string  `json:"cmdline,omitempty"` // 完整命令行，仅创始人可见
}

// ProcessList 采集本机进程列表，按常驻内存降序排列。
// 完整命令行可能携带凭据，仅向创始人返回。
func ProcessList(includeCmdline bool) ([]*ProcessInfo, error) {
	procs, err := process.Processes()
	if err != nil {
		return nil, err
	}

	list := make([]*ProcessInfo, 0, len(procs))

	var (
		mu  sync.Mutex
		wg  sync.WaitGroup
		sem = make(chan struct{}, runtime.NumCPU()*4) // 限制并发，避免进程过多时耗尽文件描述符
	)

	for _, proc := range procs {
		wg.Add(1)
		go func(proc *process.Process) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			info := collect(proc, includeCmdline)
			mu.Lock()
			list = append(list, info)
			mu.Unlock()
		}(proc)
	}

	wg.Wait()

	sort.Slice(list, func(i, j int) bool {
		if list[i].MemoryRSS == list[j].MemoryRSS {
			return list[i].PID < list[j].PID
		}
		return list[i].MemoryRSS > list[j].MemoryRSS
	})

	return list, nil
}

// collect 采集单个进程的字段；任一字段失败时降级为零值，而不是丢弃整个进程
func collect(proc *process.Process, includeCmdline bool) *ProcessInfo {
	info := &ProcessInfo{PID: proc.Pid}

	if ppid, err := proc.Ppid(); err == nil {
		info.PPID = ppid
	}
	if name, err := proc.Name(); err == nil {
		info.Name = name
	}
	if username, err := proc.Username(); err == nil {
		info.Username = username
	}
	if status, err := proc.Status(); err == nil && len(status) > 0 {
		info.Status = status[0]
	}
	if ts, err := proc.CreateTime(); err == nil {
		info.CreateTime = ts
	}
	if percent, err := proc.MemoryPercent(); err == nil {
		info.MemoryPercent = percent
	}
	if mem, err := proc.MemoryInfo(); err == nil && mem != nil {
		info.MemoryRSS = mem.RSS
	}
	if includeCmdline {
		if cmdline, err := proc.Cmdline(); err == nil {
			info.Cmdline = cmdline
		}
	}

	// CPU 占用取自启动以来的均值（与 ps 的 %CPU 口径一致），
	// 只需一次采样，避免逐进程阻塞等待导致列表接口变慢。
	if times, err := proc.Times(); err == nil && times != nil && info.CreateTime > 0 {
		if elapsed := float64(time.Now().UnixMilli()-info.CreateTime) / 1000; elapsed > 0 {
			info.CPUPercent = (times.User + times.System) / elapsed * 100
		}
	}

	return info
}

// ProcessKill 终止进程；force 为 true 时发送 SIGKILL，否则发送 SIGTERM
func ProcessKill(pid int32, force bool) error {
	// 保护 init 进程与 isrvd 自身，避免系统崩溃或面板自杀
	if pid <= 1 || pid == int32(os.Getpid()) {
		return ErrProtectedProcess
	}

	proc, err := process.NewProcess(pid)
	if err != nil {
		return err
	}

	if force {
		return proc.Kill()
	}
	return proc.Terminate()
}
