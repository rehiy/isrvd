package system

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jaypipes/ghw"
	ghwgpu "github.com/jaypipes/ghw/pkg/gpu"
)

// GPUStat GPU 统计信息
type GPUStat struct {
	Index       int     `json:"index"`
	Name        string  `json:"name"`
	Vendor      string  `json:"vendor"`      // "nvidia" | "amd" | "intel"
	MemoryUsed  uint64  `json:"memoryUsed"`  // bytes
	MemoryTotal uint64  `json:"memoryTotal"` // bytes
	Utilization float64 `json:"utilization"` // 0-100
	Temperature int     `json:"temperature"` // celsius, -1 = N/A
	PowerUsage  float64 `json:"powerUsage"`  // watts, -1 = N/A
	FanSpeed    int     `json:"fanSpeed"`    // percent, -1 = N/A
}

const gpuCmdTimeout = 2 * time.Second

// 虚拟显卡 PCI vendor ID
var virtualVendorIDs = map[string]bool{
	"15ad": true, // VMware
	"1234": true, // QEMU/Bochs
	"1af4": true, // VirtIO
	"1b36": true, // Red Hat QXL
	"1414": true, // Microsoft Hyper-V
	"80ee": true, // VirtualBox
}

// 核显产品名关键词（匹配到即视为核显）
var iGPUKeywords = []string{
	"HD Graphics", "UHD Graphics", "Iris Graphics", "Iris Plus",
	"Iris Pro", "Iris Xe", "GMA",
}

// ─── 第一层：统一发现 ───

// GetGPUStats 采集所有 GPU 统计信息（ghw 发现 → 按厂商分发采集）
func GetGPUStats() []*GPUStat {
	gpuInfo, err := ghw.GPU()
	if err != nil || gpuInfo == nil {
		return nil
	}

	var nvidiaCards, amdCards, intelCards []*ghwgpu.GraphicsCard
	for _, card := range gpuInfo.GraphicsCards {
		if card.DeviceInfo == nil || card.DeviceInfo.Vendor == nil {
			continue
		}
		vid := card.DeviceInfo.Vendor.ID

		if virtualVendorIDs[vid] {
			continue
		}
		if isIntegratedGPU(card) {
			continue
		}

		switch vid {
		case "10de":
			nvidiaCards = append(nvidiaCards, card)
		case "1002":
			amdCards = append(amdCards, card)
		case "8086":
			intelCards = append(intelCards, card)
		}
	}

	var result []*GPUStat
	result = append(result, collectNvidiaCards(nvidiaCards)...)
	result = append(result, collectAMDCards(amdCards)...)
	result = append(result, collectIntelCards(intelCards)...)

	for i, s := range result {
		s.Index = i
	}
	return result
}

func isIntegratedGPU(card *ghwgpu.GraphicsCard) bool {
	if card.DeviceInfo == nil || card.DeviceInfo.Vendor == nil {
		return false
	}
	// 只对 Intel 做核显过滤；AMD APU 核显通常无独立 VRAM，后续采集时自然跳过
	if card.DeviceInfo.Vendor.ID != "8086" {
		return false
	}
	name := ""
	if card.DeviceInfo.Product != nil {
		name = card.DeviceInfo.Product.Name
	}
	// Intel Arc 独显保留
	if strings.Contains(name, "Arc") {
		return false
	}
	for _, kw := range iGPUKeywords {
		if strings.Contains(name, kw) {
			return true
		}
	}
	return false
}

func cardProductName(card *ghwgpu.GraphicsCard) string {
	if card.DeviceInfo != nil && card.DeviceInfo.Product != nil {
		return card.DeviceInfo.Product.Name
	}
	return ""
}

// ─── 第二层：NVIDIA 采集 ───

func collectNvidiaCards(cards []*ghwgpu.GraphicsCard) []*GPUStat {
	if len(cards) == 0 {
		return nil
	}
	return collectNvidiaSmi(cards)
}

// nvidia-smi fallback
func collectNvidiaSmi(cards []*ghwgpu.GraphicsCard) []*GPUStat {
	path, err := exec.LookPath("nvidia-smi")
	if err != nil || path == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), gpuCmdTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, "nvidia-smi",
		"--query-gpu=index,name,memory.used,memory.total,utilization.gpu,temperature.gpu,power.draw,fan.speed",
		"--format=csv,noheader,nounits",
	).Output()
	if err != nil {
		return nil
	}

	var stats []*GPUStat
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Split(line, ", ")
		if len(fields) < 8 {
			continue
		}

		idx, _ := strconv.Atoi(strings.TrimSpace(fields[0]))
		memUsedMB, _ := strconv.ParseFloat(strings.TrimSpace(fields[2]), 64)
		memTotalMB, _ := strconv.ParseFloat(strings.TrimSpace(fields[3]), 64)
		util, _ := strconv.ParseFloat(strings.TrimSpace(fields[4]), 64)

		stats = append(stats, &GPUStat{
			Index:       idx,
			Name:        strings.TrimSpace(fields[1]),
			Vendor:      "nvidia",
			MemoryUsed:  uint64(memUsedMB * 1024 * 1024),
			MemoryTotal: uint64(memTotalMB * 1024 * 1024),
			Utilization: util,
			Temperature: parseIntOrDefault(strings.TrimSpace(fields[5]), -1),
			PowerUsage:  parseFloatOrDefault(strings.TrimSpace(fields[6]), -1),
			FanSpeed:    parseIntOrDefault(strings.TrimSpace(fields[7]), -1),
		})
	}
	return stats
}

// ─── 第二层：AMD 采集 ───

