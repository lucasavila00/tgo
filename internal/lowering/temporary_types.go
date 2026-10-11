package lowering

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strconv"
)

func variable(name *ast.Ident, typ ast.Expr, value ast.Expr) ast.Stmt {
	spec := &ast.ValueSpec{Names: []*ast.Ident{name}, Type: typ}
	if value != nil {
		spec.Values = []ast.Expr{value}
	}
	return &ast.DeclStmt{Decl: &ast.GenDecl{Tok: token.VAR, Specs: []ast.Spec{spec}}}
}

func (r *rewrite) typeOf(expr ast.Expr) types.Type {
	if typ := r.temporaryTypes[expr]; typ != nil {
		return typ
	}
	if origin := r.origins[expr]; origin != nil {
		expr = origin
	}
	return r.source.Package.TypesInfo.TypeOf(expr)
}

// temporaryType names the checked type without using a hidden source binding.
func (r *rewrite) temporaryType(typ types.Type, original ast.Expr) ast.Expr {
	var problem string
	unsafePointer := false
	parameterNames := make(map[*types.TypeParam]string)
	var check func(types.Type)
	var checkName func(*types.TypeName)
	check = func(typ types.Type) {
		switch t := typ.(type) {
		case *types.Basic:
			if t.Kind() == types.UnsafePointer {
				unsafePointer = true
				return
			}
			checkName(types.Universe.Lookup(types.Default(t).String()).(*types.TypeName))
		case *types.TypeParam:
			scope := r.source.Package.Types.Scope().Innermost(original.Pos())
			_, resolved := scope.LookupParent(t.Obj().Name(), original.Pos())
			if resolved != t.Obj() {
				parameterNames[t] = r.parameterAlias(t).Name
			}
		case *types.Named:
			checkName(t.Obj())
			for i := 0; i < t.TypeArgs().Len(); i++ {
				check(t.TypeArgs().At(i))
			}
		case *types.Alias:
			checkName(t.Obj())
			for i := 0; i < t.TypeArgs().Len(); i++ {
				check(t.TypeArgs().At(i))
			}
		case *types.Pointer:
			check(t.Elem())
		case *types.Array:
			check(t.Elem())
		case *types.Slice:
			check(t.Elem())
		case *types.Map:
			check(t.Key())
			check(t.Elem())
		case *types.Chan:
			check(t.Elem())
		case *types.Signature:
			check(t.Params())
			check(t.Results())
		case *types.Tuple:
			for i := 0; i < t.Len(); i++ {
				check(t.At(i).Type())
			}
		case *types.Struct:
			for i := 0; i < t.NumFields(); i++ {
				field := t.Field(i)
				if field.Pkg() != nil && field.Pkg() != r.source.Package.Types && !field.Exported() {
					problem = "the type has a private field from " + field.Pkg().Path()
				}
				check(field.Type())
			}
		case *types.Interface:
			for i := 0; i < t.NumExplicitMethods(); i++ {
				method := t.ExplicitMethod(i)
				if method.Pkg() != nil && method.Pkg() != r.source.Package.Types && !method.Exported() {
					problem = "the type has a private method from " + method.Pkg().Path()
				}
				check(method.Type())
			}
			for i := 0; i < t.NumEmbeddeds(); i++ {
				check(t.EmbeddedType(i))
			}
		}
	}
	// A named type needs its name, not the spelling of its underlying type.
	checkName = func(object *types.TypeName) {
		if object.Pkg() != nil && object.Pkg() != r.source.Package.Types {
			if !object.Exported() {
				problem = "the type is private to " + object.Pkg().Path()
			}
			return
		}
		scope := r.source.Package.Types.Scope().Innermost(original.Pos())
		_, resolved := scope.LookupParent(object.Name(), original.Pos())
		if resolved == object {
			return
		}
		if resolved != object {
			problem = "the source hides the type name " + object.Name()
		}
	}
	check(types.Default(typ))
	if problem != "" {
		if r.err == nil {
			position := r.source.Package.Fset.Position(original.Pos())
			var selected token.Pos
			var best int
			for expr, operation := range r.propagations {
				distance := int(expr.Pos() - original.Pos())
				if distance < 0 {
					distance = -distance
				}
				if operation.Position.IsValid() && r.frame != nil && contains(r.frame.function, expr) && (selected == 0 || distance < best || distance == best && expr.Pos() < selected) {
					position = operation.Position
					selected = expr.Pos()
					best = distance
				}
			}
			r.err = fmt.Errorf("%s: cannot name type %s in a generated typed temporary: %s; expose the type with an accessible name", position, typ, problem)
		}
		return ast.NewIdent("invalid")
	}
	text := types.TypeString(types.Default(typ), func(pkg *types.Package) string {
		if pkg == r.source.Package.Types {
			return ""
		}
		if r.typeImports == nil {
			r.typeImports = make(map[*types.Package]*ast.Ident)
		}
		alias := r.typeImports[pkg]
		if alias == nil {
			alias = r.name()
			r.typeImports[pkg] = alias
			r.extraImports = append(r.extraImports, &ast.ImportSpec{Name: alias, Path: &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(pkg.Path())}})
		}
		return alias.Name
	})
	var unsafeAlias *ast.Ident
	if unsafePointer {
		unsafeAlias = r.name()
		r.extraImports = append(r.extraImports, &ast.ImportSpec{Name: unsafeAlias, Path: &ast.BasicLit{Kind: token.STRING, Value: `"unsafe"`}})
	}
	result, err := parser.ParseExpr(text)
	if err != nil {
		panic(err)
	}
	// Follow checked types so equal spellings do not merge different bindings.
	var rename func(types.Type, ast.Expr)
	fields := func(tuple *types.Tuple, list *ast.FieldList) {
		if list == nil {
			return
		}
		for i, field := range list.List {
			expr := field.Type
			typ := tuple.At(i).Type()
			if ellipsis, ok := expr.(*ast.Ellipsis); ok {
				expr = ellipsis.Elt
				typ = typ.(*types.Slice).Elem()
			}
			rename(typ, expr)
		}
	}
	rename = func(typ types.Type, expr ast.Expr) {
		for {
			parenthesized, ok := expr.(*ast.ParenExpr)
			if !ok {
				break
			}
			expr = parenthesized.X
		}
		switch t := typ.(type) {
		case *types.TypeParam:
			if name := parameterNames[t]; name != "" {
				expr.(*ast.Ident).Name = name
			}
		case *types.Basic:
			if t.Kind() == types.UnsafePointer {
				expr.(*ast.SelectorExpr).X = unsafeAlias
			}
		case *types.Named, *types.Alias:
			var args *types.TypeList
			if named, ok := t.(*types.Named); ok {
				args = named.TypeArgs()
			} else {
				args = t.(*types.Alias).TypeArgs()
			}
			switch e := expr.(type) {
			case *ast.IndexExpr:
				rename(args.At(0), e.Index)
			case *ast.IndexListExpr:
				for i, index := range e.Indices {
					rename(args.At(i), index)
				}
			}
		case *types.Pointer:
			rename(t.Elem(), expr.(*ast.StarExpr).X)
		case *types.Array:
			rename(t.Elem(), expr.(*ast.ArrayType).Elt)
		case *types.Slice:
			rename(t.Elem(), expr.(*ast.ArrayType).Elt)
		case *types.Map:
			e := expr.(*ast.MapType)
			rename(t.Key(), e.Key)
			rename(t.Elem(), e.Value)
		case *types.Chan:
			rename(t.Elem(), expr.(*ast.ChanType).Value)
		case *types.Struct:
			for i, field := range expr.(*ast.StructType).Fields.List {
				rename(t.Field(i).Type(), field.Type)
			}
		case *types.Signature:
			e := expr.(*ast.FuncType)
			fields(t.Params(), e.Params)
			fields(t.Results(), e.Results)
		case *types.Interface:
			list := expr.(*ast.InterfaceType).Methods.List
			for i := 0; i < t.NumExplicitMethods(); i++ {
				rename(t.ExplicitMethod(i).Type(), list[i].Type)
			}
			for i := 0; i < t.NumEmbeddeds(); i++ {
				rename(t.EmbeddedType(i), list[t.NumExplicitMethods()+i].Type)
			}
		}
	}
	rename(types.Default(typ), result)
	return result
}
