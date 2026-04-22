package gpu

import (
	"path/filepath"

	ghwgpu "github.com/jaypipes/ghw/pkg/gpu"
)

func collectIntelCards(cards []*ghwgpu.GraphicsCard) []*DeviceStat {
	if len(cards) == 0 {
		return nil
	}

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
			name = "Intel GPU"
		}

		util := parseFloatOrDefault(readSysfsStr(filepath.Join(deviceDir, "gpu_busy_percent")), 0)
		memUsed := parseSysfsUint64(filepath.Join(deviceDir, "mem_info_vram_used"))
		memTotal := parseSysfsUint64(filepath.Join(deviceDir, "mem_info_vram_total"))

		temp := -1
		if t := readHwmonTemp(cardDir, ""); t >= 0 {
			temp = t
		}
		power := -1.0
		if p := readHwmonPower(cardDir); p >= 0 {
			power = p
		}

		stats = append(stats, &DeviceStat{
			Address:     cardAddress(card),
			Name:        name,
			Vendor:      "intel",
			MemoryUsed:  memUsed,
			MemoryTotal: memTotal,
			Utilization: util,
			Temperature: temp,
			PowerUsage:  power,
			FanSpeed:    -1,
		})
	}
	return stats
}