func collectAMDCards(cards []*ghwgpu.GraphicsCard) []*GPUStat {
	if len(cards) == 0 {
		return nil
	}
	// 优先 sysfs（零依赖），fallback rocm-smi
	if stats := collectAMDSysfs(cards); len(stats) > 0 {
		return stats
	}
	return collectAMDRocmSmi(cards)
}

func collectAMDSysfs(cards []*ghwgpu.GraphicsCard) []*GPUStat {
	var stats []*GPUStat
	for i, card := range cards {
		cardDir := findDrmCardDir(card.Address)
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

		stats = append(stats, &GPUStat{
			Index:       i,
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

func collectAMDRocmSmi(cards []*ghwgpu.GraphicsCard) []*GPUStat {
	path, err := exec.LookPath("rocm-smi")
	if err != nil || path == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), gpuCmdTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, "rocm-smi",
		"--showid", "--showuse", "--showmeminfo", "vram", "--showtemp", "--showpower",
		"--csv",
	).Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(string(out), "\n")
	if len(lines) < 2 {
		return nil
	}

	header := strings.Split(lines[0], ",")
	colIdx := make(map[string]int)
	for i, h := range header {
		colIdx[strings.TrimSpace(strings.ToLower(h))] = i
	}

	var stats []*GPUStat
	cardIdx := 0
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Split(line, ",")

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
			continue
		}

		stats = append(stats, &GPUStat{
			Index:       cardIdx,
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
	return stats
}

// ─── 第二层：Intel 采集 ───

func collectIntelCards(cards []*ghwgpu.GraphicsCard) []*GPUStat {
	if len(cards) == 0 {
		return nil
	}

	var stats []*GPUStat
	for i, card := range cards {
		cardDir := findDrmCardDir(card.Address)
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

		stats = append(stats, &GPUStat{
			Index:       i,
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

// ─── DRM / sysfs 辅助函数 ───

// findDrmCardDir 通过 PCI 地址找到 /sys/class/drm/cardN 目录
func findDrmCardDir(pciAddr string) string {
	entries, _ := filepath.Glob("/sys/class/drm/card[0-9]*")
	for _, entry := range entries {
		// 跳过 cardN-* 类型的子设备（如 card0-DP-1）
		base := filepath.Base(entry)
		if strings.Contains(base, "-") {
			continue
		}
		link, err := os.Readlink(entry)
		if err != nil {
			link = entry
		}
		if strings.Contains(link, pciAddr) {
			return entry
		}
		// 也检查 device 软链接指向的 PCI 地址
		devLink, err := os.Readlink(filepath.Join(entry, "device"))
		if err == nil && strings.Contains(devLink, pciAddr) {
			return entry
		}
	}
	return ""
}

func readSysfsStr(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func parseSysfsUint64(path string) uint64 {
	s := readSysfsStr(path)
	if s == "" {
		return 0
	}
	v, _ := strconv.ParseUint(s, 10, 64)
	return v
}

func readHwmonTemp(cardDir string, label string) int {
	hwmons, _ := filepath.Glob(filepath.Join(cardDir, "device", "hwmon", "hwmon*"))
	for _, hw := range hwmons {
		inputs, _ := filepath.Glob(filepath.Join(hw, "temp*_input"))
		for _, input := range inputs {
			if label != "" {
				labelFile := strings.Replace(input, "_input", "_label", 1)
				l := readSysfsStr(labelFile)
				if !strings.EqualFold(l, label) {
					continue
				}
			}
			s := readSysfsStr(input)
			if s == "" {
				continue
			}
			v, err := strconv.Atoi(s)
			if err != nil {
				continue
			}
			return v / 1000
		}
	}
	return -1
}

func readHwmonPower(cardDir string) float64 {
	hwmons, _ := filepath.Glob(filepath.Join(cardDir, "device", "hwmon", "hwmon*"))
	for _, hw := range hwmons {
		inputs, _ := filepath.Glob(filepath.Join(hw, "power*_average"))
		for _, input := range inputs {
			s := readSysfsStr(input)
			if s == "" {
				continue
			}
			v, err := strconv.ParseFloat(s, 64)
			if err != nil {
				continue
			}
			return v / 1000000
		}
	}
	return -1
}

func readHwmonFanPercent(cardDir string) int {
	hwmons, _ := filepath.Glob(filepath.Join(cardDir, "device", "hwmon", "hwmon*"))
	for _, hw := range hwmons {
		s := readSysfsStr(filepath.Join(hw, "pwm1"))
		if s != "" {
			v, err := strconv.Atoi(s)
			if err == nil {
				return v * 100 / 255
			}
		}
	}
	return -1
}

// ─── 通用解析辅助函数 ───

func parseIntOrDefault(s string, def int) int {
	s = strings.TrimSpace(s)
	if s == "" || s == "[N/A]" || s == "N/A" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

func parseFloatOrDefault(s string, def float64) float64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "[N/A]" || s == "N/A" {
		return def
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return def
	}
	return v
}

func csvField(fields []string, colIdx map[string]int, key string) string {
	if idx, ok := colIdx[key]; ok && idx < len(fields) {
		return strings.TrimSpace(fields[idx])
	}
	return ""
}

func parseUint64Bytes(valStr, unitStr string) uint64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(valStr), 64)
	switch strings.ToLower(strings.TrimSpace(unitStr)) {
	case "gb":
		return uint64(v * 1024 * 1024 * 1024)
	case "mb":
		return uint64(v * 1024 * 1024)
	case "kb":
		return uint64(v * 1024)
	default:
		return uint64(v)
	}
}
