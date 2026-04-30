package gpu

import (
	"context"
	"encoding/csv"
	"io"
	"strconv"
	"strings"
	"time"

	ghwgpu "github.com/jaypipes/ghw/pkg/gpu"
)

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

func commandContext(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, gpuCmdTimeout)
}

func normalizePCIAddress(addr string) string {
	addr = strings.ToLower(strings.TrimSpace(addr))
	if addr == "" {
		return ""
	}
	parts := strings.Split(addr, ":")
	if len(parts) != 3 {
		return addr
	}
	domain := parts[0]
	if len(domain) > 4 {
		domain = domain[len(domain)-4:]
	}
	for len(domain) < 4 {
		domain = "0" + domain
	}
	return domain + ":" + parts[1] + ":" + parts[2]
}

func readCSVRecords(data []byte) ([][]string, error) {
	r := csv.NewReader(strings.NewReader(string(data)))
	r.TrimLeadingSpace = true
	var records [][]string
	for {
		record, err := r.Read()
		if err == io.EOF {
			return records, nil
		}
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
}

func addressSet(cards []*ghwgpu.GraphicsCard) map[string]bool {
	if len(cards) == 0 {
		return nil
	}
	allowed := make(map[string]bool, len(cards))
	for _, card := range cards {
		if addr := cardAddress(card); addr != "" {
			allowed[addr] = true
		}
	}
	if len(allowed) == 0 {
		return nil
	}
	return allowed
}

func csvField(fields []string, colIdx map[string]int, key string) string {
	if idx, ok := colIdx[key]; ok && idx < len(fields) {
		return strings.TrimSpace(fields[idx])
	}
	return ""
}

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
