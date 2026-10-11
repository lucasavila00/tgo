package compilerv2

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
)

func (s *Source) lowerPropagation(site Site, operation propagation) (map[string][]byte, error) {
	return s.insert(site, nil, &operation)
}

// prepareReturns obtains typed zero results without spelling their types in the body.
func (r *rewrite) prepareReturns(original *ast.FuncType) (*ast.FuncType, []ast.Stmt) {
	if original.Results == nil {
		panic("propagation requires an error return")
	}
	copy := *original
	results := *original.Results
	results.List = make([]*ast.Field, len(original.Results.List))
	var prefix []ast.Stmt
	for i, field := range original.Results.List {
		clone := *field
		clone.Names = append([]*ast.Ident(nil), field.Names...)
		if len(clone.Names) == 0 {
			clone.Names = []*ast.Ident{r.name()}
		}
		for j, name := range clone.Names {
			if name.Name == "_" {
				name = r.name()
				clone.Names[j] = name
			}
			value := r.name()
			prefix = append(prefix, &ast.AssignStmt{Lhs: []ast.Expr{value}, Tok: token.DEFINE, Rhs: []ast.Expr{name}})
			r.returnValues = append(r.returnValues, value)
		}
		results.List[i] = &clone
	}
	copy.Results = &results
	return &copy, prefix
}

func (r *rewrite) propagatedCall(marker *ast.CallExpr) ([]ast.Stmt, []ast.Expr) {
	raw := marker.Args[0]
	before, values := r.save(raw)
	if len(values) == 0 {
		panic("propagation requires an error result")
	}
	function := r.returnFunction
	var signature *types.Signature
	switch fn := function.(type) {
	case *ast.FuncDecl:
		signature = r.source.Package.TypesInfo.Defs[fn.Name].Type().(*types.Signature)
	case *ast.FuncLit:
		signature = r.source.Package.TypesInfo.TypeOf(fn).(*types.Signature)
	}
	last := signature.Results().At(signature.Results().Len() - 1).Type()
	if _, ok := last.Underlying().(*types.Interface); !ok || !types.AssignableTo(last, types.Universe.Lookup("error").Type()) {
		panic("propagation requires a final error interface return")
	}
	errorValue := r.returnValues[len(r.returnValues)-1]
	before = append(before, &ast.AssignStmt{Lhs: []ast.Expr{errorValue}, Tok: token.ASSIGN, Rhs: values[len(values)-1:]})
	var failure []ast.Stmt
	if r.propagation.wrap {
		wrapped := &ast.CallExpr{Fun: &ast.SelectorExpr{X: r.formatAlias, Sel: ast.NewIdent("Errorf")}, Args: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(fmt.Sprintf("%s: %%w", r.propagation.callName))}, errorValue}}
		failure = append(failure, &ast.AssignStmt{Lhs: []ast.Expr{errorValue}, Tok: token.ASSIGN, Rhs: []ast.Expr{wrapped}})
	}
	failure = append(failure, &ast.ReturnStmt{Results: r.returnValues})
	before = append(before, &ast.IfStmt{Cond: &ast.BinaryExpr{X: errorValue, Op: token.NEQ, Y: ast.NewIdent("nil")}, Body: &ast.BlockStmt{List: failure}})
	r.inserted = true
	return before, values[:len(values)-1]
}

func (s *Source) unreachable(expr ast.Expr) bool {
	for node := ast.Node(expr); s.parents[node] != nil; node = s.parents[node] {
		binary, ok := s.parents[node].(*ast.BinaryExpr)
		if !ok || node != binary.Y {
			continue
		}
		value := s.Package.TypesInfo.Types[binary.X].Value
		if value == nil {
			continue
		}
		if (binary.Op == token.LAND && value.String() == "false") || (binary.Op == token.LOR && value.String() == "true") {
			return true
		}
	}
	return false
}

func removeUnusedMarkers(source *Source, operations map[string]propagation) {
	used := make(map[string]bool)
	for _, file := range source.Package.Syntax {
		ast.Inspect(file, func(node ast.Node) bool {
			if id, ok := node.(*ast.Ident); ok && source.Package.TypesInfo.Uses[id] != nil {
				used[id.Name] = true
			}
			return true
		})
	}
	for _, file := range source.Package.Syntax {
		declarations := file.Decls[:0]
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok {
				if _, marker := operations[fn.Name.Name]; marker && !used[fn.Name.Name] {
					continue
				}
			}
			declarations = append(declarations, decl)
		}
		file.Decls = declarations
		writeSyntax(source.Package.Fset, file)
	}
}
