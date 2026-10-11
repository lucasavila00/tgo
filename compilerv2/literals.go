package compilerv2

import (
	"go/ast"
	"go/token"
	"go/types"
)

// contextLiteral keeps an omitted literal type through its original container.
func (r *rewrite) contextLiteral(value, original ast.Expr) ast.Expr {
	literal, ok := original.(*ast.CompositeLit)
	if !ok || literal.Type != nil {
		return value
	}
	parent := r.source.parents[original]
	if pair, ok := parent.(*ast.KeyValueExpr); ok {
		parent = r.source.parents[pair]
	}
	container, ok := parent.(*ast.CompositeLit)
	if !ok {
		return value
	}
	typ := r.source.Package.TypesInfo.TypeOf(container).Underlying()
	pointer := false
	if p, ok := typ.(*types.Pointer); ok {
		typ = p.Elem().Underlying()
		pointer = true
	}
	wrapper := &ast.CompositeLit{Type: container.Type}
	var access ast.Expr
	switch t := typ.(type) {
	case *types.Struct:
		name := ""
		for i, element := range container.Elts {
			if element == original {
				name = t.Field(i).Name()
				break
			}
			if pair, ok := element.(*ast.KeyValueExpr); ok && (pair.Value == original || pair.Key == original) {
				name = pair.Key.(*ast.Ident).Name
				break
			}
		}
		wrapper.Elts = []ast.Expr{&ast.KeyValueExpr{Key: ast.NewIdent(name), Value: value}}
		access = &ast.SelectorExpr{Sel: ast.NewIdent(name)}
	case *types.Array, *types.Slice:
		wrapper.Elts = []ast.Expr{value}
		access = &ast.IndexExpr{Index: integer(0)}
	case *types.Map:
		pair := r.source.parents[original].(*ast.KeyValueExpr)
		if pair.Key == original {
			return value
		}
		key := r.zeroMapKey(container, pointer)
		wrapper.Elts = []ast.Expr{&ast.KeyValueExpr{Key: key, Value: value}}
		r.prepareCollectionImports()
		var mapValue ast.Expr = wrapper
		if container.Type == nil {
			mapValue = r.contextLiteral(wrapper, container)
		}
		if pointer {
			mapValue = &ast.StarExpr{X: mapValue}
		}
		values := &ast.CallExpr{Fun: &ast.SelectorExpr{X: r.mapAlias, Sel: ast.NewIdent("Values")}, Args: []ast.Expr{mapValue}}
		collected := &ast.CallExpr{Fun: &ast.SelectorExpr{X: r.sliceAlias, Sel: ast.NewIdent("Collect")}, Args: []ast.Expr{values}}
		return &ast.IndexExpr{X: collected, Index: integer(0)}
	default:
		return value
	}
	var wrapped ast.Expr = wrapper
	if container.Type == nil {
		wrapped = r.contextLiteral(wrapper, container)
	}
	if pointer {
		wrapped = &ast.StarExpr{X: wrapped}
	}
	wrapped = &ast.ParenExpr{X: wrapped}
	switch a := access.(type) {
	case *ast.SelectorExpr:
		a.X = wrapped
	case *ast.IndexExpr:
		a.X = wrapped
	}
	return access
}

func (r *rewrite) zeroMapKey(container *ast.CompositeLit, pointer bool) ast.Expr {
	r.prepareCollectionImports()
	empty := r.contextLiteral(&ast.CompositeLit{Type: container.Type}, container)
	if pointer {
		empty = &ast.StarExpr{X: empty}
	}
	keys := &ast.CallExpr{Fun: &ast.SelectorExpr{X: r.mapAlias, Sel: ast.NewIdent("Keys")}, Args: []ast.Expr{empty}}
	return r.sequenceZero(keys)
}

func (r *rewrite) prepareCollectionImports() {
	if r.mapAlias != nil {
		return
	}
	r.mapAlias, r.sliceAlias = r.name(), r.name()
	for _, spec := range []struct {
		name *ast.Ident
		path string
	}{{r.mapAlias, `"maps"`}, {r.sliceAlias, `"slices"`}} {
		r.extraImports = append(r.extraImports, &ast.ImportSpec{Name: spec.name, Path: &ast.BasicLit{Kind: token.STRING, Value: spec.path}})
	}
}

func (r *rewrite) snapshotLiteral(literal *ast.CompositeLit, original ast.Expr) ([]ast.Stmt, []ast.Expr) {
	copy := *literal
	copy.Elts = make([]ast.Expr, len(literal.Elts))
	var before []ast.Stmt
	for i, element := range literal.Elts {
		if pair, ok := element.(*ast.KeyValueExpr); ok {
			item := *pair
			setup, value := r.capture(pair.Value)
			before = append(before, setup...)
			item.Value = value
			copy.Elts[i] = &item
		} else {
			setup, value := r.capture(element)
			before = append(before, setup...)
			copy.Elts[i] = value
		}
	}
	if r.origins == nil {
		r.origins = make(map[ast.Expr]ast.Expr)
	}
	r.origins[&copy] = original
	return before, []ast.Expr{&copy}
}
