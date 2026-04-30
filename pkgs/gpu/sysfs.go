package gpu

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// findDRMCardDir 通过 PCI 地址找到 /sys/class/drm/cardN 目录
func findDRMCardDir(pciAddr string) string {
	pciAddr = normalizePCIAddress(pciAddr)
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
		if strings.Contains(strings.ToLower(link), pciAddr) {
			return entry
		}
		// 也检查 device 软链接指向的 PCI 地址
		devLink, err := os.Readlink(filepath.Join(entry, "device"))
		if err == nil && strings.Contains(strings.ToLower(devLink), pciAddr) {
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
