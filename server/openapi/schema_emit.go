package openapi

import (
	"fmt"
	"strings"
)

// 整数类型集合（用于 buildProperty 和 extractDefaultQueryParam 的快速判断）
var intTypes = map[string]bool{
	"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
	"byte": true, "rune": true,
}

// retainReferencedSchemas 仅发布接口可达的类型，并拒绝无法解析的引用。
func (oa *OpenAPI) retainReferencedSchemas() error {
	reachable := map[string]any{}
	var visit func(any) error
	visit = func(value any) error {
		switch value := value.(type) {
		case map[string]any:
			if ref, ok := value["$ref"].(string); ok {
				if name, ok := strings.CutPrefix(ref, "#/components/schemas/"); ok {
					if _, visited := reachable[name]; !visited {
						schema, exists := oa.Components.Schemas[name]
						if !exists {
							return fmt.Errorf("OpenAPI 类型引用无法解析: %s", name)
						}
						reachable[name] = schema
						if err := visit(schema); err != nil {
							return err
						}
					}
				}
			}
			for _, field := range value {
				if err := visit(field); err != nil {
					return err
				}
			}
		case []any:
			for _, item := range value {
				if err := visit(item); err != nil {
					return err
				}
			}
		case []map[string]any:
			for _, item := range value {
				if err := visit(item); err != nil {
					return err
				}
			}
		}
		return nil
	}
	for _, path := range oa.Paths {
		if err := visit(path); err != nil {
			return err
		}
	}
	oa.Components.Schemas = reachable
	return nil
}

func (g *generator) collectAllSchemas() map[string]*SchemaInfo {
	result := make(map[string]*SchemaInfo)
	for cacheKey, schema := range g.structCache {
		// 只保留规范化名称
		normalized := normalizeSchemaName(cacheKey)
		result[normalized] = schema
	}
	return result
}

func buildSchemaObject(schema *SchemaInfo, allSchemas map[string]*SchemaInfo) map[string]any {
	props := make(map[string]any)
	var required []string

	for _, f := range schema.Fields {
		prop := buildProperty(f.Type, allSchemas)

		// 添加字段描述（如果有）
		if f.Description != "" {
			prop["description"] = f.Description
		} else {
			// 为缺少注释的字段使用字段名作为默认描述
			prop["description"] = f.Name
		}

		props[f.Name] = prop
		if f.Required {
			required = append(required, f.Name)
		}
	}

	result := map[string]any{
		"type":       "object",
		"properties": props,
	}
	if len(required) > 0 {
		result["required"] = required
	}
	return result
}

// 标准库类型，避免在 buildProperty 中生成无效 $ref
var stdTypes = map[string]bool{
	"time.Time":     true,
	"time.Duration": true,
}

func buildProperty(goType string, allSchemas map[string]*SchemaInfo) map[string]any {
	// 处理指针类型
	goType = strings.TrimPrefix(goType, "*")

	switch {
	case goType == "string":
		return map[string]any{"type": "string"}
	case intTypes[goType]:
		return map[string]any{"type": "integer"}
	case goType == "float64" || goType == "float32":
		return map[string]any{"type": "number"}
	case goType == "bool":
		return map[string]any{"type": "boolean"}
	case strings.HasPrefix(goType, "[]"):
		itemType := strings.TrimPrefix(goType, "[]")
		itemType = strings.TrimPrefix(itemType, "*")
		items := buildProperty(itemType, allSchemas)
		return map[string]any{"type": "array", "items": items}
	case strings.HasPrefix(goType, "map["):
		return map[string]any{"type": "object"}
	case goType == "json.RawMessage":
		return map[string]any{"type": "object"}
	case goType == "any" || goType == "interface{}":
		return map[string]any{} // any type
	default:
		// 标准库类型降级为基本类型
		if stdTypes[goType] {
			return map[string]any{"type": "string"}
		}
		// 尝试引用其他 schema
		if _, _, ok := strings.Cut(goType, "."); ok {
			return map[string]any{"$ref": "#/components/schemas/" + normalizeSchemaName(goType)}
		}
		// 未知类型默认为 object
		return map[string]any{"type": "object"}
	}
}

