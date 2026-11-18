package web

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"
)

// PairingRoster reads the current character sources instead of the registry
// compiled into the running server.
func PairingRoster(source fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		kinds, err := readPairingRoster(source)
		if err != nil {
			log.Printf("pairing roster: %v", err)
			http.Error(w, "无法读取战斗机名单", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(kinds)
	}
}

func readPairingRoster(source fs.FS) ([]string, error) {
	set := token.NewFileSet()
	all, err := parseCharacterFile(set, source, "all.go", parser.ImportsOnly)
	if err != nil {
		return nil, err
	}
	kinds := []string{}
	seen := map[string]bool{"靶子": true}
	for _, imp := range all.Imports {
		importPath, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			return nil, err
		}
		dir, ok := strings.CutPrefix(importPath, "xqdj/character/")
		if !ok {
			continue
		}
		packKinds, err := readPackFighters(set, source, dir)
		if err != nil {
			return nil, err
		}
		for _, kind := range packKinds {
			if !seen[kind] {
				seen[kind] = true
				kinds = append(kinds, kind)
			}
		}
	}
	return kinds, nil
}

func parseCharacterFile(set *token.FileSet, source fs.FS, name string, mode parser.Mode) (*ast.File, error) {
	data, err := fs.ReadFile(source, name)
	if err != nil {
		return nil, err
	}
	return parser.ParseFile(set, name, data, mode|parser.SkipObjectResolution)
}

func readPackFighters(set *token.FileSet, source fs.FS, dir string) ([]string, error) {
	entries, err := fs.ReadDir(source, dir)
	if err != nil {
		return nil, err
	}
	// Character packages register unit.Spec literals with Fighter: true.
	// Resolve their Kind strings from the same package, including across files.
	constants := make(map[string]ast.Expr)
	var kindExprs []ast.Expr
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parseCharacterFile(set, source, path.Join(dir, name), 0)
		if err != nil {
			return nil, err
		}
		for _, decl := range file.Decls {
			if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.CONST {
				for _, spec := range gen.Specs {
					value := spec.(*ast.ValueSpec)
					for i, name := range value.Names {
						if i < len(value.Values) {
							constants[name.Name] = value.Values[i]
						}
					}
				}
			}
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return true
			}
			register, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || register.Sel.Name != "Register" {
				return true
			}
			spec, ok := call.Args[0].(*ast.CompositeLit)
			if !ok {
				return true
			}
			var fighter bool
			var kind ast.Expr
			for _, elt := range spec.Elts {
				field, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				key, ok := field.Key.(*ast.Ident)
				if !ok {
					continue
				}
				switch key.Name {
				case "Fighter":
					value, _ := field.Value.(*ast.Ident)
					fighter = value != nil && value.Name == "true"
				case "Kind":
					kind = field.Value
				}
			}
			if fighter {
				kindExprs = append(kindExprs, kind)
			}
			return true
		})
	}
	var kinds []string
	for _, expr := range kindExprs {
		if id, ok := expr.(*ast.Ident); ok {
			expr = constants[id.Name]
		}
		lit, ok := expr.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return nil, fmt.Errorf("%s: fighter Kind must be a string literal or a local string constant", dir)
		}
		kind, err := strconv.Unquote(lit.Value)
		if err != nil {
			return nil, err
		}
		kinds = append(kinds, kind)
	}
	return kinds, nil
}
