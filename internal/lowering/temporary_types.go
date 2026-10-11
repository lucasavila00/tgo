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
	localNames := make(map[string]string)
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
				localNames[t.Obj().Name()] = r.parameterAlias(t).Name
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
	var rename func(ast.Expr)
	fields := func(list *ast.FieldList) {
		if list != nil {
			for _, field := range list.List {
				rename(field.Type)
			}
		}
	}
	rename = func(expr ast.Expr) {
		switch e := expr.(type) {
		case *ast.SelectorExpr:
			if id, ok := e.X.(*ast.Ident); ok && id.Name == "unsafe" && unsafeAlias != nil {
				id.Name = unsafeAlias.Name
			}
		case *ast.Ident:
			if name := localNames[e.Name]; name != "" {
				e.Name = name
			}
		case *ast.StarExpr:
			rename(e.X)
		case *ast.ArrayType:
			rename(e.Elt)
		case *ast.MapType:
			rename(e.Key)
			rename(e.Value)
		case *ast.ChanType:
			rename(e.Value)
		case *ast.StructType:
			fields(e.Fields)
		case *ast.InterfaceType:
			fields(e.Methods)
		case *ast.FuncType:
			fields(e.Params)
			fields(e.Results)
		case *ast.IndexExpr:
			rename(e.X)
			rename(e.Index)
		case *ast.IndexListExpr:
			rename(e.X)
			for _, index := range e.Indices {
				rename(index)
			}
		}
	}
	rename(result)
	return result
}