// buildDataSchema 根据 Go 类型字符串生成 data 字段的 schema
func buildDataSchema(typeName string, allSchemas map[string]*SchemaInfo) map[string]any {
	// 处理带包名的 slice 类型，如 account.[]*MemberInfo
	if pkgName, elemType, ok := strings.Cut(typeName, ".[]"); ok {
		elemType = strings.TrimPrefix(elemType, "*") // 去除指针前缀

		// 判断是否是基本类型
		if isBasicType(elemType) {
			return map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": getOpenAPIType(elemType),
				},
			}
		}

		// 复杂类型，使用 $ref
		// elemType 可能已经包含包名（如 account.MemberInfo），也可能不包含
		schemaName := elemType
		if !strings.Contains(elemType, ".") {
			schemaName = pkgName + "." + elemType
		}

		// 检查 schema 是否存在，如果不存在则使用基本类型
		if _, exists := allSchemas[schemaName]; !exists {
			// 尝试不带包名前缀
			if _, exists2 := allSchemas[elemType]; exists2 {
				schemaName = elemType
			} else {
				// schema 不存在，返回基本 object 类型
				return map[string]any{
					"type":  "array",
					"items": missingSchema(schemaName),
				}
			}
		}

		return map[string]any{
			"type": "array",
			"items": map[string]any{
				"$ref": "#/components/schemas/" + schemaName,
			},
		}
	}

	// 处理不带包名的 slice 类型，如 []*MemberInfo 或 []string
	if after, ok := strings.CutPrefix(typeName, "[]"); ok {
		elemType := after
		elemType = strings.TrimPrefix(elemType, "*") // 去除指针前缀

		// 判断是否是基本类型
		if isBasicType(elemType) {
			return map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": getOpenAPIType(elemType),
				},
			}
		}

		// 复杂类型，使用 $ref
		schemaName := strings.ReplaceAll(elemType, ".", "_")

		// 检查 schema 是否存在
		if _, exists := allSchemas[schemaName]; !exists {
			// schema 不存在，返回基本 object 类型
			return map[string]any{
				"type":  "array",
				"items": missingSchema(schemaName),
			}
		}

		return map[string]any{
			"type": "array",
			"items": map[string]any{
				"$ref": "#/components/schemas/" + schemaName,
			},
		}
	}

	// 处理 map 类型
	if strings.HasPrefix(typeName, "map[") {
		return map[string]any{
			"type":        "object",
			"description": "Map 类型: " + typeName,
		}
	}

	// 处理指针类型
	typeName = strings.TrimPrefix(typeName, "*")

	// 判断是否是基本类型
	if isBasicType(typeName) {
		return map[string]any{
			"type": getOpenAPIType(typeName),
		}
	}

	// 复杂类型，使用 $ref
	// 注意：schema 名称可能包含 "."，如 "account.MemberInfo"
	schemaName := typeName // 保持原始的包名.类型名格式

	// 检查 schema 是否存在
	if _, exists := allSchemas[schemaName]; !exists {
		// schema 不存在，返回基本 object 类型
		return missingSchema(schemaName)
	}

	return map[string]any{
		"$ref": "#/components/schemas/" + schemaName,
	}
}

// normalizeSchemaName 将 Go import 别名映射为可读的 schema 名称
func normalizeSchemaName(s string) string {
	// 移除指针和数组前缀
	s = strings.TrimPrefix(s, "*")
	s = strings.TrimPrefix(s, "[]")
	s = strings.TrimPrefix(s, "*")

	if alias, name, ok := strings.Cut(s, "."); ok {
		if normalized := normalizeAliasToDir(alias); normalized != "" {
			return normalized + "." + name
		}
	}
	return s
}

