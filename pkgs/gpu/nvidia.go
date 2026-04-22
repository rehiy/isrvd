package gpu

import (
	"context"
	"os/exec"
	"strconv"
	"strings"

	ghwgpu "github.com/jaypipes/ghw/pkg/gpu"
)

func collectNvidiaCards(ctx context.Context, cards []*ghwgpu.GraphicsCard) ([]*DeviceStat, error) {
	return collectNvidiaSMI(ctx, cards)
}

// nvidia-smi fallback
func collectNvidiaSMI(ctx context.Context, cards []*ghwgpu.GraphicsCard) ([]*DeviceStat, error) {
	path, err := exec.LookPath("nvidia-smi")
	if err != nil || path == "" {
		return nil, nil
	}

	cmdCtx, cancel := commandContext(ctx)
	defer cancel()

	out, err := exec.CommandContext(cmdCtx, path,
		"--query-gpu=index,pci.bus_id,name,memory.used,memory.total,utilization.gpu,temperature.gpu,power.draw,fan.speed",
		"--format=csv,noheader,nounits",
	).Output()
	if err != nil {
		return nil, err
	}

	records, err := readCSVRecords(out)
	if err != nil {
		return nil, err
	}

	allowed := addressSet(cards)
	var stats []*DeviceStat
	for _, fields := range records {
		if len(fields) < 9 {
			continue
		}

		busID := normalizePCIAddress(strings.TrimSpace(fields[1]))
		if len(allowed) > 0 && !allowed[busID] {
			continue
		}

		name := strings.TrimSpace(fields[2])
		memUsedMB, _ := strconv.ParseFloat(strings.TrimSpace(fields[3]), 64)
		memTotalMB, _ := strconv.ParseFloat(strings.TrimSpace(fields[4]), 64)
		util, _ := strconv.ParseFloat(strings.TrimSpace(fields[5]), 64)

		stats = append(stats, &DeviceStat{
			Address:     busID,
			Name:        name,
			Vendor:      "nvidia",
			MemoryUsed:  uint64(memUsedMB * 1024 * 1024),
			MemoryTotal: uint64(memTotalMB * 1024 * 1024),
			Utilization: util,
			Temperature: parseIntOrDefault(strings.TrimSpace(fields[6]), -1),
			PowerUsage:  parseFloatOrDefault(strings.TrimSpace(fields[7]), -1),
			FanSpeed:    parseIntOrDefault(strings.TrimSpace(fields[8]), -1),
		})
	}
	return stats, nil
}
