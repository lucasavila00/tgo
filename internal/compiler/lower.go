package compiler

import (
	"go/ast"
	"go/token"
	"go/types"
)

// prepare marks generated declarations and lowers constructors and defaults.
func (p *packageUnit) prepare() {
	p.generated = make(map[ast.Decl]bool)
	p.generatedValues = make(map[*ast.ValueSpec]bool)
	p.checkedLiterals = make(map[*ast.CompositeLit]bool)
	p.erasedImports = make(map[*ast.ImportSpec]bool)
	p.references = nil
	for _, source := range p.Sources {
		p.markGenerated(source)
		p.lowerSuccessReturns(source)
		p.addDefaults(source)
	}
}

// lowerConstructions resolves and lowers checked and enum literals.
func (p *packageUnit) lowerConstructions() {
	for _, source := range p.Sources {
		for _, declaration := range source.File.Decls {
			var exempt *model
			if function, ok := declaration.(*ast.FuncDecl); ok && function.Name.Name == "check" {
				if receiver, ok := receiverName(function); ok {
					candidate := p.Models[receiver]
					if candidate != nil && candidate.CheckedStruct {
						exempt = candidate
					}
				}
			}
			transform(declaration, func(node ast.Node) ast.Node {
				replacement := p.lowerConstruction(source.File, node, exempt)
				if replacement != node {
					source.Lowered = true
				}
				return replacement
			})
		}
	}
}

// generatedNames returns type and constructor names emitted for all models.
func generatedNames(models []*model) map[string]bool {
	names := make(map[string]bool)
	for _, model := range models {
		if len(model.Variants) > 0 {
			names[model.Name] = true
			names[model.Name+"Tag"] = true
			for _, variant := range model.Variants {
				names[model.Name+variant.Name] = true
				names[enumConstructorName(model.Name, variant.Name)] = true
				if len(variant.Fields) > 0 {
					names[enumCarrierName(model.Name, variant.Name)] = true
				}
			}
		}
	}
	return names
}

// markGenerated records declarations that source safety checks must skip.
func (p *packageUnit) markGenerated(source *source) {
	names := generatedNames(source.Models)
	for _, declaration := range source.File.Decls {
		switch node := declaration.(type) {
		case *ast.GenDecl:
			if len(node.Specs) != 1 {
				continue
			}
			spec, ok := node.Specs[0].(*ast.TypeSpec)
			if ok && names[spec.Name.Name] {
				p.generated[declaration] = true
			}
		case *ast.FuncDecl:
			if names[node.Name.Name] || source.GeneratedHelpers[node.Name.Name] ||
				generatedMethod(node, source.Models) {
				p.generated[declaration] = true
			}
		}
	}
}

// generatedMethod reports whether tgo emitted a method declaration.
func generatedMethod(function *ast.FuncDecl, models []*model) bool {
	receiver, ok := receiverName(function)
	if !ok {
		return false
	}

	for _, model := range models {
		if model.Enum && receiver == model.Name &&
			(function.Name.Name == "MarshalJSON" ||
				function.Name.Name == "MarshalJSONTo" ||
				function.Name.Name == "UnmarshalJSON" ||
				function.Name.Name == "UnmarshalJSONFrom") {
			return true
		}
		if generatedEnumMethod(receiver, function.Name.Name, model) {
			return true
		}
	}
	return false
}

// receiverName returns a simple named receiver type.
func receiverName(function *ast.FuncDecl) (string, bool) {
	if function.Recv == nil || len(function.Recv.List) != 1 {
		return "", false
	}
	receiver, ok := function.Recv.List[0].Type.(*ast.Ident)
	if ok {
		return receiver.Name, true
	}
	pointer, ok := function.Recv.List[0].Type.(*ast.StarExpr)
	if !ok {
		return "", false
	}
	receiver, ok = pointer.X.(*ast.Ident)
	if !ok {
		return "", false
	}
	return receiver.Name, true
}

