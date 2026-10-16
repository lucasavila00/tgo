package shared

import (
	"go/token"

	"github.com/lucasavila00/tgo/internal/ast"
	"github.com/lucasavila00/tgo/internal/checker/types"
	"github.com/lucasavila00/tgo/internal/lowering/enums"
)

// enumDeclarations replaces source enum types with storage and public Go functions.
func (r *Context) enumDeclarations(file *ast.File) *ast.File {
	output := *file
	output.Decls = nil
	for _, declaration := range file.Decls {
		group, ok := declaration.(*ast.GenDecl)
		if !ok || group.Tok != token.TYPE || !hasEnumDeclaration(group) {
			output.Decls = append(output.Decls, declaration)
			continue
		}
		for _, specification := range group.Specs {
			typeSpec := specification.(*ast.TypeSpec)
			if _, ok := typeSpec.Type.(*ast.EnumType); !ok {
				output.Decls = append(output.Decls, &ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{typeSpec}})
				continue
			}
			enum := types.EnumOf(r.input.Inventory.Checked.TypesInfo.Defs[typeSpec.Name].Type())
			payloads := make([]*ast.StructType, len(enum.Variants()))
			for index, variant := range enum.Variants() {
				payloads[index] = variant.Declaration().Type
				if payloads[index] == nil {
					payloads[index] = &ast.StructType{Fields: &ast.FieldList{}}
				}
			}
			output.Decls = append(output.Decls, enums.Declarations(typeSpec, enum, payloads, r.Name)...)
		}
	}
	return &output
}

// hasEnumDeclaration reports whether a type group needs generated declarations.
func hasEnumDeclaration(group *ast.GenDecl) bool {
	for _, specification := range group.Specs {
		if _, ok := specification.(*ast.TypeSpec).Type.(*ast.EnumType); ok {
			return true
		}
	}
	return false
}

// enumLiteral evaluates the written fields once before calling the public constructor.
func (r *Context) enumLiteral(source *ast.CompositeLit, payload *ast.CompositeLit, variant *types.EnumVariant) Result {
	constructor := r.enumMember(variant, source, source.Type, "New"+variant.Enum().Named().Obj().Name()+variant.Name())
	call := &ast.CallExpr{Fun: constructor}
	fields := variant.Type().Underlying().(*types.Struct)
	var before []ast.Stmt
	if fields.NumFields() != 0 {
		payload.Type = r.enumMember(variant, source, source.Type, variant.Type().Obj().Name())
		value := r.NameFor("payload")
		before = append(before, Variable(value, payload.Type, payload))
		used := false
		for index := 0; index < fields.NumFields(); index++ {
			field := fields.Field(index)
			if field.Name() == "_" {
				zero := r.NameFor("zero")
				before = append(before, Variable(zero, r.TemporaryType(field.Type(), source), nil))
				call.Args = append(call.Args, zero)
			} else {
				used = true
				call.Args = append(call.Args, &ast.SelectorExpr{X: value, Sel: ast.NewIdent(field.Name())})
			}
		}
		if !used {
			before = append(before, &ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent("_")},
				Tok: token.ASSIGN, Rhs: []ast.Expr{value}})
		}
	}
	result := r.NameFor("enum")
	typeName := ast.Unparen(source.Type).(*ast.SelectorExpr).X
	before = append(before, Variable(result, typeName, call))
	r.recordTemporary(result, variant.Enum().Named(), source)
	r.CompleteExpression(source)
	return r.ResultOf(before, []ast.Expr{result})
}

// enumMember retains a written package qualifier or creates a safe generated reference.
func (r *Context) enumMember(variant *types.EnumVariant, source, selected ast.Expr, name string) ast.Expr {
	selector := ast.Unparen(selected).(*ast.SelectorExpr)
	if enumSelector, ok := ast.Unparen(selector.X).(*ast.SelectorExpr); ok {
		if packageName, ok := enumSelector.X.(*ast.Ident); ok {
			if _, imported := r.input.Inventory.Checked.TypesInfo.Uses[packageName].(*types.PkgName); imported {
				return &ast.SelectorExpr{X: packageName, Sel: ast.NewIdent(name)}
			}
		}
	}
	enum := variant.Enum().Named()
	if enum.Obj().Pkg() != r.input.Inventory.Checked.Types {
		reference := r.TemporaryType(enum, source).(*ast.SelectorExpr)
		return &ast.SelectorExpr{X: reference.X, Sel: ast.NewIdent(name)}
	}
	object := enum.Obj().Pkg().Scope().Lookup(name)
	if r.VisibleBinding(source, name) == object {
		return ast.NewIdent(name)
	}
	alias := r.NameFor(name)
	if _, isType := object.(*types.TypeName); isType {
		r.file.extraDeclarations = append(r.file.extraDeclarations, &ast.GenDecl{Tok: token.TYPE,
			Specs: []ast.Spec{&ast.TypeSpec{Name: alias, Assign: token.Pos(1), Type: ast.NewIdent(name)}}})
	} else {
		r.file.extraDeclarations = append(r.file.extraDeclarations, &ast.GenDecl{Tok: token.VAR,
			Specs: []ast.Spec{&ast.ValueSpec{Names: []*ast.Ident{alias}, Values: []ast.Expr{ast.NewIdent(name)}}}})
	}
	return alias
}
