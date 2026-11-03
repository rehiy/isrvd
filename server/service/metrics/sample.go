package metrics

import (
	"context"
	"math"
	"sort"
	"strings"
	"time"
)

const maxSites = 20 // 每个采样区间保留请求增量最多的站点数

// counters 请求计数、响应字节数与耗时，可表示累计值或区间增量。
type counters struct {
	R2, R3, R4, R5 float64
	Out            float64
	LatSum, LatCnt float64 // 耗时总和（秒）与样本数
}

// record 一个有效采样区间的增量，查询时不再处理计数器重置。
type record struct {
	start, ts int64
	total     counters
	sites     map[string]counters
}

// observation 单个网关的完整原始序列基线，不存入历史。
type observation struct {
	ts     int64
	series map[string]counterValue
}

type counterValue struct {
	value float64
	ts    int64 // 最后出现时间，缺失序列最多保留 5 分钟
}

func (c counters) requests() float64 { return c.R2 + c.R3 + c.R4 + c.R5 }

func (c counters) plus(o counters) counters {
	return counters{c.R2 + o.R2, c.R3 + o.R3, c.R4 + o.R4, c.R5 + o.R5, c.Out + o.Out, c.LatSum + o.LatSum, c.LatCnt + o.LatCnt}
}

func (c counters) scale(factor float64) counters {
	return counters{c.R2 * factor, c.R3 * factor, c.R4 * factor, c.R5 * factor, c.Out * factor, c.LatSum * factor, c.LatCnt * factor}
}

func (c counters) finite() bool {
	for _, v := range []float64{c.R2, c.R3, c.R4, c.R5, c.Out, c.LatSum, c.LatCnt, c.requests(), avgMs(c)} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	return true
}

// ─── 采样 ───

const sampleInterval = 30 * time.Second // 采样间隔

// Start 启动后台采样；已启动或没有抓取源时忽略。
func (s *Service) Start(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil || len(s.sources) == 0 {
		return
	}
	ctx, s.cancel = context.WithCancel(ctx)
	s.done = make(chan struct{})
	go func() {
		defer close(s.done)
		s.sample(ctx)
		ticker := time.NewTicker(sampleInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.sample(ctx)
			}
		}
	}()
}

// Stop 停止采样
func (s *Service) Stop() {
	if s == nil {
		return
	}
	s.mu.Lock()
	cancel, done := s.cancel, s.done
	s.cancel = nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
		<-done
	}
}

const maxSamples = 2880 // 每个网关保留的区间数（30 秒 × 2880 = 24 小时）

// sample 抓取并记录有效区间；失败不更新基线。
func (s *Service) sample(ctx context.Context) {
	for _, src := range s.sources {
		if src.name != GatewayApisix {
			continue // Caddy 原生 handler 指标不能还原整请求统计
		}
		data, err := s.fetch(ctx, src.url)
		if err != nil {
			continue
		}
		rec, ok := src.observe(data, time.Now().Unix())
		if !ok {
			continue
		}
		s.mu.Lock()
		h := append(s.history[src.name], rec)
		if len(h) > maxSamples {
			h = h[len(h)-maxSamples:]
		}
		s.history[src.name] = h
		s.mu.Unlock()
	}
}

const maxSampleGap = 300 // 超过 5 分钟的采样中断不计入统计

// observe 验证原文，按完整标签标识逐序列差分，再聚合为区间增量。
func (src *source) observe(data []byte, ts int64) (record, bool) {
	samples, err := parseText(data)
	if err != nil {
		return record{}, false
	}
	prev := src.last
	if prev != nil && ts <= prev.ts {
		return record{}, false
	}
	next := &observation{ts: ts, series: map[string]counterValue{}}
	rec := record{ts: ts, sites: map[string]counters{}}
	for _, sm := range samples {
		if _, tracked := sm.counter(sm.value); !tracked {
			continue
		}
		if sm.value < 0 || math.IsNaN(sm.value) || math.IsInf(sm.value, 0) {
			return record{}, false
		}
		key := sm.key()
		if _, duplicate := next.series[key]; duplicate {
			return record{}, false
		}
		next.series[key] = counterValue{value: sm.value, ts: ts}
		value := sm.value
		if prev != nil {
			if old, ok := prev.series[key]; ok && ts-old.ts <= maxSampleGap && value >= old.value {
				value -= old.value
			}
		}
		c, _ := sm.counter(value)
		rec.total = rec.total.plus(c)
		site := sm.labels["route"]
		if site == "" {
			site = "-"
		}
		rec.sites[site] = rec.sites[site].plus(c)
	}
	if !rec.total.scale(maxSamples).finite() {
		return record{}, false // 留出整个历史窗口累加的有限数值空间
	}
	if prev != nil && len(next.series) == 0 && len(prev.series) > 0 {
		return record{}, false
	}
	if prev == nil || ts-prev.ts > maxSampleGap {
		src.last = next
		return record{}, false
	}
	rec.start = prev.ts
	for key, old := range prev.series {
		if _, exists := next.series[key]; !exists && ts-old.ts <= maxSampleGap {
			next.series[key] = old
		}
	}
	src.last = next
	if len(rec.sites) > maxSites {
		sites := make(map[string]counters, maxSites)
		for _, name := range topSites(rec.sites) {
			sites[name] = rec.sites[name]
		}
		rec.sites = sites
	}
	return rec, true
}

// ─── 辅助函数 ───

// counter 只选择 APISIX 请求类计数，耗时由毫秒换算为秒。
func (sm sample) counter(value float64) (c counters, tracked bool) {
	switch sm.name {
	case "apisix_http_status":
		switch {
		case strings.HasPrefix(sm.labels["code"], "2"):
			c.R2 = value
		case strings.HasPrefix(sm.labels["code"], "3"):
			c.R3 = value
		case strings.HasPrefix(sm.labels["code"], "4"):
			c.R4 = value
		case strings.HasPrefix(sm.labels["code"], "5"):
			c.R5 = value
		default:
			return c, false
		}
	case "apisix_bandwidth":
		if sm.labels["type"] != "egress" {
			return c, false
		}
		c.Out = value
	case "apisix_http_latency_count":
		if sm.labels["type"] != "request" {
			return c, false
		}
		c.LatCnt = value
	case "apisix_http_latency_sum":
		if sm.labels["type"] != "request" {
			return c, false
		}
		c.LatSum = value / 1000
	default:
		return c, false
	}
	return c, true
}

// topSites 返回请求增量最多的站点名称，同值按名称排序。
func topSites(sites map[string]counters) []string {
	names := make([]string, 0, len(sites))
	for name := range sites {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		a, b := sites[names[i]].requests(), sites[names[j]].requests()
		if a == b {
			return names[i] < names[j]
		}
		return a > b
	})
	return names[:min(len(names), maxSites)]
}
