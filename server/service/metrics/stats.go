package metrics

import "time"

// StatsPoint 趋势图上的一个点，速率单位为 次/秒，耗时单位为毫秒
type StatsPoint struct {
	Ts   int64   `json:"ts"`
	RPS  float64 `json:"rps"`  // 总请求速率
	S2xx float64 `json:"s2xx"` // 2xx 速率
	S3xx float64 `json:"s3xx"` // 3xx 速率
	S4xx float64 `json:"s4xx"` // 4xx 速率
	S5xx float64 `json:"s5xx"` // 5xx 速率
	Avg  float64 `json:"avg"`  // 平均耗时（ms）
}

// StatsSummary 查询窗口内的汇总
type StatsSummary struct {
	Requests  float64 `json:"requests"`  // 请求总数
	S2xx      float64 `json:"s2xx"`      // 2xx 请求数
	S3xx      float64 `json:"s3xx"`      // 3xx 请求数
	S4xx      float64 `json:"s4xx"`      // 4xx 请求数
	S5xx      float64 `json:"s5xx"`      // 5xx 请求数
	ErrorRate float64 `json:"errorRate"` // 5xx 占比（0-1）
	RPS       float64 `json:"rps"`       // 平均请求速率
	Avg       float64 `json:"avg"`       // 平均耗时（ms）
	BytesOut  float64 `json:"bytesOut"`  // 响应流量（字节）
}

// SiteStat 单个站点在查询窗口内的汇总
type SiteStat struct {
	Site     string  `json:"site"`
	Requests float64 `json:"requests"` // 请求总数
	S4xx     float64 `json:"s4xx"`     // 4xx 请求数
	S5xx     float64 `json:"s5xx"`     // 5xx 请求数
	Avg      float64 `json:"avg"`      // 平均耗时（ms）
	BytesOut float64 `json:"bytesOut"` // 响应流量（字节）
}

// GatewayStats 单个网关的统计
type GatewayStats struct {
	Name    string       `json:"name"`            // 网关标识：caddy / apisix
	Summary StatsSummary `json:"summary"`         // 窗口汇总
	Series  []StatsPoint `json:"series"`          // 趋势
	Sites   []SiteStat   `json:"sites"`           // 站点排行
	Error   string       `json:"error,omitempty"` // 无法提供可靠统计时的说明
}

const (
	maxWindow = 24 * 3600 // 查询窗口上限（秒）
	minWindow = 60
)

// Stats 查询最近 sinceSeconds 秒内各网关的统计；没有采样数据的网关返回空序列。
func (s *Service) Stats(sinceSeconds int64) []GatewayStats {
	sinceSeconds = min(max(sinceSeconds, minWindow), maxWindow)
	until := time.Now().Unix()
	cutoff := until - sinceSeconds

	out := make([]GatewayStats, 0, len(s.sources))
	for _, src := range s.sources {
		s.mu.Lock()
		var records []record
		for _, r := range s.history[src.name] {
			if r.ts > cutoff && r.start < until {
				records = append(records, r)
			}
		}
		s.mu.Unlock()
		out = append(out, compute(src.name, records, cutoff, until))
	}
	return out
}

const seriesPoints = 120 // 趋势图目标点数

// compute 汇总区间增量；跨窗口边界的部分按时间比例估算，不外推未采样时段。
func compute(name string, records []record, cutoff, until int64) GatewayStats {
	g := GatewayStats{Name: name, Series: []StatsPoint{}, Sites: []SiteStat{}}
	if name == GatewayCaddy {
		g.Error = "Caddy 原生指标按处理器计数，无法可靠生成访问统计；原始指标出口仍可用。"
		return g
	}

	step := max(1, (len(records)+seriesPoints-1)/seriesPoints)
	var sum, bucket counters
	sites := map[string]counters{}
	var covered, bucketSeconds, bucketTs int64
	flush := func() {
		if bucketSeconds == 0 {
			return
		}
		secs := float64(bucketSeconds)
		g.Series = append(g.Series, StatsPoint{
			Ts: bucketTs, RPS: bucket.requests() / secs, S2xx: bucket.R2 / secs, S3xx: bucket.R3 / secs, S4xx: bucket.R4 / secs, S5xx: bucket.R5 / secs, Avg: avgMs(bucket),
		})
		bucket, bucketSeconds = counters{}, 0
	}
	for i, rec := range records {
		dt := rec.ts - rec.start
		start, end := max(rec.start, cutoff), min(rec.ts, until)
		if dt <= 0 || dt > maxSampleGap || end <= start {
			flush()
			continue
		}
		if bucketSeconds > 0 && start != bucketTs {
			flush()
		}
		seconds := end - start
		factor := float64(seconds) / float64(dt)
		d := rec.total.scale(factor)
		covered += seconds
		sum = sum.plus(d)
		for site, c := range rec.sites {
			sites[site] = sites[site].plus(c.scale(factor))
		}
		bucket = bucket.plus(d)
		bucketSeconds += seconds
		bucketTs = end
		if (i+1)%step == 0 {
			flush()
		}
	}
	flush()

	total := sum.requests()
	g.Summary = StatsSummary{Requests: total, S2xx: sum.R2, S3xx: sum.R3, S4xx: sum.R4, S5xx: sum.R5, Avg: avgMs(sum), BytesOut: sum.Out}
	if total > 0 {
		g.Summary.ErrorRate = sum.R5 / total
	}
	if covered > 0 {
		g.Summary.RPS = total / float64(covered)
	}

	for _, site := range topSites(sites) {
		c := sites[site]
		if c.requests() > 0 {
			g.Sites = append(g.Sites, SiteStat{Site: site, Requests: c.requests(), S4xx: c.R4, S5xx: c.R5, Avg: avgMs(c), BytesOut: c.Out})
		}
	}
	return g
}

// ─── 辅助函数 ───

func avgMs(c counters) float64 {
	if c.LatCnt <= 0 {
		return 0
	}
	return c.LatSum / c.LatCnt * 1000
}
