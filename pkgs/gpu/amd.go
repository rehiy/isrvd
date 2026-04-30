package gpu

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"

	ghwgpu "github.com/jaypipes/ghw/pkg/gpu"
)

func collectAMDCards(ctx context.Context, cards []*ghwgpu.GraphicsCard) ([]*DeviceStat, error) {
	if len(cards) > 0 {
		// 优先 sysfs（零依赖），fallback rocm-smi
		if stats := collectAMDSysfs(cards); len(stats) > 0 {
			return stats, nil
		}
	}
	return collectAMDRocmSMI(ctx, cards)
}

func collectAMDSysfs(cards []*ghwgpu.GraphicsCard) []*DeviceStat {
	var stats []*DeviceStat
	for _, card := range cards {
		cardDir := findDRMCardDir(card.Address)
		if cardDir == "" {
			continue
		}
		deviceDir := filepath.Join(cardDir, "device")

		name := cardProductName(card)
		if name == "" {
			name = readSysfsStr(filepath.Join(deviceDir, "product_name"))
		}
		if name == "" {
			name = "AMD GPU"
		}

		// 判断是否为 APU 核显：无独立 VRAM 则跳过
		memTotal := parseSysfsUint64(filepath.Join(deviceDir, "mem_info_vram_total"))
		if memTotal == 0 {
			continue
		}

		util := parseFloatOrDefault(readSysfsStr(filepath.Join(deviceDir, "gpu_busy_percent")), 0)
		memUsed := parseSysfsUint64(filepath.Join(deviceDir, "mem_info_vram_used"))

		temp := -1
		if t := readHwmonTemp(cardDir, "edge"); t >= 0 {
			temp = t
		}
		power := -1.0
		if p := readHwmonPower(cardDir); p >= 0 {
			power = p
		}
		fan := -1
		if f := readHwmonFanPercent(cardDir); f >= 0 {
			fan = f
		}

		stats = append(stats, &DeviceStat{
			Address:     cardAddress(card),
			Name:        name,
			Vendor:      "amd",
			MemoryUsed:  memUsed,
			MemoryTotal: memTotal,
			Utilization: util,
			Temperature: temp,
			PowerUsage:  power,
			FanSpeed:    fan,
		})
	}
	return stats
}

func collectAMDRocmSMI(ctx context.Context, cards []*ghwgpu.GraphicsCard) ([]*DeviceStat, error) {
	path, err := exec.LookPath("rocm-smi")
	if err != nil || path == "" {
		return nil, nil
	}

	cmdCtx, cancel := commandContext(ctx)
	defer cancel()

	out, err := exec.CommandContext(cmdCtx, path,
		"--showid", "--showuse", "--showmeminfo", "vram", "--showtemp", "--showpower",
		"--csv",
	).Output()
	if err != nil {
		return nil, err
	}

	records, err := readCSVRecords(out)
	if err != nil {
		return nil, err
	}
	if len(records) < 2 {
		return nil, nil
	}

	header := records[0]
	colIdx := make(map[string]int, len(header))
	for i, h := range header {
		colIdx[strings.TrimSpace(strings.ToLower(h))] = i
	}

	var stats []*DeviceStat
	cardIdx := 0
	for _, fields := range records[1:] {
		if len(fields) == 0 {
			continue
		}

		name := csvField(fields, colIdx, "card series")
		if name == "" {
			name = csvField(fields, colIdx, "name")
		}
		if cardIdx < len(cards) && name == "" {
			name = cardProductName(cards[cardIdx])
		}
		if name == "" {
			name = "AMD GPU"
		}

		memTotal := parseUint64Bytes(csvField(fields, colIdx, "vram total"), csvField(fields, colIdx, "vram total unit"))
		if memTotal == 0 {
			cardIdx++
			continue
		}

		addr := ""
		if cardIdx < len(cards) {
			addr = cardAddress(cards[cardIdx])
		}

		stats = append(stats, &DeviceStat{
			Address:     addr,
			Name:        strings.TrimSpace(name),
			Vendor:      "amd",
			MemoryUsed:  parseUint64Bytes(csvField(fields, colIdx, "vram used"), csvField(fields, colIdx, "vram used unit")),
			MemoryTotal: memTotal,
			Utilization: parseFloatOrDefault(csvField(fields, colIdx, "gpu use (%)"), 0),
			Temperature: parseIntOrDefault(csvField(fields, colIdx, "temperature (sensor edge) (c)"), parseIntOrDefault(csvField(fields, colIdx, "temperature (sensor junction) (c)"), -1)),
			PowerUsage:  parseFloatOrDefault(csvField(fields, colIdx, "average graphics package power (w)"), -1),
			FanSpeed:    -1,
		})
		cardIdx++
	}
	return stats, nil
}
