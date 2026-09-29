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

var processMetricSamples = struct {
	sync.Mutex
	values map[int32]processMetricSample
}{
	values: make(map[int32]processMetricSample),
}

type processMetricSample struct {
	createTime   int64
	cpuSeconds   float64
	readBytes    uint64
	writeBytes   uint64
	cpuCollected bool
	ioCollected  bool
	sampledAt    time.Time
}

// ProcessInfo 进程信息
type ProcessInfo struct {
	PID           int32    `json:"pid"`                  // 进程 ID
	PPID          int32    `json:"ppid"`                 // 父进程 ID
	Name          string   `json:"name"`                 // 进程名
	Username      string   `json:"username"`             // 运行用户
	Status        string   `json:"status"`               // 运行状态
	CPUMillis     uint64   `json:"cpuMillis"`            // 累计 CPU 时间（毫秒）
	CPUPercent    *float64 `json:"cpuPercent,omitempty"` // CPU 占用（%，相邻采样区间均值，可超过 100）
	MemoryPercent float32  `json:"memoryPercent"`        // 内存占用（%）
	MemoryRSS     uint64   `json:"memoryRss"`            // 常驻内存（字节）
	IOReadBPS     *uint64  `json:"ioReadBps,omitempty"`  // 磁盘读取速率（字节/秒），首次采样或不可用时省略
	IOWriteBPS    *uint64  `json:"ioWriteBps,omitempty"` // 磁盘写入速率（字节/秒），首次采样或不可用时省略
	CreateTime    int64    `json:"createTime"`           // 启动时间（Unix 毫秒）
	Cmdline       string   `json:"cmdline,omitempty"`    // 完整命令行，仅创始人可见
	cpuSeconds    float64  // 本轮累计 CPU 时间（秒），仅用于计算速率
	cpuCollected  bool     // 本轮是否成功采集 CPU 时间
	ioReadBytes   uint64   // 本轮累计读取字节，仅用于计算速率
	ioWriteBytes  uint64   // 本轮累计写入字节，仅用于计算速率
	ioCollected   bool     // 本轮是否成功采集 I/O 计数
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
	applyProcessRates(list, time.Now())

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
	if ioCounters, err := proc.IOCounters(); err == nil && ioCounters != nil {
		info.ioReadBytes = ioCounters.ReadBytes
		info.ioWriteBytes = ioCounters.WriteBytes
		info.ioCollected = true
	}
	if includeCmdline {
		if cmdline, err := proc.Cmdline(); err == nil {
			info.Cmdline = cmdline
		}
	}

	if times, err := proc.Times(); err == nil && times != nil {
		info.cpuSeconds = times.User + times.System
		info.CPUMillis = uint64(info.cpuSeconds * 1000)
		info.cpuCollected = true
	}

	return info
}

// applyProcessRates 用相邻两次采样计算 CPU 与 I/O 速率；PID 复用或计数回退时不返回速率。
func applyProcessRates(list []*ProcessInfo, now time.Time) {
	processMetricSamples.Lock()
	defer processMetricSamples.Unlock()

	next := make(map[int32]processMetricSample, len(list))
	for _, info := range list {
		if !info.cpuCollected && !info.ioCollected {
			continue
		}

		previous, sameProcess := processMetricSamples.values[info.PID]
		sameProcess = sameProcess && info.CreateTime > 0 && previous.createTime == info.CreateTime
		if sameProcess && info.cpuCollected && previous.cpuCollected {
			if cpuPercent, ok := cpuRate(previous.cpuSeconds, info.cpuSeconds, now.Sub(previous.sampledAt)); ok {
				info.CPUPercent = &cpuPercent
			}
		}
		if sameProcess && info.ioCollected && previous.ioCollected {
			if readBPS, ok := ioRate(previous.readBytes, info.ioReadBytes, now.Sub(previous.sampledAt)); ok {
				info.IOReadBPS = &readBPS
			}
			if writeBPS, ok := ioRate(previous.writeBytes, info.ioWriteBytes, now.Sub(previous.sampledAt)); ok {
				info.IOWriteBPS = &writeBPS
			}
		}

		next[info.PID] = processMetricSample{
			createTime:   info.CreateTime,
			cpuSeconds:   info.cpuSeconds,
			readBytes:    info.ioReadBytes,
			writeBytes:   info.ioWriteBytes,
			cpuCollected: info.cpuCollected,
			ioCollected:  info.ioCollected,
			sampledAt:    now,
		}
	}
	processMetricSamples.values = next
}

func cpuRate(previous, current float64, elapsed time.Duration) (float64, bool) {
	if current < previous || elapsed <= 0 {
		return 0, false
	}
	return (current - previous) / elapsed.Seconds() * 100, true
}

func ioRate(previous, current uint64, elapsed time.Duration) (uint64, bool) {
	if current < previous || elapsed <= 0 {
		return 0, false
	}
	return uint64(float64(current-previous) / elapsed.Seconds()), true
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
