package gateway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"isrvd/server/service/account"
)

// 节点视角下的启动数据（/n/<节点ID>/api/overview/bootstrap）：
// 认证、AI 助手、API 文档入口取中控，业务服务可用性、标题、文件限额、应用市场取节点，快捷链接两者合并。

// nodeBootstrap 以中控的认证信息、节点自身的服务可用性和页面配置组合启动数据。
// 必须串行：只有中控确认了调用者身份才会去请求节点，未登录的请求不应触发隧道流量。
func (g *gateway) nodeBootstrap(w http.ResponseWriter, r *http.Request, id string) {
	status, body, err := g.bootstrapCall(r)
	if err != nil {
		fail(w, r, http.StatusBadGateway, "中控 isrvd 不可用")
		return
	}
	if status == http.StatusOK {
		if merged := g.mergeBootstrap(r.Context(), id, body); merged != nil {
			body = merged
		}
	}
	writeRaw(w, status, body)
}

// bootstrapCall 仅为节点视角的页面加载中控启动数据，不用于业务请求的身份校验。
// 只传递认证与语言信息，避免把浏览器的条件请求、升级或内容头带入聚合查询。
func (g *gateway) bootstrapCall(r *http.Request) (int, []byte, error) {
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, g.upstream.JoinPath("api/overview/bootstrap").String(), nil)
	if err != nil {
		return 0, nil, err
	}
	for _, name := range []string{"Authorization", "Accept-Language", account.GatewayUserHeader} {
		if value := r.Header.Get(name); value != "" {
			req.Header.Set(name, value)
		}
	}
	if lang := r.URL.Query().Get("lang"); lang != "" {
		req.URL.RawQuery = url.Values{"lang": {lang}}.Encode()
	}
	// WebSocket 与下载无法携带请求头，凭据在 ?token= 中；仅限 isrvd 允许的范围
	if req.Header.Get("Authorization") == "" && g.allowQueryToken(r) {
		if token := r.URL.Query().Get("token"); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	return resp.StatusCode, body, err
}

// mergeBootstrap 把节点的启动数据合并进中控的响应；中控响应解析失败或未登录时返回 nil。
// 节点不可达时保留中控的配置，业务服务全部视为不可用。
func (g *gateway) mergeBootstrap(ctx context.Context, id string, body []byte) []byte {
	var env map[string]any
	if json.Unmarshal(body, &env) != nil {
		return nil
	}
	// 对 nil map 取值是安全的，缺失层级一律得到零值
	payload, _ := env["payload"].(map[string]any)
	auth, _ := payload["auth"].(map[string]any)
	if username, _ := auth["username"].(string); username == "" {
		return nil
	}

	cfg, _ := payload["config"].(map[string]any)
	if cfg == nil {
		cfg = map[string]any{}
		payload["config"] = cfg
	}
	probe := map[string]any{"node": false, "copilot": false, "apisix": false, "caddy": false, "docker": false, "swarm": false, "compose": false}
	local, _ := payload["probe"].(map[string]any)
	payload["probe"] = probe

	var remote struct {
		Payload struct {
			Probe  map[string]any `json:"probe"`
			Config map[string]any `json:"config"`
		} `json:"payload"`
	}
	if data, err := g.nodes.Fetch(ctx, id, "/api/overview/bootstrap"); err == nil {
		_ = json.Unmarshal(data, &remote)
	}
	applyNodeBootstrap(cfg, probe, local, remote.Payload.Config, remote.Payload.Probe)

	out, err := json.Marshal(env)
	if err != nil {
		return nil
	}
	return out
}

// applyNodeBootstrap 合并启动数据中的 config 与 probe，节点内容不可信，类型不符的值一律忽略。
//   - probe：node、copilot 取中控（localProbe），其余业务服务取节点
//   - config：title 随节点（用于区分机器），文件限额与应用市场地址取节点，links 与中控合并，其余保持中控
func applyNodeBootstrap(cfg, probe, localProbe, remoteCfg, remoteProbe map[string]any) {
	for _, k := range []string{"node", "copilot"} {
		if v, ok := localProbe[k]; ok {
			probe[k] = v
		}
	}
	for _, k := range []string{"apisix", "caddy", "docker", "swarm", "compose"} {
		if v, ok := remoteProbe[k]; ok {
			probe[k] = v
		}
	}
	for _, k := range []string{"title", "marketplaceUrl"} {
		if v, ok := remoteCfg[k].(string); ok {
			cfg[k] = v
		}
	}
	for _, k := range []string{"maxUploadSize", "maxEditSize"} {
		if v, ok := remoteCfg[k].(float64); ok {
			cfg[k] = v
		}
	}
	cfg["links"] = mergeLinks(cfg["links"], remoteCfg["links"])
}

// mergeLinks 合并快捷链接：中控在前、节点在后，按 url 去重。
// 节点内容不可信：只接受 http(s) 链接，避免注入 javascript: 等链接；
// 并只保留 label、url、icon 三个字符串字段，避免类型异常导致中控顶栏渲染出错。
func mergeLinks(center, remote any) []any {
	out := []any{}
	seen := map[string]bool{}
	for _, list := range []any{center, remote} {
		items, _ := list.([]any)
		for _, item := range items {
			raw, _ := item.(map[string]any)
			label, _ := raw["label"].(string)
			url, _ := raw["url"].(string)
			icon, _ := raw["icon"].(string)
			lower := strings.ToLower(url)
			if seen[url] || !(strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "http://")) {
				continue
			}
			seen[url] = true
			out = append(out, map[string]any{"label": label, "url": url, "icon": icon})
		}
	}
	return out
}
