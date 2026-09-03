package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// ErrOpenAPIUnavailable 嵌入的 OpenAPI 文档缺失或尚未加载
var ErrOpenAPIUnavailable = errors.New("OpenAPI 文档不可用")

const openAPIListLimit = 40

// OpenAPIQuery 查阅官方 OpenAPI 的过滤条件；均不传时返回模块目录
type OpenAPIQuery struct {
	Tag    string `form:"tag" json:"tag"`       // 模块标签，如 docker、apisix
	Path   string `form:"path" json:"path"`     // API 路径，如 /docker/containers
	Method string `form:"method" json:"method"` // HTTP 方法：get / post / put / patch / delete
	Q      string `form:"q" json:"q"`           // 关键词，匹配路径、摘要、operationId
}

type openAPISpec struct {
	schemas map[string]any
	ops     []openAPIOp
}

type openAPIOp struct {
	Method      string
	Path        string
	Summary     string
	Description string
	OperationID string
	Tag         string
	Raw         map[string]any
}

// OpenAPITagInfo 模块目录中的标签与接口数量
type OpenAPITagInfo struct {
	Name  string `json:"name"`  // 模块标签
	Count int    `json:"count"` // 该标签下的接口数量
}

// OpenAPIOpBrief 接口列表中的单条摘要
type OpenAPIOpBrief struct {
	Method      string `json:"method"`                // HTTP 方法
	Path        string `json:"path"`                  // 相对于 /api 的路径
	Summary     string `json:"summary"`               // 中文摘要
	OperationID string `json:"operationId,omitempty"` // OpenAPI operationId
	Tag         string `json:"tag,omitempty"`         // 模块标签
}

// OpenAPILookupResult 查阅官方 OpenAPI 的结果，mode 为 catalog / list / detail
type OpenAPILookupResult struct {
	Mode        string           `json:"mode"`                  // catalog / list / detail
	Hint        string           `json:"hint,omitempty"`        // 下一步查阅建议
	Tags        []OpenAPITagInfo `json:"tags,omitempty"`        // catalog：模块目录
	Total       int              `json:"total,omitempty"`       // list：匹配总数
	Truncated   bool             `json:"truncated,omitempty"`   // list：是否截断
	Operations  []OpenAPIOpBrief `json:"operations,omitempty"`  // list：接口摘要
	Method      string           `json:"method,omitempty"`      // detail：HTTP 方法
	Path        string           `json:"path,omitempty"`        // detail：相对于 /api 的路径
	Summary     string           `json:"summary,omitempty"`     // detail：中文摘要
	Description string           `json:"description,omitempty"` // detail：补充说明
	OperationID string           `json:"operationId,omitempty"` // detail：OpenAPI operationId
	Tag         string           `json:"tag,omitempty"`         // detail：模块标签
	Parameters  []map[string]any `json:"parameters,omitempty"`  // detail：路径/查询参数
	RequestBody any              `json:"requestBody,omitempty"` // detail：请求体字段（已展开 $ref）
	Response    any              `json:"response,omitempty"`    // detail：200 响应字段（已展开 $ref）
}

// LoadOpenAPI 解析 OpenAPI 3 文档并建立查阅索引。
func (s *Service) LoadOpenAPI(data []byte) error {
	spec, err := parseOpenAPI(data)
	if err != nil {
		return err
	}
	s.spec = spec
	return nil
}

// OpenAPILookup 按模块、路径或关键词查阅官方 OpenAPI。
func (s *Service) OpenAPILookup(q OpenAPIQuery) (*OpenAPILookupResult, error) {
	if s.spec == nil {
		return nil, ErrOpenAPIUnavailable
	}

	tag := strings.TrimSpace(strings.ToLower(q.Tag))
	method := strings.TrimSpace(strings.ToLower(q.Method))
	keyword := strings.TrimSpace(strings.ToLower(q.Q))
	path := normalizeOpenAPIPath(q.Path)

	if tag == "" && method == "" && keyword == "" && path == "" {
		return s.spec.catalog(), nil
	}

	matched := s.spec.filter(tag, method, keyword, path)
	if len(matched) == 1 {
		return s.spec.detail(matched[0]), nil
	}
	return s.spec.list(matched), nil
}

