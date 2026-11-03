package main

import (
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"strings"
)

var (
	routes      []RouteDef
	inlineFuncs = map[string]*ast.FuncLit{} // 内联 handler 闭包，键为 inline:<id>
)

type RouteDef struct {
	Method      string // GET/POST/PUT/PATCH/DELETE/ANY
	Path        string // /overview/probe
	Label       string // 中文描述
	Module      string // 模块名
	Handler     string // 函数名，如 app.overviewProbe；内联 handler 为 inline:<id>
	HandlerFile string // handler 所在 ctrl 文件

	// 请求信息
	JSONBody    *SchemaInfo // ShouldBindJSON 类型
	QueryType   *SchemaInfo // ShouldBindQuery 类型
	FormData    bool        // multipart/form-data
	FormFields  []ParamDef  // c.PostForm / c.GetPostForm / c.FormFile 字段
	PathParams  []ParamDef  // c.Param 参数
	QueryParams []ParamDef  // c.Query / c.DefaultQuery 参数
	IsWS        bool        // WebSocket 路由
	IsSSE       bool        // SSE 路由

	// 响应信息
	ResponseBodyType string // 响应类型的 Go 类型字符串（用于 slice/map 等复杂类型）
	RawContentType   string // 非 JSON 封装的文本响应媒体类型
}

type ParamDef struct {
	Name     string
	Type     string // string, integer, boolean
	Required bool
}

func parseCtrlFile(filename string) {
	f := parseFile(filename)
	if f == nil {
		fmt.Fprintf(os.Stderr, "解析 %s 失败\n", filename)
		return
	}

	ast.Inspect(f, func(n ast.Node) bool {
		funcDecl, ok := n.(*ast.FuncDecl)
		if !ok || funcDecl.Body == nil {
			return true
		}

		// 匹配 define*Routes() 方法
		if !strings.HasPrefix(funcDecl.Name.Name, "define") || !strings.HasSuffix(funcDecl.Name.Name, "Routes") {
			return true
		}

		// 在函数体中查找返回语句中的 []Route 复合字面量
		ast.Inspect(funcDecl.Body, func(n2 ast.Node) bool {
			retStmt, ok := n2.(*ast.ReturnStmt)
			if !ok {
				return true
			}
			for _, expr := range retStmt.Results {
				composite, ok := expr.(*ast.CompositeLit)
				if !ok {
					continue
				}
				// 检查是否是 []Route
				arrType, ok := composite.Type.(*ast.ArrayType)
				if !ok {
					continue
				}
				sel, ok := arrType.Elt.(*ast.Ident)
				if !ok || sel.Name != "Route" {
					continue
				}

				// 解析数组中的每个 Route 字面量
				for _, elt := range composite.Elts {
					cl, ok := elt.(*ast.CompositeLit)
					if !ok {
						continue
					}
					route := parseRouteLiteral(cl)
					route.HandlerFile = filename
					// AG-UI 是 CopilotKit 的内部流式协议，契约见 docs/references/copilot.md。
					// 不加入通用 REST 文档，避免被当作普通 JSON 接口调用。
					if route.Path == "/copilot/agui" {
						continue
					}
					if route.Method != "" {
						routes = append(routes, route)
					}
				}
			}
			return true
		})
		return true
	})
}

// ─── 辅助函数 ───

// getCtrlFiles 从目录 entries 中筛选 handler 文件并返回完整路径列表
func getCtrlFiles(dir string, entries []os.DirEntry) []string {
	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), cfg.CtrlPrefix) && strings.HasSuffix(entry.Name(), ".go") {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}
	return files
}

var inlineSeq int // 内联 handler 序号

func parseRouteLiteral(cl *ast.CompositeLit) RouteDef {
	var r RouteDef
	for _, elt := range cl.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}
		switch key.Name {
		case "Method":
			if lit, ok := kv.Value.(*ast.BasicLit); ok {
				r.Method = strings.Trim(lit.Value, `"`)
			}
		case "Path":
			if lit, ok := kv.Value.(*ast.BasicLit); ok {
				r.Path = strings.Trim(lit.Value, `"`)
			}
		case "Label":
			if lit, ok := kv.Value.(*ast.BasicLit); ok {
				r.Label = strings.Trim(lit.Value, `"`)
			}
		case "Module":
			if lit, ok := kv.Value.(*ast.BasicLit); ok {
				r.Module = strings.Trim(lit.Value, `"`)
			}
		case "Handler":
			handler := kv.Value
			if call, ok := handler.(*ast.CallExpr); ok {
				handler = call.Fun
			}
			if sel, ok := handler.(*ast.SelectorExpr); ok {
				if x, ok := sel.X.(*ast.Ident); ok {
					r.Handler = x.Name + "." + sel.Sel.Name
					continue
				}
			}
			if fn, ok := kv.Value.(*ast.FuncLit); ok {
				inlineSeq++
				id := fmt.Sprintf("inline:%d", inlineSeq)
				inlineFuncs[id] = fn
				r.Handler = id
			}
		}
	}
	return r
}
