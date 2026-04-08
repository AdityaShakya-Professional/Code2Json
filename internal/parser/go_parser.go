package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// ParseGoStruct parses Go source and returns a JSON-serializable map
// built from the first exported struct found. Nested structs in the same file are resolved.
func ParseGoStruct(src string) (interface{}, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.AllErrors)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Go source: %v", err)
	}

	// Collect all struct type declarations
	structs := map[string]*ast.StructType{}
	var primaryName string

	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}
			structs[ts.Name.Name] = st
			if primaryName == "" && ts.Name.IsExported() {
				primaryName = ts.Name.Name
			}
		}
	}

	if primaryName == "" {
		// Try any struct
		for name := range structs {
			primaryName = name
			break
		}
	}
	if primaryName == "" {
		return nil, fmt.Errorf("no struct found in Go source")
	}

	result := buildGoStructMap(primaryName, structs, map[string]bool{})
	return result, nil
}

func buildGoStructMap(name string, structs map[string]*ast.StructType, visited map[string]bool) map[string]interface{} {
	if visited[name] {
		return map[string]interface{}{}
	}
	visited[name] = true

	st, ok := structs[name]
	if !ok {
		return map[string]interface{}{}
	}

	result := map[string]interface{}{}
	for _, field := range st.Fields.List {
		// Get JSON tag name if present, else use field name
		fieldKey := ""
		if field.Tag != nil {
			tag := strings.Trim(field.Tag.Value, "`")
			fieldKey = extractJSONTag(tag)
		}

		for _, ident := range field.Names {
			key := fieldKey
			if key == "" || key == "-" {
				if key == "-" {
					continue
				}
				key = ident.Name
			}
			result[key] = goDefaultValue(field.Type, structs, visited)
		}

		// Embedded struct (anonymous field)
		if len(field.Names) == 0 {
			embedded := goDefaultValue(field.Type, structs, visited)
			if m, ok := embedded.(map[string]interface{}); ok {
				for k, v := range m {
					result[k] = v
				}
			}
		}
	}
	return result
}

func extractJSONTag(tag string) string {
	for _, part := range strings.Split(tag, " ") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, `json:"`) {
			val := strings.TrimPrefix(part, `json:"`)
			val = strings.TrimSuffix(val, `"`)
			name := strings.Split(val, ",")[0]
			return name
		}
	}
	return ""
}

func goDefaultValue(expr ast.Expr, structs map[string]*ast.StructType, visited map[string]bool) interface{} {
	switch t := expr.(type) {
	case *ast.Ident:
		return goIdentDefault(t.Name, structs, visited)
	case *ast.StarExpr:
		return goDefaultValue(t.X, structs, visited)
	case *ast.ArrayType:
		elem := goDefaultValue(t.Elt, structs, visited)
		return []interface{}{elem}
	case *ast.MapType:
		return map[string]interface{}{}
	case *ast.SelectorExpr:
		// e.g. time.Time, sql.NullString
		pkg := fmt.Sprintf("%v", t.X)
		sel := t.Sel.Name
		return goSelectorDefault(pkg, sel)
	case *ast.StructType:
		// Anonymous inline struct
		tmp := buildGoStructMap("__inline__", map[string]*ast.StructType{"__inline__": t}, visited)
		return tmp
	case *ast.InterfaceType:
		return nil
	case *ast.ChanType:
		return nil
	case *ast.FuncType:
		return nil
	}
	return nil
}

func goIdentDefault(name string, structs map[string]*ast.StructType, visited map[string]bool) interface{} {
	switch name {
	case "string":
		return ""
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "byte", "rune":
		return 0
	case "float32", "float64":
		return 0.0
	case "bool":
		return false
	case "complex64", "complex128":
		return nil
	case "error":
		return nil
	}
	// Locally defined struct
	if _, ok := structs[name]; ok {
		return buildGoStructMap(name, structs, copyVisited(visited))
	}
	return nil
}

func goSelectorDefault(pkg, sel string) interface{} {
	switch pkg + "." + sel {
	case "time.Time", "time.Duration":
		return "0001-01-01T00:00:00Z"
	case "sql.NullString":
		return map[string]interface{}{"String": "", "Valid": false}
	case "sql.NullInt64", "sql.NullInt32":
		return map[string]interface{}{"Int64": 0, "Valid": false}
	case "sql.NullFloat64":
		return map[string]interface{}{"Float64": 0.0, "Valid": false}
	case "sql.NullBool":
		return map[string]interface{}{"Bool": false, "Valid": false}
	case "uuid.UUID":
		return "00000000-0000-0000-0000-000000000000"
	case "decimal.Decimal":
		return "0"
	}
	return nil
}

func copyVisited(v map[string]bool) map[string]bool {
	c := map[string]bool{}
	for k, val := range v {
		c[k] = val
	}
	return c
}
