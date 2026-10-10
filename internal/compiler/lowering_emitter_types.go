package compiler

import (
	"go/ast"
	"go/token"
	"go/types"
	"strconv"

	"golang.org/x/tools/go/ast/astutil"
)

func (e *loweringEmitter) contextTypeExpression(
	value plannedValue,
	output *ast.BlockStmt,
) ast.Expr {
	reference := value.typeReference
	if len(reference.blockers) != 0 {
		e.renameTypeReferenceBlockers(value)
		e.emitTypeAliasesBeforeBlockers(value)
		return e.typeExpression(value.typ, value.position)
	}
	if reference.direct {
		return e.typeExpression(value.typ, value.position)
	}
	typ := value.typ
	aliases := e.typeAliases[output]
	if aliases == nil {
		aliases = make(map[types.Type]*ast.Ident)
		e.typeAliases[output] = aliases
	}
	if alias := aliases[typ]; alias != nil {
		return ast.NewIdent(alias.Name)
	}
	alias := e.freshName("operandType")
	aliases[typ] = alias
	aliasPosition := output.Lbrace
	if aliasPosition == token.NoPos {
		aliasPosition = e.plan.function.body.Lbrace
	}
	typeExpression := e.typeExpression(typ, aliasPosition)
	declaration := &ast.DeclStmt{Decl: &ast.GenDecl{
		Tok: token.TYPE, Specs: []ast.Spec{&ast.TypeSpec{
			Name: alias, Assign: aliasPosition, Type: typeExpression,
		}},
	}}
	output.List = append([]ast.Stmt{declaration}, output.List...)
	return ast.NewIdent(alias.Name)
}

func (e *loweringEmitter) emitTypeAliasesBeforeBlockers(value plannedValue) {
	for _, planned := range value.typeReference.blockers {
		if !planned.typeName || e.typeObjectAliases[planned.intended] != nil {
			continue
		}
		if planned.alias.foreignQualified {
			e.ensureQualifiedTypeOwner(planned.intended.Pkg())
			continue
		}
		if planned.alias.packageScope {
			e.emitPackageTypeAlias(planned)
			continue
		}
		site, ok := e.typeDefinitionSites[planned.definition]
		if !ok {
			continue
		}
		alias := e.freshName(planned.intended.Name() + "Type")
		position := planned.alias.before.Pos()
		declaration := &ast.DeclStmt{Decl: &ast.GenDecl{
			Tok: token.TYPE, Specs: []ast.Spec{
				e.typeAliasSpecification(alias, planned.alias, position),
			},
		}}
		site.block.List = append(site.block.List, nil)
		copy(site.block.List[site.index+1:], site.block.List[site.index:])
		site.block.List[site.index] = declaration
		for definition, other := range e.typeDefinitionSites {
			if other.block == site.block && other.index >= site.index {
				other.index++
				e.typeDefinitionSites[definition] = other
			}
		}
		e.typeObjectAliases[planned.intended] = alias
	}
}

func (e *loweringEmitter) ensureQualifiedTypeOwner(owner *types.Package) {
	if e.forcedQualifiers[owner] != "" {
		return
	}
	alias := freshASTIdentifier(e.source.File, owner.Name())
	astutil.AddNamedImport(e.unit.fs, e.source.File, alias, owner.Path())
	e.forcedQualifiers[owner] = alias
	e.names[alias] = true
}

func (e *loweringEmitter) emitPackageTypeAlias(planned plannedTypeBlocker) {
	alias := e.freshName(planned.intended.Name() + "Type")
	position := e.source.File.Package
	specification := e.typeAliasSpecification(alias, planned.alias, position)
	declaration := &ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{specification}}
	index := 0
	for index < len(e.source.File.Decls) {
		imports, ok := e.source.File.Decls[index].(*ast.GenDecl)
		if !ok || imports.Tok != token.IMPORT {
			break
		}
		index++
	}
	e.source.File.Decls = append(e.source.File.Decls, nil)
	copy(e.source.File.Decls[index+1:], e.source.File.Decls[index:])
	e.source.File.Decls[index] = declaration
	e.typeObjectAliases[planned.intended] = alias
}

