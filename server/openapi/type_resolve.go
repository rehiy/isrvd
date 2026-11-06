package openapi

import (
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// resolveCallReturnType 解析函数调用的返回类型
func (g *generator) resolveCallReturnType(call *ast.CallExpr, filename string) string {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}

	methodName := sel.Sel.Name

	// 处理链式调用: app.accountSvc.Login
	if recvSel, ok := sel.X.(*ast.SelectorExpr); ok {
		svcVarName := recvSel.Sel.Name
		pkgName := inferPkgFromVar(filename, svcVarName)
		if pkgName != "" {
			// findMethodReturnType 已经处理了 import 别名解析和 pkgName 前缀
			return g.findMethodReturnType(filename, pkgName, methodName)
		}
	}

	return ""
}

// findMethodReturnType 在 service 包中查找方法的返回类型
func (g *generator) findMethodReturnType(ctrlFile, pkgName, methodName string) string {
	searchDir := filepath.Join(g.root, g.cfg.ServiceDir, pkgName)
	entries, err := os.ReadDir(searchDir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		filename := filepath.Join(searchDir, entry.Name())
		retType := g.findMethodReturnTypeInFile(filename, methodName)
		if retType != "" {
			// 解析返回类型中的 import 别名
			retType = g.resolveReturnType(filename, pkgName, retType)
			return retType
		}
	}
	return ""
}

// findMethodReturnTypeInFile 在单个文件中查找方法的返回类型
func (g *generator) findMethodReturnTypeInFile(filename, methodName string) string {
	f := g.parseFile(filename)
	if f == nil {
		return ""
	}

	for _, decl := range f.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Name.Name != methodName || fd.Type.Results == nil {
			continue
		}
		for _, field := range fd.Type.Results.List {
			goType := typeExprToString(field.Type)
			if goType == "error" {
				continue
			}
			if goType == "bool" {
				continue
			}
			// 过滤掉 any、interface{} 等空接口类型
			if goType == "any" || goType == "interface{}" {
				continue
			}
			// 去除所有指针前缀
			goType = strings.TrimLeft(goType, "*")
			return goType
		}
	}
	return ""
}

// resolveReturnType 将方法返回类型解析为最终的 schema 名称格式
// 处理 slice、指针、import 别名等情况
// 输入如: []pkgApisix.Consumer, *LoginResponse, []*config.PasskeyCredential
// 输出如: apisix.[]Consumer, account.LoginResponse, config.[]PasskeyCredential
func (g *generator) resolveReturnType(svcFilename, pkgName, retType string) string {
	// 提取 slice 前缀
	slicePrefix := ""
	inner := retType
	if strings.HasPrefix(inner, "[]") {
		slicePrefix = "[]"
		inner = inner[2:]
	}
	// 去指针
	inner = strings.TrimPrefix(inner, "*")

	// 解析 import 别名
	if strings.Contains(inner, ".") {
		inner = g.resolveImportAlias(svcFilename, inner)
	} else {
		// 不包含 "."，说明是本包类型，添加 pkgName 前缀
		inner = pkgName + "." + inner
	}

	// 重新组装: pkgName.[]TypeName 格式
	if slicePrefix != "" {
		// 将 "pkg.TypeName" → "pkg.[]TypeName"
		if before, after, ok := strings.CutLast(inner, "."); ok {
			return before + "." + slicePrefix + after
		}
		return slicePrefix + inner
	}

	return inner
}

// ─── 辅助函数 ───

// resolveImportAlias 解析导入别名，将别名.类型名 转换为实际的包名.类型名
// 例如: pkgApisix.Consumer → apisix.Consumer
func (g *generator) resolveImportAlias(filename, typeName string) string {
	// 检查是否包含 "."，且第一部分可能是导入别名
	alias, suffix, found := strings.Cut(typeName, ".")
	if !found {
		return typeName
	}
	typeSuffix := "." + suffix // 包含 "."

	// 从文件中提取导入别名映射
	importMap := g.extractImportAliases(filename)

	// 如果别名在映射中，替换为实际的包名
	if realPkg, ok := importMap[alias]; ok {
		return realPkg + typeSuffix
	}

	return typeName
}

// extractImportAliases 从 Go 文件中提取导入别名映射
// 返回 map[别名]实际包名
func (g *generator) extractImportAliases(filename string) map[string]string {
	result := make(map[string]string)

	f := g.parseFile(filename)
	if f == nil {
		return result
	}

	for _, imp := range f.Imports {
		// 导入路径，如 "isrvd/pkgs/apisix"
		importPath := strings.Trim(imp.Path.Value, "\"")

		// 提取包名（路径最后一段）
		parts := strings.Split(importPath, "/")
		realPkgName := parts[len(parts)-1]

		if imp.Name != nil {
			// 有别名: pkgApisix "isrvd/pkgs/apisix"
			alias := imp.Name.Name
			result[alias] = realPkgName
		} else {
			// 无别名: "isrvd/pkgs/apisix"
			result[realPkgName] = realPkgName
		}
	}

	return result
}

// inferPkgFromVar 从变量名推断包名
func inferPkgFromVar(filename, varName string) string {
	if before, ok := strings.CutSuffix(varName, "Svc"); ok {
		return strings.ToLower(before)
	}
	return ""
}

func (g *generator) collectLocalTypes(f *ast.File) map[string]string {
	result := make(map[string]string)
	typeAliases := make(map[string]string) // 本地别名 → 原始类型

	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			typeName := ts.Name.Name
			if ts.Assign.IsValid() {
				// type X = Y 类型别名
				collectTypeAlias(ts, typeAliases)
			} else if st, ok := ts.Type.(*ast.StructType); ok {
				result[typeName] = structTypeToJSONSchema(st)
			}
		}
	}

	// 将别名信息也保存（用特殊前缀标记）
	for alias, target := range typeAliases {
		result["__alias__"+alias] = target
	}
	return result
}

// collectTypeAlias 从 TypeSpec 中收集类型别名
func collectTypeAlias(ts *ast.TypeSpec, typeAliases map[string]string) {
	typeName := ts.Name.Name
	if sel, ok := ts.Type.(*ast.SelectorExpr); ok {
		typeAliases[typeName] = fullSelName(sel)
	} else if ident, ok := ts.Type.(*ast.Ident); ok {
		typeAliases[typeName] = ident.Name
	}
}

func fullSelName(sel *ast.SelectorExpr) string {
	if ident, ok := sel.X.(*ast.Ident); ok {
		return ident.Name + "." + sel.Sel.Name
	}
	return sel.Sel.Name
}

func typeExprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return fullSelName(t)
	case *ast.StarExpr:
		return "*" + typeExprToString(t.X)
	case *ast.ArrayType:
		return "[]" + typeExprToString(t.Elt)
	case *ast.MapType:
		return "map[" + typeExprToString(t.Key) + "]" + typeExprToString(t.Value)
	default:
		return fmt.Sprintf("%T", expr)
	}
}