// externalSchemas 无法从仓库源码解析的第三方类型（如 libgo 的类型别名）的手写描述。
// 字段需与第三方定义保持一致，升级依赖时同步核对。
var externalSchemas = map[string]map[string]any{
	"caddy.Handler": {"type": "object"},
	"psutil.DetailStat": {
		"type":        "object",
		"description": "系统资源明细，其他字段见 overview 接口文档",
		"required":    []string{"cpu"},
		"properties": map[string]any{
			"cpu":          map[string]any{"$ref": "#/components/schemas/psutil.CPUStat"},
			"cpuCore":      map[string]any{"type": "integer"},
			"cpuCoreLogic": map[string]any{"type": "integer"},
			"cpuPercent":   map[string]any{"type": "array", "items": map[string]any{"type": "number"}},
			"netInterface": map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/psutil.NetInterface"}},
		},
	},
	"psutil.NetInterface": {
		"type":     "object",
		"required": []string{"name", "bytesRecv", "bytesSent"},
		"properties": map[string]any{
			"name":      map[string]any{"type": "string"},
			"bytesRecv": map[string]any{"type": "integer", "description": "累计接收字节数"},
			"bytesSent": map[string]any{"type": "integer", "description": "累计发送字节数"},
			"dropin":    map[string]any{"type": "integer"},
			"dropout":   map[string]any{"type": "integer"},
			"ipv4List":  map[string]any{"type": "array", "nullable": true, "items": map[string]any{"type": "string"}},
			"ipv6List":  map[string]any{"type": "array", "nullable": true, "items": map[string]any{"type": "string"}},
		},
	},
	"psutil.CPUStat": {
		"type":     "object",
		"required": []string{"models", "packages", "physicalCores", "logicalCPUs", "processors"},
		"properties": map[string]any{
			"models":        map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "非空型号去重，不表示封装数"},
			"packages":      map[string]any{"type": "integer", "nullable": true, "description": "系统报告的逻辑处理器所涉及的封装数，不含空插槽和不可见封装；拓扑不完整时未知"},
			"physicalCores": map[string]any{"type": "integer", "nullable": true, "description": "物理核心总数；无法确认时未知"},
			"logicalCPUs":   map[string]any{"type": "integer", "nullable": true, "description": "系统报告的逻辑处理器总数；未知为 null"},
			"processors":    map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/psutil.CPUProcessor"}, "description": "Linux 可识别的逻辑处理器；其他平台或无数据时为空数组"},
		},
	},
	"psutil.CPUProcessor": {
		"type":     "object",
		"required": []string{"id", "packageId", "coreId", "model", "vendor"},
		"properties": map[string]any{
			"id":        map[string]any{"type": "integer", "description": "系统逻辑处理器 ID"},
			"packageId": map[string]any{"type": "string", "nullable": true, "description": "系统封装 ID；0 有效，未知为 null"},
			"coreId":    map[string]any{"type": "string", "nullable": true, "description": "封装内物理核心 ID；须与 packageId 一起使用，未知为 null"},
			"model":     map[string]any{"type": "string"},
			"vendor":    map[string]any{"type": "string"},
		},
	},
	// github.com/rehiy/libgo/webssh.ListResult
	"webssh.SFTPListResult": {
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{"type": "string", "description": "实际目录路径"},
			"files": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name":       map[string]any{"type": "string"},
						"size":       map[string]any{"type": "integer"},
						"mode":       map[string]any{"type": "string"},
						"modTime":    map[string]any{"type": "integer"},
						"isDir":      map[string]any{"type": "boolean"},
						"isLink":     map[string]any{"type": "boolean"},
						"linkTarget": map[string]any{"type": "string"},
					},
				},
			},
		},
	},
}

// missingSchema 为找不到定义的类型生成降级 schema：
// 自由结构（gin.H、map、json.RawMessage）按任意对象处理，已登记的第三方类型使用手写描述，
// 其余保留 "Schema not found" 提示，便于发现真正遗漏的类型。
func missingSchema(schemaName string) map[string]any {
	if ext, ok := externalSchemas[schemaName]; ok {
		return ext
	}
	_, name, _ := strings.Cut(schemaName, ".")
	if schemaName == "gin.H" || schemaName == "json.RawMessage" || name == "map" || strings.HasPrefix(name, "map[") {
		return map[string]any{
			"type":                 "object",
			"description":          "任意 JSON 对象",
			"additionalProperties": true,
		}
	}
	return map[string]any{
		"type":        "object",
		"description": "Schema not found: " + schemaName,
	}
}

// basicTypes 基本类型集合（用于 isBasicType 快速判断）
var basicTypes = map[string]bool{
	"string": true, "bool": true, "byte": true, "rune": true,
	"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
	"float32": true, "float64": true,
}

// isBasicType 判断是否是 Go 基本类型
func isBasicType(typeName string) bool {
	return basicTypes[typeName]
}

// getOpenAPIType 将 Go 类型转换为 OpenAPI 类型
func getOpenAPIType(goType string) string {
	switch goType {
	case "string":
		return "string"
	case "int", "int8", "int16", "int32", "uint", "uint8", "uint16", "uint32":
		return "integer"
	case "int64", "uint64":
		return "integer"
	case "float32", "float64":
		return "number"
	case "bool":
		return "boolean"
	default:
		return "string"
	}
}
