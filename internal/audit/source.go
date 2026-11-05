package audit

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type contactRoot struct {
	node  ast.Node
	event string
}

type sourcePackage struct {
	set           *token.FileSet
	files         []*ast.File
	funcs         map[string]*ast.FuncDecl
	unitNames     map[string]bool
	damageAliases map[string]bool
}

// CheckSources follows package-local helpers from registered fighters' body
// Collision handlers. It deliberately does not claim general cross-event data
// flow analysis; engine-owned runtime origins provide a second independent guard.
func CheckSources(root string) error {
	var dirs []string
	err := filepath.WalkDir(filepath.Join(root, "character"), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == "testdata" || strings.HasPrefix(d.Name(), ".")) {
			return filepath.SkipDir
		}
		if d.IsDir() {
			dirs = append(dirs, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	sort.Strings(dirs)
	for _, dir := range dirs {
		if err := checkPackage(dir); err != nil {
			return err
		}
	}
	return nil
}

func MustCheckSources(root string) {
	if err := CheckSources(root); err != nil {
		panic(err)
	}
}

func checkPackage(dir string) error {
	p := &sourcePackage{set: token.NewFileSet(), funcs: map[string]*ast.FuncDecl{}, unitNames: map[string]bool{}, damageAliases: map[string]bool{}}
	paths, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return err
	}
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(p.set, path, nil, parser.ParseComments)
		if err != nil {
			return err
		}
		if strings.Contains(f.Doc.Text(), "go:build ignore") {
			continue
		}
		p.files = append(p.files, f)
		for _, im := range f.Imports {
			v, _ := strconv.Unquote(im.Path.Value)
			if v == "xqdj/internal/unit" {
				name := "unit"
				if im.Name != nil {
					name = im.Name.Name
				}
				p.unitNames[name] = true
			}
		}
		for _, decl := range f.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok {
				key := fn.Name.Name
				if fn.Recv != nil {
					key = receiverType(fn) + "." + key
				}
				p.funcs[key] = fn
			}
		}
	}
	fighters := map[string]bool{}
	for _, f := range p.files {
		ast.Inspect(f, func(n ast.Node) bool {
			if alias, ok := n.(*ast.TypeSpec); ok && alias.Assign.IsValid() && p.unitType(alias.Type, "Damage") {
				p.damageAliases[alias.Name.Name] = true
			}
			return true
		})
	}
	var registrationErr error
	for _, f := range p.files {
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) < 2 {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Register" {
				return true
			}
			spec, ok := call.Args[0].(*ast.CompositeLit)
			if !ok {
				registrationErr = fmt.Errorf("audit: cannot resolve registered Spec at %s; use an explicit unit.Spec literal", p.set.Position(call.Pos()))
				return true
			}
			if !p.unitType(spec.Type, "Spec") {
				registrationErr = fmt.Errorf("audit: cannot resolve registered Spec at %s; use an explicit unit.Spec literal", p.set.Position(call.Pos()))
				return true
			}
			fighter := false
			for _, e := range spec.Elts {
				if kv, ok := e.(*ast.KeyValueExpr); ok {
					key, _ := kv.Key.(*ast.Ident)
					if key != nil && ((key.Name == "Fighter" && isIdent(kv.Value, "true")) || (key.Name == "Role" && p.unitType(kv.Value, "RoleFighter"))) {
						fighter = true
					}
				}
			}
			if !fighter {
				return true
			}
			types := p.factoryTypes(call.Args[1], map[string]bool{})
			if len(types) == 0 {
				registrationErr = fmt.Errorf("audit: cannot resolve fighter factory at %s", p.set.Position(call.Pos()))
			}
			for _, typ := range types {
				fighters[typ] = true
			}
			return true
		})
	}
	if registrationErr != nil {
		return registrationErr
	}
	var names []string
	for name := range fighters {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		handle := p.funcs[name+".Handle"]
		if handle == nil {
			return fmt.Errorf("audit: cannot resolve %s.Handle in %s", name, dir)
		}
		var roots []contactRoot
		ast.Inspect(handle.Body, func(n ast.Node) bool {
			if c, ok := n.(*ast.CaseClause); ok {
				for _, typ := range c.List {
					if p.unitType(typ, "Collision") || p.unitType(typ, "FanHit") {
						for _, stmt := range c.Body {
							roots = append(roots, contactRoot{stmt, typ.(*ast.SelectorExpr).Sel.Name})
						}
					}
				}
			}
			if a, ok := n.(*ast.TypeAssertExpr); ok && (p.unitType(a.Type, "Collision") || p.unitType(a.Type, "FanHit")) {
				roots = append(roots, contactRoot{handle.Body, a.Type.(*ast.SelectorExpr).Sel.Name})
				return false
			}
			return true
		})
		for _, root := range roots {
			if err := p.trace(root.node, handle, []string{name + ".Handle[" + root.event + "]"}, map[string]bool{}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *sourcePackage) trace(node ast.Node, fn *ast.FuncDecl, chain []string, seen map[string]bool) error {
	var result error
	ast.Inspect(node, func(n ast.Node) bool {
		if result != nil {
			return false
		}
		if lit, ok := n.(*ast.CompositeLit); ok && p.damageType(lit.Type) {
			result = fmt.Errorf("audit: forbidden fighter body collision damage at %s\n  %s -> unit.Damage; move damage to an attack fan", p.set.Position(lit.Pos()), strings.Join(chain, " -> "))
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if p.damageType(call.Fun) {
			result = fmt.Errorf("audit: forbidden fighter body collision damage conversion at %s\n  %s", p.set.Position(call.Pos()), strings.Join(chain, " -> "))
			return false
		}
		key := ""
		switch target := call.Fun.(type) {
		case *ast.Ident:
			key = target.Name
		case *ast.SelectorExpr:
			if id, ok := target.X.(*ast.Ident); ok && fn.Recv != nil && len(fn.Recv.List[0].Names) > 0 && id.Name == fn.Recv.List[0].Names[0].Name {
				key = receiverType(fn) + "." + target.Sel.Name
			}
		}
		if next := p.funcs[key]; next != nil && !seen[key] {
			seen[key] = true
			result = p.trace(next.Body, next, append(append([]string(nil), chain...), key), seen)
		}
		return result == nil
	})
	return result
}

func (p *sourcePackage) factoryTypes(expr ast.Expr, seen map[string]bool) []string {
	var out []string
	if id, ok := expr.(*ast.Ident); ok && !seen[id.Name] {
		seen[id.Name] = true
		if fn := p.funcs[id.Name]; fn != nil {
			return p.factoryTypes(&ast.FuncLit{Type: fn.Type, Body: fn.Body}, seen)
		}
	}
	if lit, ok := expr.(*ast.FuncLit); ok && lit.Type.Results != nil {
		for _, r := range lit.Type.Results.List {
			t := r.Type
			if star, ok := t.(*ast.StarExpr); ok {
				t = star.X
			}
			if id, ok := t.(*ast.Ident); ok {
				out = append(out, id.Name)
			}
		}
	}
	ast.Inspect(expr, func(n ast.Node) bool {
		ret, ok := n.(*ast.ReturnStmt)
		if !ok {
			return true
		}
		for _, r := range ret.Results {
			if u, ok := r.(*ast.UnaryExpr); ok {
				r = u.X
			}
			if lit, ok := r.(*ast.CompositeLit); ok {
				if id, ok := lit.Type.(*ast.Ident); ok {
					out = append(out, id.Name)
				}
			}
			if c, ok := r.(*ast.CallExpr); ok {
				if id, ok := c.Fun.(*ast.Ident); ok && !seen[id.Name] {
					seen[id.Name] = true
					if fn := p.funcs[id.Name]; fn != nil {
						out = append(out, p.factoryTypes(&ast.FuncLit{Type: fn.Type, Body: fn.Body}, seen)...)
					}
				}
			}
		}
		return true
	})
	return out
}

func (p *sourcePackage) unitType(expr ast.Expr, name string) bool {
	s, ok := expr.(*ast.SelectorExpr)
	if !ok || s.Sel.Name != name {
		return false
	}
	id, ok := s.X.(*ast.Ident)
	return ok && p.unitNames[id.Name]
}

func (p *sourcePackage) damageType(expr ast.Expr) bool {
	if p.unitType(expr, "Damage") {
		return true
	}
	id, ok := expr.(*ast.Ident)
	return ok && p.damageAliases[id.Name]
}
func isIdent(expr ast.Expr, name string) bool {
	id, ok := expr.(*ast.Ident)
	return ok && id.Name == name
}
func receiverType(fn *ast.FuncDecl) string {
	t := fn.Recv.List[0].Type
	if ptr, ok := t.(*ast.StarExpr); ok {
		t = ptr.X
	}
	if id, ok := t.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}