func (e *loweringEmitter) typeAliasSpecification(
	alias *ast.Ident,
	action *plannedTypeAliasAction,
	position token.Pos,
) *ast.TypeSpec {
	specification := &ast.TypeSpec{
		Name: alias, Assign: position, Type: e.typeObjectExpression(action.object, position),
	}
	if len(action.parameters) == 0 {
		return specification
	}
	parameters := make([]*ast.Field, 0, len(action.parameters))
	arguments := make([]ast.Expr, 0, len(action.parameters))
	for _, parameter := range action.parameters {
		name := ast.NewIdent(parameter.name)
		parameters = append(parameters, &ast.Field{
			Names: []*ast.Ident{name}, Type: e.typeExpression(parameter.constraint, position),
		})
		arguments = append(arguments, ast.NewIdent(name.Name))
	}
	specification.TypeParams = &ast.FieldList{List: parameters}
	specification.Type = &ast.IndexListExpr{X: specification.Type, Indices: arguments}
	return specification
}

func (e *loweringEmitter) typeObjectExpression(object types.Object, position token.Pos) ast.Expr {
	if object.Pkg() == nil {
		return e.unit.generatedUniverse(object.Name(), position)
	}
	if object.Pkg() == e.unit.typed && object.Parent() != e.unit.typed.Scope() {
		return ast.NewIdent(object.Name())
	}
	qualifier := e.typeOwnerQualifier(object.Pkg())
	return e.unit.generatedObject(
		qualifier, packagePath(object.Pkg()), object.Name(), position,
	)
}

func (e *loweringEmitter) typeOwnerQualifier(owner *types.Package) string {
	if qualifier := e.forcedQualifiers[owner]; qualifier != "" {
		return qualifier
	}
	return e.unit.ownerQualifier(e.source.File, owner)
}

func (e *loweringEmitter) renameTypeReferenceBlockers(value plannedValue) {
	for _, planned := range value.typeReference.blockers {
		if planned.typeName {
			continue
		}
		blocker := planned.object
		name := e.renamedTypeBlockers[blocker]
		if name == "" {
			name = e.freshName(blocker.Name() + "Value").Name
			e.renamedTypeBlockers[blocker] = name
		}
		for _, identifier := range planned.identifiers {
			identifier.Name = name
		}
	}
}

func (e *loweringEmitter) typeExpression(typ types.Type, position token.Pos) ast.Expr {
	if typ == nil {
		return nil
	}
	if _, alias := typ.(*types.Alias); alias {
		typ = types.Unalias(typ)
	}
	switch item := typ.(type) {
	case *types.Basic:
		return e.basicTypeExpression(item, position)
	case *types.Named:
		return e.namedTypeExpression(item, position)
	case *types.Pointer:
		return e.containerTypeExpression(item, position)
	case *types.Slice:
		return e.containerTypeExpression(item, position)
	case *types.Array:
		return e.containerTypeExpression(item, position)
	case *types.Map:
		return e.containerTypeExpression(item, position)
	case *types.Chan:
		return e.containerTypeExpression(item, position)
	case *types.TypeParam:
		return e.typeParameterExpression(item)
	case *types.Struct:
		return e.structTypeExpression(item, position)
	case *types.Interface:
		return e.interfaceTypeExpression(item, position)
	case *types.Union:
		return e.unionTypeExpression(item, position)
	case *types.Signature:
		return &ast.FuncType{
			Params:  e.tupleFields(item.Params(), item.Variadic(), position),
			Results: e.tupleFields(item.Results(), false, position),
		}
	}
	return nil
}

func (e *loweringEmitter) typeParameterExpression(item *types.TypeParam) ast.Expr {
	object := item.Obj()
	if alias := e.typeObjectAliases[object]; alias != nil {
		return ast.NewIdent(alias.Name)
	}
	return ast.NewIdent(object.Name())
}

func (e *loweringEmitter) containerTypeExpression(typ types.Type, position token.Pos) ast.Expr {
	switch item := typ.(type) {
	case *types.Pointer:
		return &ast.StarExpr{X: e.typeExpression(item.Elem(), position)}
	case *types.Slice:
		return &ast.ArrayType{Elt: e.typeExpression(item.Elem(), position)}
	case *types.Array:
		return &ast.ArrayType{
			Len: &ast.BasicLit{Kind: token.INT, Value: strconv.FormatInt(item.Len(), 10)},
			Elt: e.typeExpression(item.Elem(), position),
		}
	case *types.Map:
		return &ast.MapType{
			Key:   e.typeExpression(item.Key(), position),
			Value: e.typeExpression(item.Elem(), position),
		}
	case *types.Chan:
		direction := ast.SEND | ast.RECV
		if item.Dir() == types.SendOnly {
			direction = ast.SEND
		} else if item.Dir() == types.RecvOnly {
			direction = ast.RECV
		}
		return &ast.ChanType{Dir: direction, Value: e.typeExpression(item.Elem(), position)}
	}
	return nil
}

