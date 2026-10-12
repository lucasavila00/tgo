package lowering

import (
	"fmt"
	"github.com/lucasavila00/tgo/internal/checker/types"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
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
	if typ := r.functionState.temporaryTypes[expr]; typ != nil {
		return typ
	}
	if origin := r.functionState.origins[expr]; origin != nil {
		expr = origin
	}
	return r.input.source.inventory.Checked.TypesInfo.TypeOf(expr)
}

// temporaryType names the checked type without using a hidden source binding.
func (r *rewrite) temporaryType(typ types.Type, original ast.Expr) ast.Expr {
	return r.temporaryTypeAt(typ, original, original.Pos())
}

func (r *rewrite) temporaryTypeAt(typ types.Type, original ast.Expr, position token.Pos) ast.Expr {
	nameCheck := r.checkTypeNames(typ, position)
	if nameCheck.problem != "" {
		if r.request.err == nil {
			errorPosition := r.temporaryTypeErrorPosition(original)
			r.request.err = fmt.Errorf("%s: cannot name type %s in a generated typed temporary: %s; expose the type with an accessible name", errorPosition, typ, nameCheck.problem)
		}
		return ast.NewIdent("invalid")
	}
	text := types.TypeString(types.Default(typ), func(pkg *types.Package) string {
		if pkg == r.input.source.inventory.Checked.Types {
			return ""
		}
		if r.file.typeImports == nil {
			r.file.typeImports = make(map[*types.Package]*ast.Ident)
		}
		alias := r.file.typeImports[pkg]
		if alias == nil {
			alias = r.nameFor(pkg.Name())
			r.file.typeImports[pkg] = alias
			r.file.extraImports = append(r.file.extraImports, &ast.ImportSpec{Name: alias, Path: &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(pkg.Path())}})
		}
		return alias.Name
	})
	var unsafeAlias *ast.Ident
	if nameCheck.unsafePointer {
		unsafeAlias = r.nameFor("unsafe")
		r.file.extraImports = append(r.file.extraImports, &ast.ImportSpec{Name: unsafeAlias, Path: &ast.BasicLit{Kind: token.STRING, Value: `"unsafe"`}})
	}
	result, err := parser.ParseExpr(text)
	if err != nil {
		panic(err)
	}
	// Parsed type positions do not refer to the source file set.
	ast.Inspect(result, func(node ast.Node) bool {
		if node == nil {
			return true
		}
		value := reflect.ValueOf(node).Elem()
		for i := 0; i < value.NumField(); i++ {
			field := value.Field(i)
			if field.Type() == reflect.TypeFor[token.Pos]() {
				field.SetInt(0)
			}
		}
		return true
	})
	r.renameTypeBindings(typ, result, nameCheck.parameterNames, unsafeAlias)
	return result
}

func (r *rewrite) renameTypeBindings(typ types.Type, result ast.Expr, parameterNames map[*types.TypeParam]string, unsafeAlias *ast.Ident) {
	// Follow checked types so equal spellings do not merge different bindings.
	var rename func(types.Type, ast.Expr)

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
			renameTypeArguments(t, expr, rename)
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
			renameTupleFields(t.Params(), e.Params, rename)
			renameTupleFields(t.Results(), e.Results, rename)
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
}

func renameTypeArguments(typ types.Type, expr ast.Expr, rename func(types.Type, ast.Expr)) {
	var args *types.TypeList
	if named, ok := typ.(*types.Named); ok {
		args = named.TypeArgs()
	} else {
		args = typ.(*types.Alias).TypeArgs()
	}
	switch e := expr.(type) {
	case *ast.IndexExpr:
		rename(args.At(0), e.Index)
	case *ast.IndexListExpr:
		for i, index := range e.Indices {
			rename(args.At(i), index)
		}
	}
}

func renameTupleFields(tuple *types.Tuple, list *ast.FieldList, rename func(types.Type, ast.Expr)) {
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
