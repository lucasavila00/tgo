package enums

import (
	"go/token"
	"strconv"

	"github.com/lucasavila00/tgo/internal/ast"
	"github.com/lucasavila00/tgo/internal/checker/types"
)

// Declarations replaces one checked enum with its Go storage and public API.
func Declarations(
	specification *ast.TypeSpec, enum *types.Enum, payloads []*ast.StructType, tagType ast.Expr,
	fresh func() *ast.Ident,
) []ast.Decl {
	name := specification.Name.Name
	storage := &ast.StructType{Fields: &ast.FieldList{}}
	underlying := enum.Named().Underlying().(*types.Struct)
	for index := 0; index < underlying.NumFields(); index++ {
		field := underlying.Field(index)
		storage.Fields.List = append(storage.Fields.List, namedField(field.Name(), storageType(field.Type())))
	}
	specificationCopy := *specification
	specificationCopy.Type = storage
	declarations := []ast.Decl{
		typeDeclaration(&specificationCopy),
		typeDeclaration(&ast.TypeSpec{Name: ast.NewIdent(name + "Tag"), Type: tagType}),
		tagConstants(enum),
		tagMethod(enum, fresh()),
	}
	for index, variant := range enum.Variants() {
		declarations = append(declarations,
			typeDeclaration(&ast.TypeSpec{Name: ast.NewIdent(variant.Type().Obj().Name()), Type: payloads[index]}),
			constructor(variant, payloads[index], fresh),
			accessor(variant, fresh()),
		)
	}
	return declarations
}

// storageType names a private field type selected by the layout pass.
func storageType(typ types.Type) ast.Expr {
	switch typ := typ.(type) {
	case *types.Named:
		return ast.NewIdent(typ.Obj().Name())
	case *types.Basic:
		return ast.NewIdent(typ.Name())
	case *types.Interface:
		return &ast.InterfaceType{Methods: &ast.FieldList{}}
	case *types.Array:
		return &ast.ArrayType{Len: integer(int(typ.Len())), Elt: &ast.FuncType{Params: &ast.FieldList{}}}
	default:
		panic("unexpected enum storage field")
	}
}

// tagConstants gives each declared variant its stable declaration-order number.
func tagConstants(enum *types.Enum) *ast.GenDecl {
	name := enum.Named().Obj().Name()
	declaration := &ast.GenDecl{Tok: token.CONST, Lparen: token.Pos(1)}
	for _, variant := range enum.Variants() {
		declaration.Specs = append(declaration.Specs, &ast.ValueSpec{
			Names:  []*ast.Ident{ast.NewIdent(name + "Tag" + variant.Name())},
			Type:   ast.NewIdent(name + "Tag"),
			Values: []ast.Expr{integer(variant.Index())},
		})
	}
	return declaration
}

// constructor copies its arguments into fresh storage for the selected variant.
func constructor(variant *types.EnumVariant, payload *ast.StructType, fresh func() *ast.Ident) *ast.FuncDecl {
	enumName := variant.Enum().Named().Obj().Name()
	parameters := &ast.FieldList{}
	payloadValue := &ast.CompositeLit{Type: ast.NewIdent(variant.Type().Obj().Name())}
	index := 0
	for _, field := range payload.Fields.List {
		count := len(field.Names)
		if count == 0 {
			count = 1
		}
		for range count {
			parameter := fresh().Name
			parameters.List = append(parameters.List, namedField(parameter, field.Type))
			payloadValue.Elts = append(payloadValue.Elts, ast.NewIdent(parameter))
			index++
		}
	}
	value := &ast.CompositeLit{Type: ast.NewIdent(enumName), Elts: []ast.Expr{
		&ast.KeyValueExpr{Key: ast.NewIdent("tgoTag"), Value: ast.NewIdent(enumName + "Tag" + variant.Name())},
	}}
	if index != 0 {
		field := "tgoVariant" + strconv.Itoa(variant.Index())
		if variant.Boxed() {
			field = "tgoPayload"
		}
		value.Elts = append(value.Elts, &ast.KeyValueExpr{Key: ast.NewIdent(field), Value: payloadValue})
	}
	return &ast.FuncDecl{
		Name: ast.NewIdent("New" + enumName + variant.Name()),
		Type: &ast.FuncType{Params: parameters, Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent(enumName)}}}},
		Body: &ast.BlockStmt{List: []ast.Stmt{returnValue(value)}},
	}
}

// accessor copies the active payload from its selected private field.
func accessor(variant *types.EnumVariant, receiver *ast.Ident) *ast.FuncDecl {
	payloadType := ast.NewIdent(variant.Type().Obj().Name())
	var value ast.Expr = &ast.CompositeLit{Type: payloadType}
	if variant.Type().Underlying().(*types.Struct).NumFields() != 0 {
		value = selectField(receiver.Name, "tgoVariant"+strconv.Itoa(variant.Index()))
		if variant.Boxed() {
			value = &ast.TypeAssertExpr{X: selectField(receiver.Name, "tgoPayload"), Type: payloadType}
		}
	}
	return method(receiver, variant.Enum().Named().Obj().Name(), variant.Name()+"Payload", payloadType,
		[]ast.Stmt{returnValue(value)})
}

// typeDeclaration wraps one type specification in a Go declaration.
func typeDeclaration(specification *ast.TypeSpec) *ast.GenDecl {
	return &ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{specification}}
}

// method builds a value receiver method with one result.
func method(receiver *ast.Ident, receiverType, name string, result ast.Expr, body []ast.Stmt) *ast.FuncDecl {
	return &ast.FuncDecl{
		Recv: &ast.FieldList{List: []*ast.Field{namedField(receiver.Name, ast.NewIdent(receiverType))}},
		Name: ast.NewIdent(name),
		Type: &ast.FuncType{Params: &ast.FieldList{}, Results: &ast.FieldList{List: []*ast.Field{{Type: result}}}},
		Body: &ast.BlockStmt{List: body},
	}
}

// namedField makes one named parameter, receiver, or storage field.
func namedField(name string, typ ast.Expr) *ast.Field {
	return &ast.Field{Names: []*ast.Ident{ast.NewIdent(name)}, Type: typ}
}

// selectField builds a field selection on a generated local.
func selectField(receiver, field string) *ast.SelectorExpr {
	return &ast.SelectorExpr{X: ast.NewIdent(receiver), Sel: ast.NewIdent(field)}
}

// returnValue builds a one-result return statement.
func returnValue(value ast.Expr) *ast.ReturnStmt {
	return &ast.ReturnStmt{Results: []ast.Expr{value}}
}

// integer builds a decimal integer literal.
func integer(value int) *ast.BasicLit {
	return &ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(value)}
}

// tagMethod reads the active variant from a copied enum value.
func tagMethod(enum *types.Enum, receiver *ast.Ident) *ast.FuncDecl {
	name := enum.Named().Obj().Name()
	return method(receiver, name, "Tag", ast.NewIdent(name+"Tag"),
		[]ast.Stmt{returnValue(selectField(receiver.Name, "tgoTag"))})
}
