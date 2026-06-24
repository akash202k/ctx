package parse

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// GoParser implements Parser for Go source files using go/ast (no CGo).
type GoParser struct{}

func NewGoParser() *GoParser { return &GoParser{} }

func (p *GoParser) Extensions() []string { return []string{".go"} }

func (p *GoParser) Parse(path, content string) (ParsedFile, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, content, parser.AllErrors)
	if err != nil {
		// Return partial results on parse error so we don't drop the file from the graph.
		pf := ParsedFile{Path: path}
		if f != nil {
			pf.Package = f.Name.Name
		}
		return pf, nil
	}

	pf := ParsedFile{
		Path:    path,
		Package: f.Name.Name,
	}

	// Collect imports.
	for _, imp := range f.Imports {
		raw := strings.Trim(imp.Path.Value, `"`)
		pf.Imports = append(pf.Imports, raw)
	}

	// Collect top-level symbols and call references.
	callSet := make(map[string]bool)
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			sym := extractFuncSymbol(fset, d)
			pf.Symbols = append(pf.Symbols, sym)
			// Walk function body for call expressions.
			if d.Body != nil {
				ast.Inspect(d.Body, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}
					ref := callRef(call)
					if ref != "" {
						callSet[ref] = true
					}
					return true
				})
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					pf.Symbols = append(pf.Symbols, Symbol{
						Name:      s.Name.Name,
						Signature: fmt.Sprintf("type %s %s", s.Name.Name, typeString(s.Type)),
						Kind:      "type",
					})
				case *ast.ValueSpec:
					kind := "var"
					if d.Tok == token.CONST {
						kind = "const"
					}
					for _, name := range s.Names {
						if ast.IsExported(name.Name) {
							pf.Symbols = append(pf.Symbols, Symbol{
								Name:      name.Name,
								Signature: fmt.Sprintf("%s %s", kind, name.Name),
								Kind:      kind,
							})
						}
					}
				}
			}
		}
	}

	for ref := range callSet {
		pf.Calls = append(pf.Calls, ref)
	}
	return pf, nil
}

func (p *GoParser) ExtractSignatures(path, content string) ([]Symbol, error) {
	fset := token.NewFileSet()
	// Parse declarations only (skip function bodies).
	f, err := parser.ParseFile(fset, path, content, parser.SkipObjectResolution)
	if err != nil && f == nil {
		return nil, err
	}

	var syms []Symbol
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if ast.IsExported(d.Name.Name) {
				syms = append(syms, extractFuncSymbol(fset, d))
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if ast.IsExported(s.Name.Name) {
						syms = append(syms, Symbol{
							Name:      s.Name.Name,
							Signature: fmt.Sprintf("type %s %s", s.Name.Name, typeString(s.Type)),
							Kind:      "type",
						})
					}
				case *ast.ValueSpec:
					kind := "var"
					if d.Tok == token.CONST {
						kind = "const"
					}
					for _, name := range s.Names {
						if ast.IsExported(name.Name) {
							syms = append(syms, Symbol{
								Name:      name.Name,
								Signature: fmt.Sprintf("%s %s", kind, name.Name),
								Kind:      kind,
							})
						}
					}
				}
			}
		}
	}
	return syms, nil
}

// extractFuncSymbol builds a Symbol from a FuncDecl. It includes receiver type
// and parameter/return types but not the body.
func extractFuncSymbol(fset *token.FileSet, d *ast.FuncDecl) Symbol {
	var sb strings.Builder
	sb.WriteString("func ")
	if d.Recv != nil && len(d.Recv.List) > 0 {
		sb.WriteString("(")
		sb.WriteString(fieldListString(d.Recv))
		sb.WriteString(") ")
	}
	sb.WriteString(d.Name.Name)
	sb.WriteString("(")
	if d.Type.Params != nil {
		sb.WriteString(fieldListString(d.Type.Params))
	}
	sb.WriteString(")")
	if d.Type.Results != nil && len(d.Type.Results.List) > 0 {
		sb.WriteString(" ")
		if len(d.Type.Results.List) > 1 {
			sb.WriteString("(")
		}
		sb.WriteString(fieldListString(d.Type.Results))
		if len(d.Type.Results.List) > 1 {
			sb.WriteString(")")
		}
	}
	return Symbol{
		Name:      d.Name.Name,
		Signature: sb.String(),
		Kind:      "func",
	}
}

func fieldListString(fl *ast.FieldList) string {
	if fl == nil {
		return ""
	}
	var parts []string
	for _, f := range fl.List {
		t := typeString(f.Type)
		if len(f.Names) == 0 {
			parts = append(parts, t)
		} else {
			var names []string
			for _, n := range f.Names {
				names = append(names, n.Name)
			}
			parts = append(parts, strings.Join(names, ", ")+" "+t)
		}
	}
	return strings.Join(parts, ", ")
}

func typeString(expr ast.Expr) string {
	if expr == nil {
		return ""
	}
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + typeString(t.X)
	case *ast.SelectorExpr:
		return typeString(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + typeString(t.Elt)
		}
		return "[...]" + typeString(t.Elt)
	case *ast.MapType:
		return "map[" + typeString(t.Key) + "]" + typeString(t.Value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.StructType:
		return "struct{...}"
	case *ast.FuncType:
		return "func(...)"
	case *ast.ChanType:
		return "chan " + typeString(t.Value)
	case *ast.Ellipsis:
		return "..." + typeString(t.Elt)
	case *ast.IndexExpr:
		return typeString(t.X) + "[" + typeString(t.Index) + "]"
	default:
		return fmt.Sprintf("%T", expr)
	}
}

// callRef extracts a "pkg.Symbol" or "Symbol" string from a call expression,
// used as a best-effort cross-file call reference.
func callRef(call *ast.CallExpr) string {
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		return fn.Name
	case *ast.SelectorExpr:
		if id, ok := fn.X.(*ast.Ident); ok {
			return id.Name + "." + fn.Sel.Name
		}
	}
	return ""
}