// generatedEnumMethod recognizes tag and payload accessor methods.
func generatedEnumMethod(receiver, method string, model *model) bool {
	if len(model.Variants) == 0 {
		return false
	}
	if receiver != model.Name {
		return false
	}
	if method == "Tag" || method == "UnknownTag" {
		return true
	}
	if payloadFreeEnum(model) && (method == "GobEncode" || method == "GobDecode") {
		return true
	}
	for _, variant := range model.Variants {
		if method == variant.Name+"Payload" {
			return true
		}
	}
	return false
}

// lowerConstruction replaces protected literals with their validation calls.
func (p *packageUnit) lowerConstruction(
	file *ast.File,
	node ast.Node,
	exempt *model,
) ast.Node {
	literal, ok := node.(*ast.CompositeLit)
	if !ok {
		return node
	}
	typ := p.info.TypeOf(literal)
	owner, declaration := p.modelOwner(typ)
	if declaration == nil {
		owner, declaration = p.namedLiteralModel(literal.Type)
	}
	if declaration != nil && declaration.CheckedStruct {
		if owner == p && declaration == exempt {
			p.checkedLiterals[literal] = true
			return node
		}
		p.checkedLiterals[literal] = true
		return call(&ast.SelectorExpr{
			X: literal, Sel: &ast.Ident{NamePos: literal.End(), Name: "check"},
		})
	}
	selector, owner, declaration, variant := p.enumLiteral(literal)
	if declaration == nil {
		return node
	}
	typ = p.info.TypeOf(selector.X)
	named := types.Unalias(typ).(*types.Named)
	path := owner.Path
	p.markErasedOwnerImport(file, selector.X, named.Obj().Pkg())
	prefix := p.ownerQualifier(file, named.Obj().Pkg())
	return p.enumConstructorCall(
		literal, prefix, path, declaration, variant, selector.Sel.Pos(),
	)
}

func (p *packageUnit) enumLiteral(
	literal *ast.CompositeLit,
) (*ast.SelectorExpr, *packageUnit, *model, *variant) {
	selector, ok := literal.Type.(*ast.SelectorExpr)
	if !ok || !p.info.Types[selector.X].IsType() {
		return nil, nil, nil, nil
	}
	typ := p.info.TypeOf(selector.X)
	owner, declaration := p.modelOwner(typ)
	if declaration == nil || len(declaration.Variants) == 0 {
		return nil, nil, nil, nil
	}
	for index := range declaration.Variants {
		if selector.Sel.Name == declaration.Variants[index].Name {
			return selector, owner, declaration, &declaration.Variants[index]
		}
	}
	p.fail(literal, "unknown variant %s.%s", declaration.Name, selector.Sel.Name)
	return selector, owner, nil, nil
}

func (p *packageUnit) enumConstructorCall(
	literal *ast.CompositeLit,
	prefix string,
	path string,
	declaration *model,
	variant *variant,
	at token.Pos,
) ast.Expr {
	payloadObject := p.enumPayloadObject(path, declaration.Name+variant.Name)
	if payloadObject == nil {
		return literal
	}
	structure, ok := payloadObject.Type().Underlying().(*types.Struct)
	if !ok || structure.NumFields() != len(variant.Fields) {
		return literal
	}
	values, evaluation, indices, keyed, valid := p.enumLiteralValues(literal, structure)
	if !valid {
		return literal
	}
	constructor := p.generatedObject(
		prefix,
		path,
		enumConstructorName(declaration.Name, variant.Name),
		at,
	)
	if !keyed {
		return call(constructor, values...)
	}
	carrierNames := enumCarrierFieldNames(variant.Fields)
	carrier := enumCarrierName(declaration.Name, variant.Name)
	carrierElements := make([]ast.Expr, 0, len(evaluation))
	for index, value := range evaluation {
		carrierElements = append(carrierElements, &ast.KeyValueExpr{
			Key:   ast.NewIdent(carrierNames[indices[index]]),
			Value: value,
		})
	}
	inputName := freshASTIdentifier(literal, "input")
	arguments := make([]ast.Expr, len(values))
	for index := range arguments {
		arguments[index] = &ast.SelectorExpr{
			X: ast.NewIdent(inputName), Sel: ast.NewIdent(carrierNames[index]),
		}
	}
	return call(
		&ast.FuncLit{
			Type: &ast.FuncType{
				Params: &ast.FieldList{List: []*ast.Field{{
					Names: []*ast.Ident{ast.NewIdent(inputName)},
					Type:  p.generatedObject(prefix, path, carrier, literal.Lbrace),
				}}},
				Results: &ast.FieldList{List: []*ast.Field{{
					Type: p.generatedObject(
						prefix, path, declaration.Name, literal.Lbrace,
					),
				}}},
			},
			Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{
				Results: []ast.Expr{call(constructor, arguments...)},
			}}},
		},
		&ast.CompositeLit{
			Type: p.generatedObject(prefix, path, carrier, literal.Lbrace),
			Elts: carrierElements,
		},
	)
}

