package monitor

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/docker/docker/api/types/container"

	"isrvd/pkgs/docker"
)

// textRow 一行样本：labels 为 名称/取值 交替的键值对
type textRow struct {
	labels []string
	value  float64
}

// textWriter 按指标族分组输出指标文本格式（text exposition format）
type textWriter struct{ b strings.Builder }

func (w *textWriter) write(typ, name, help string, rows ...textRow) {
	if len(rows) == 0 {
		return
	}
	fmt.Fprintf(&w.b, "# HELP %s %s\n# TYPE %s %s\n", name, help, name, typ)
	for _, r := range rows {
		w.b.WriteString(name)
		for i := 0; i+1 < len(r.labels); i += 2 {
			sep := ","
			if i == 0 {
				sep = "{"
			}
			fmt.Fprintf(&w.b, `%s%s="%s"`, sep, r.labels[i], textEscaper.Replace(r.labels[i+1]))
		}
		if len(r.labels) > 0 {
			w.b.WriteByte('}')
		}
		w.b.WriteString(" " + strconv.FormatFloat(r.value, 'g', -1, 64) + "\n")
	}
}

var textEscaper = strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`)

// MetricsText 把主机监控数据渲染为指标文本格式。
// 累计量为 counter（_total 后缀），其余为 gauge。
func (h *HostStat) MetricsText() []byte {
	var w textWriter
	if h == nil {
		return nil
	}
	one := func(v float64) textRow { return textRow{value: v} }

	if sys := h.System; sys != nil {
		if s := sys.SummaryStat; s != nil {
			cpu := 0.0
			for _, v := range s.CpuPercent {
				cpu += v / float64(len(s.CpuPercent))
			}
			w.write("gauge", "isrvd_host_uptime_seconds", "Host uptime in seconds.", one(float64(s.Uptime)))
			w.write("gauge", "isrvd_host_cpu_usage_percent", "Average CPU usage in percent (0-100).", one(cpu))
			w.write("gauge", "isrvd_host_memory_total_bytes", "Total physical memory in bytes.", one(float64(s.MemoryTotal)))
			w.write("gauge", "isrvd_host_memory_used_bytes", "Used physical memory in bytes.", one(float64(s.MemoryUsed)))
		}
		w.write("gauge", "isrvd_host_swap_total_bytes", "Total swap in bytes.", one(float64(sys.SwapTotal)))
		w.write("gauge", "isrvd_host_swap_used_bytes", "Used swap in bytes.", one(float64(sys.SwapUsed)))

		var size, used, rx, tx []textRow
		for _, p := range sys.DiskPartition {
			l := []string{"device", p.Device, "mountpoint", p.Mountpoint, "fstype", p.Fstype}
			size = append(size, textRow{l, float64(p.Total)})
			used = append(used, textRow{l, float64(p.Used)})
		}
		for _, n := range sys.NetInterface {
			l := []string{"interface", n.Name}
			rx = append(rx, textRow{l, float64(n.BytesRecv)})
			tx = append(tx, textRow{l, float64(n.BytesSent)})
		}
		w.write("gauge", "isrvd_host_filesystem_size_bytes", "Filesystem size in bytes.", size...)
		w.write("gauge", "isrvd_host_filesystem_used_bytes", "Filesystem used space in bytes.", used...)
		w.write("counter", "isrvd_host_network_receive_bytes_total", "Bytes received by the network interface.", rx...)
		w.write("counter", "isrvd_host_network_transmit_bytes_total", "Bytes sent by the network interface.", tx...)
	}

	var rd, wr []textRow
	for _, d := range h.DiskIO {
		if d != nil {
			l := []string{"device", d.Name}
			rd = append(rd, textRow{l, float64(d.ReadBytes)})
			wr = append(wr, textRow{l, float64(d.WriteBytes)})
		}
	}
	w.write("counter", "isrvd_host_disk_read_bytes_total", "Bytes read from the disk.", rd...)
	w.write("counter", "isrvd_host_disk_written_bytes_total", "Bytes written to the disk.", wr...)

	var util, memUsed, memTotal, temp []textRow
	for _, g := range h.GPU {
		if g != nil {
			l := []string{"gpu", g.DeviceKey, "name", g.Name}
			util = append(util, textRow{l, g.Utilization})
			memUsed = append(memUsed, textRow{l, float64(g.MemoryUsed)})
			memTotal = append(memTotal, textRow{l, float64(g.MemoryTotal)})
			temp = append(temp, textRow{l, float64(g.Temperature)})
		}
	}
	w.write("gauge", "isrvd_host_gpu_utilization_percent", "GPU utilization in percent (0-100).", util...)
	w.write("gauge", "isrvd_host_gpu_memory_used_bytes", "GPU memory in use in bytes.", memUsed...)
	w.write("gauge", "isrvd_host_gpu_memory_total_bytes", "Total GPU memory in bytes.", memTotal...)
	w.write("gauge", "isrvd_host_gpu_temperature_celsius", "GPU temperature in degrees Celsius.", temp...)

	if g := h.Go; g != nil {
		w.write("gauge", "isrvd_go_goroutines", "Number of goroutines of the isrvd process.", one(float64(g.NumGoroutine)))
		if g.GoMemoryStat != nil {
			w.write("gauge", "isrvd_go_memory_alloc_bytes", "Bytes of allocated heap objects in the isrvd process.", one(float64(g.Alloc)))
		}
	}
	return []byte(w.b.String())
}

// ErrDockerUnavailable Docker 不可用，无法采集容器指标
var ErrDockerUnavailable = errors.New("Docker 服务不可用")

const containerStatsConcurrency = 8 // 同时采集的容器数；单个容器采样约需 1 秒

// ContainersMetricsText 采集全部运行中容器的资源指标并渲染为指标文本格式。
// 单个容器采集失败时跳过；非空容器列表全部采集失败时返回错误。
func (c *Collector) ContainersMetricsText(ctx context.Context) ([]byte, error) {
	if c == nil || c.docker == nil {
		return nil, ErrDockerUnavailable
	}
	list, err := c.docker.ContainerList(ctx, false)
	if err != nil {
		return nil, err
	}

	results := make([]struct {
		stats container.StatsResponse
		err   error
	}, len(list))
	sem := make(chan struct{}, containerStatsConcurrency)
	var wg sync.WaitGroup
	for i, ct := range list {
		select {
		case sem <- struct{}{}:
			wg.Add(1)
			go func() {
				defer func() { <-sem; wg.Done() }()
				results[i].stats, results[i].err = c.docker.ContainerStatsOnce(ctx, ct.ID)
			}()
		case <-ctx.Done():
		}
		if ctx.Err() != nil {
			break
		}
	}
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var w textWriter
	var firstErr error
	var info, cpu, memUsed, memLimit, rx, tx, rd, wr, pids []textRow
	for i, ct := range list {
		if err := results[i].err; err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		v := &results[i].stats
		name := ""
		if len(ct.Names) > 0 {
			name = strings.TrimPrefix(ct.Names[0], "/")
		}
		l := []string{"name", name}
		info = append(info, textRow{[]string{"name", name, "id", docker.ShortID(ct.ID), "image", ct.Image}, 1})
		cpu = append(cpu, textRow{l, float64(v.CPUStats.CPUUsage.TotalUsage) / 1e9})

		cache, cgroupV1 := v.MemoryStats.Stats["total_inactive_file"]
		if !cgroupV1 {
			cache = v.MemoryStats.Stats["inactive_file"]
		}
		working := v.MemoryStats.Usage
		if cache < working {
			working -= cache
		}
		memUsed = append(memUsed, textRow{l, float64(working)})
		memLimit = append(memLimit, textRow{l, float64(v.MemoryStats.Limit)})

		var rxBytes, txBytes, readBytes, writeBytes uint64
		for _, n := range v.Networks {
			rxBytes += n.RxBytes
			txBytes += n.TxBytes
		}
		for _, b := range v.BlkioStats.IoServiceBytesRecursive {
			switch strings.ToLower(b.Op) {
			case "read":
				readBytes += b.Value
			case "write":
				writeBytes += b.Value
			}
		}
		rx = append(rx, textRow{l, float64(rxBytes)})
		tx = append(tx, textRow{l, float64(txBytes)})
		rd = append(rd, textRow{l, float64(readBytes)})
		wr = append(wr, textRow{l, float64(writeBytes)})
		pids = append(pids, textRow{l, float64(v.PidsStats.Current)})
	}
	if len(info) == 0 && firstErr != nil {
		return nil, fmt.Errorf("采集容器指标失败: %w", firstErr)
	}
	w.write("gauge", "isrvd_container_info", "Running container information; the value is always 1.", info...)
	w.write("counter", "isrvd_container_cpu_usage_seconds_total", "Cumulative CPU time consumed by the container in seconds.", cpu...)
	w.write("gauge", "isrvd_container_memory_usage_bytes", "Container memory working set in bytes (usage minus inactive file cache).", memUsed...)
	w.write("gauge", "isrvd_container_memory_limit_bytes", "Container memory limit in bytes.", memLimit...)
	w.write("counter", "isrvd_container_network_receive_bytes_total", "Bytes received by the container across all interfaces.", rx...)
	w.write("counter", "isrvd_container_network_transmit_bytes_total", "Bytes sent by the container across all interfaces.", tx...)
	w.write("counter", "isrvd_container_disk_read_bytes_total", "Bytes read from block devices by the container.", rd...)
	w.write("counter", "isrvd_container_disk_written_bytes_total", "Bytes written to block devices by the container.", wr...)
	w.write("gauge", "isrvd_container_pids", "Number of processes in the container.", pids...)
	return []byte(w.b.String()), nil
}