func parseOpenAPI(data []byte) (*openAPISpec, error) {
	var doc struct {
		Paths      map[string]map[string]json.RawMessage `json:"paths"`
		Components struct {
			Schemas map[string]any `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("解析 OpenAPI 文档失败: %w", err)
	}
	if len(doc.Paths) == 0 {
		return nil, fmt.Errorf("OpenAPI 文档不含任何路径")
	}

	spec := &openAPISpec{schemas: doc.Components.Schemas}
	for path, methods := range doc.Paths {
		for name, raw := range methods {
			var op map[string]any
			if err := json.Unmarshal(raw, &op); err != nil {
				continue
			}
			spec.ops = append(spec.ops, openAPIOp{
				Method:      strings.ToLower(name),
				Path:        path,
				Summary:     stringValue(op["summary"]),
				Description: stringValue(op["description"]),
				OperationID: stringValue(op["operationId"]),
				Tag:         firstTag(op["tags"]),
				Raw:         op,
			})
		}
	}
	sort.Slice(spec.ops, func(i, j int) bool {
		if spec.ops[i].Path != spec.ops[j].Path {
			return spec.ops[i].Path < spec.ops[j].Path
		}
		return spec.ops[i].Method < spec.ops[j].Method
	})
	return spec, nil
}

func (spec *openAPISpec) catalog() *OpenAPILookupResult {
	counts := map[string]int{}
	for _, op := range spec.ops {
		name := op.Tag
		if name == "" {
			name = "other"
		}
		counts[name]++
	}
	tags := make([]OpenAPITagInfo, 0, len(counts))
	for name, count := range counts {
		tags = append(tags, OpenAPITagInfo{Name: name, Count: count})
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].Name < tags[j].Name })
	return &OpenAPILookupResult{
		Mode: "catalog",
		Hint: "指定 tag 或 q 查看接口列表，再对具体 path（可选 method）取请求/响应字段。路径均相对于 /api。",
		Tags: tags,
	}
}

func (spec *openAPISpec) filter(tag, method, keyword, path string) []openAPIOp {
	var out []openAPIOp
	for _, op := range spec.ops {
		if tag != "" && !strings.EqualFold(op.Tag, tag) {
			continue
		}
		if method != "" && op.Method != method {
			continue
		}
		if path != "" && !pathMatches(op.Path, path) {
			continue
		}
		if keyword != "" && !op.matchesKeyword(keyword) {
			continue
		}
		out = append(out, op)
	}
	return out
}

func (spec *openAPISpec) list(ops []openAPIOp) *OpenAPILookupResult {
	result := &OpenAPILookupResult{
		Mode:  "list",
		Total: len(ops),
		Hint:  "对需要调用的接口再查一次，带上 path 与 method，以获取参数和字段定义。",
	}
	if len(ops) == 0 {
		result.Hint = "没有匹配的接口，换 tag、关键词或更短的 path 再试。"
		return result
	}
	if len(ops) > openAPIListLimit {
		result.Truncated = true
		ops = ops[:openAPIListLimit]
	}
	result.Operations = make([]OpenAPIOpBrief, 0, len(ops))
	for _, op := range ops {
		result.Operations = append(result.Operations, OpenAPIOpBrief{
			Method:      op.Method,
			Path:        op.Path,
			Summary:     op.Summary,
			OperationID: op.OperationID,
			Tag:         op.Tag,
		})
	}
	return result
}

func (spec *openAPISpec) detail(op openAPIOp) *OpenAPILookupResult {
	return &OpenAPILookupResult{
		Mode:        "detail",
		Hint:        "调用 isrvd_api 时 path 去掉开头的 /（相对于 /api），路径参数把 {name} 换成实际值。",
		Method:      op.Method,
		Path:        op.Path,
		Summary:     op.Summary,
		Description: op.Description,
		OperationID: op.OperationID,
		Tag:         op.Tag,
		Parameters:  spec.collectParameters(op.Raw),
		RequestBody: spec.collectRequestBody(op.Raw),
		Response:    spec.collectResponse(op.Raw),
	}
}

func (op openAPIOp) matchesKeyword(keyword string) bool {
	if strings.Contains(strings.ToLower(op.Path), keyword) {
		return true
	}
	if strings.Contains(strings.ToLower(op.Summary), keyword) {
		return true
	}
	if strings.Contains(strings.ToLower(op.OperationID), keyword) {
		return true
	}
	return strings.Contains(strings.ToLower(op.Tag), keyword)
}

func (spec *openAPISpec) collectParameters(op map[string]any) []map[string]any {
	raw, ok := op["parameters"].([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		param := map[string]any{
			"name":     m["name"],
			"in":       m["in"],
			"required": m["required"],
		}
		if desc := stringValue(m["description"]); desc != "" {
			param["description"] = desc
		}
		if schema, ok := m["schema"]; ok {
			param["schema"] = spec.resolveSchema(schema, 0)
		}
		out = append(out, param)
	}
	return out
}

func (spec *openAPISpec) collectRequestBody(op map[string]any) any {
	body, ok := op["requestBody"].(map[string]any)
	if !ok {
		return nil
	}
	schema := mediaSchema(body)
	if schema == nil {
		return nil
	}
	return spec.resolveSchema(schema, 0)
}

func (spec *openAPISpec) collectResponse(op map[string]any) any {
	responses, ok := op["responses"].(map[string]any)
	if !ok {
		return nil
	}
	resp, ok := responses["200"].(map[string]any)
	if !ok {
		return nil
	}
	schema := mediaSchema(resp)
	if schema == nil {
		return nil
	}
	return spec.resolveSchema(schema, 0)
}

func (spec *openAPISpec) resolveSchema(node any, depth int) any {
	if depth > 5 {
		return node
	}
	switch typed := node.(type) {
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = spec.resolveSchema(item, depth)
		}
		return out
	case map[string]any:
		if ref, ok := typed["$ref"].(string); ok {
			name := strings.TrimPrefix(ref, "#/components/schemas/")
			target, ok := spec.schemas[name]
			if !ok {
				return typed
			}
			return spec.resolveSchema(target, depth+1)
		}
		out := make(map[string]any, len(typed))
		for key, value := range typed {
			out[key] = spec.resolveSchema(value, depth)
		}
		return out
	default:
		return node
	}
}

// ─── 辅助函数 ───

func normalizeOpenAPIPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	path = strings.TrimPrefix(path, "/api")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if strings.HasPrefix(part, ":") || strings.HasPrefix(part, "*") {
			parts[i] = "{" + part[1:] + "}"
		}
	}
	return strings.TrimRight(strings.Join(parts, "/"), "/")
}

func pathMatches(specPath, queryPath string) bool {
	return strings.Contains(normalizeOpenAPIPath(specPath), queryPath)
}

func mediaSchema(node map[string]any) any {
	content, ok := node["content"].(map[string]any)
	if !ok {
		return nil
	}
	for _, key := range []string{"application/json", "multipart/form-data"} {
		if media, ok := content[key].(map[string]any); ok {
			if schema, ok := media["schema"]; ok {
				return schema
			}
		}
	}
	return nil
}

func firstTag(v any) string {
	tags, ok := v.([]any)
	if !ok || len(tags) == 0 {
		return ""
	}
	return stringValue(tags[0])
}

func stringValue(v any) string {
	s, _ := v.(string)
	return s
}