func (e *loweringEmitter) basicTypeExpression(item *types.Basic, position token.Pos) ast.Expr {
	if item.Info()&types.IsUntyped != 0 {
		return e.typeExpression(types.Default(item), position)
	}
	if alias := e.typeObjectAliases[types.Universe.Lookup(item.Name())]; alias != nil {
		return ast.NewIdent(alias.Name)
	}
	if item.Kind() == types.UnsafePointer {
		qualifier := e.unit.ownerQualifier(e.source.File, types.Unsafe)
		return e.unit.generatedObject(qualifier, types.Unsafe.Path(), "Pointer", position)
	}
	return e.unit.generatedUniverse(item.Name(), position)
}

func (e *loweringEmitter) namedTypeExpression(item *types.Named, position token.Pos) ast.Expr {
	object := item.Obj()
	var expression ast.Expr
	if alias := e.typeObjectAliases[object]; alias != nil {
		expression = ast.NewIdent(alias.Name)
	} else if object.Pkg() == e.unit.typed && object.Parent() != e.unit.typed.Scope() {
		expression = ast.NewIdent(object.Name())
	} else {
		qualifier := e.typeOwnerQualifier(object.Pkg())
		expression = e.unit.generatedObject(
			qualifier, packagePath(object.Pkg()), object.Name(), position,
		)
	}
	arguments := item.TypeArgs()
	if arguments == nil || arguments.Len() == 0 {
		return expression
	}
	indices := make([]ast.Expr, 0, arguments.Len())
	for index := range arguments.Len() {
		indices = append(indices, e.typeExpression(arguments.At(index), position))
	}
	return &ast.IndexListExpr{X: expression, Indices: indices}
}

func (e *loweringEmitter) structTypeExpression(
	item *types.Struct,
	position token.Pos,
) ast.Expr {
	fields := make([]*ast.Field, 0, item.NumFields())
	for index := range item.NumFields() {
		field := item.Field(index)
		var names []*ast.Ident
		if !field.Embedded() {
			names = []*ast.Ident{ast.NewIdent(field.Name())}
		}
		var tag *ast.BasicLit
		if text := item.Tag(index); text != "" {
			tag = &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(text)}
		}
		fields = append(fields, &ast.Field{
			Names: names, Type: e.typeExpression(field.Type(), position), Tag: tag,
		})
	}
	return &ast.StructType{Fields: &ast.FieldList{List: fields}}
}

func (e *loweringEmitter) interfaceTypeExpression(
	item *types.Interface,
	position token.Pos,
) ast.Expr {
	fields := make([]*ast.Field, 0, item.NumExplicitMethods()+item.NumEmbeddeds())
	for index := range item.NumExplicitMethods() {
		method := item.ExplicitMethod(index)
		fields = append(fields, &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(method.Name())},
			Type:  e.typeExpression(method.Type(), position),
		})
	}
	for index := range item.NumEmbeddeds() {
		fields = append(fields, &ast.Field{
			Type: e.typeExpression(item.EmbeddedType(index), position),
		})
	}
	return &ast.InterfaceType{Methods: &ast.FieldList{List: fields}}
}

func (e *loweringEmitter) unionTypeExpression(item *types.Union, position token.Pos) ast.Expr {
	var expression ast.Expr
	for index := range item.Len() {
		term := item.Term(index)
		termExpression := e.typeExpression(term.Type(), position)
		if term.Tilde() {
			termExpression = &ast.UnaryExpr{Op: token.TILDE, X: termExpression}
		}
		if expression == nil {
			expression = termExpression
			continue
		}
		expression = &ast.BinaryExpr{X: expression, Op: token.OR, Y: termExpression}
	}
	return expression
}

func (e *loweringEmitter) tupleFields(
	tuple *types.Tuple,
	variadic bool,
	position token.Pos,
) *ast.FieldList {
	if tuple == nil || tuple.Len() == 0 {
		return nil
	}
	fields := make([]*ast.Field, 0, tuple.Len())
	for index := range tuple.Len() {
		typ := e.typeExpression(tuple.At(index).Type(), position)
		if variadic && index == tuple.Len()-1 {
			if slice, ok := typ.(*ast.ArrayType); ok && slice.Len == nil {
				typ = &ast.Ellipsis{Elt: slice.Elt}
			}
		}
		fields = append(fields, &ast.Field{Type: typ})
	}
	return &ast.FieldList{List: fields}
}