func (p *packageUnit) enumPayloadObject(path string, name string) *types.TypeName {
	owner := packageByPath(p.typed, path, make(map[*types.Package]bool))
	if owner == nil {
		return nil
	}
	object, _ := owner.Scope().Lookup(name).(*types.TypeName)
	return object
}

func (p *packageUnit) enumLiteralValues(
	literal *ast.CompositeLit,
	structure *types.Struct,
) ([]ast.Expr, []ast.Expr, []int, bool, bool) {
	values := make([]ast.Expr, structure.NumFields())
	evaluation := make([]ast.Expr, 0, len(literal.Elts))
	indices := make([]int, 0, len(literal.Elts))
	keyed := false
	unkeyed := false
	supplied := make(map[int]bool)
	for index, element := range literal.Elts {
		fieldIndex, value, pair := enumLiteralElement(structure, index, element)
		keyed = keyed || pair
		unkeyed = unkeyed || !pair
		if keyed && unkeyed {
			p.fail(literal, "mixture of field:value and value elements in struct literal")
			return nil, nil, nil, false, false
		}
		if fieldIndex < 0 || fieldIndex >= len(values) {
			p.fail(element, "unknown field in enum variant literal")
			return nil, nil, nil, false, false
		}
		if supplied[fieldIndex] {
			p.fail(
				element,
				"duplicate field %s in enum variant literal",
				structure.Field(fieldIndex).Name(),
			)
			return nil, nil, nil, false, false
		}
		supplied[fieldIndex] = true
		values[fieldIndex] = value
		evaluation = append(evaluation, value)
		indices = append(indices, fieldIndex)
	}
	for index, value := range values {
		if value == nil {
			p.fail(literal, "missing required field %s", structure.Field(index).Name())
			return nil, nil, nil, false, false
		}
	}
	return values, evaluation, indices, keyed, true
}

func enumLiteralElement(
	structure *types.Struct,
	index int,
	element ast.Expr,
) (int, ast.Expr, bool) {
	pair, ok := element.(*ast.KeyValueExpr)
	if !ok {
		return index, element, false
	}
	name, ok := pair.Key.(*ast.Ident)
	if !ok {
		return -1, pair.Value, true
	}
	for fieldIndex := 0; fieldIndex < structure.NumFields(); fieldIndex++ {
		if structure.Field(fieldIndex).Name() == name.Name {
			return fieldIndex, pair.Value, true
		}
	}
	return -1, pair.Value, true
}

// namedLiteralModel resolves a named literal when an outer marker blocks type information.
func (p *packageUnit) namedLiteralModel(expression ast.Expr) (*packageUnit, *model) {
	switch node := expression.(type) {
	case *ast.Ident:
		return p, p.Models[node.Name]
	case *ast.SelectorExpr:
		name, ok := node.X.(*ast.Ident)
		if !ok {
			return nil, nil
		}
		ownerName, ok := p.info.Uses[name].(*types.PkgName)
		if !ok {
			return nil, nil
		}
		owner := p.Imports[ownerName.Imported().Path()]
		if owner == nil {
			return nil, nil
		}
		return owner, owner.Models[node.Sel.Name]
	default:
		return nil, nil
	}
}
