package compiler

import (
	"go/ast"
	"go/types"
)

// prepare marks generated declarations and lowers constructors and defaults.
func (p *packageUnit) prepare() {
	p.generated = make(map[ast.Decl]bool)
	p.generatedValues = make(map[*ast.ValueSpec]bool)
	p.checkedLiterals = make(map[*ast.CompositeLit]bool)
	p.erasedImports = make(map[*ast.ImportSpec]bool)
	p.references = nil
	p.usedIdentifiers = nil
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
			names[model.Name+"Tag"] = true
			for _, variant := range model.Variants {
				names[model.Name+variant.Name] = true
			}
		}
		if model.Predicate != "" {
			names["New"+model.Name] = true
			names["tgo"+model.Name+"Error"] = true
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
		if generatedCheckedMethod(receiver, function.Name.Name, model) {
			return true
		}
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

// generatedCheckedMethod recognizes checked-value support methods.
func generatedCheckedMethod(receiver, method string, model *model) bool {
	if model.Predicate == "" {
		return false
	}
	valueMethod := receiver == model.Name && method == "Value"
	errorMethod := receiver == "tgo"+model.Name+"Error" && method == "Error"
	return valueMethod || errorMethod
}

// generatedEnumMethod recognizes tag and payload accessor methods.
func generatedEnumMethod(receiver, method string, model *model) bool {
	if len(model.Variants) == 0 {
		return false
	}
	for _, variant := range model.Variants {
		if method == model.Name && receiver == model.Name+variant.Name {
			return true
		}
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
	selector, ok := literal.Type.(*ast.SelectorExpr)
	if !ok {
		return node
	}
	if !p.info.Types[selector.X].IsType() {
		return node
	}
	typ = p.info.TypeOf(selector.X)
	owner, model := p.modelOwner(typ)
	if model == nil || len(model.Variants) == 0 {
		return node
	}
	named := types.Unalias(typ).(*types.Named)
	path := owner.Path
	p.markErasedOwnerImport(file, selector.X, named.Obj().Pkg())
	prefix := p.ownerQualifier(file, named.Obj().Pkg())
	at := selector.Sel.Pos()
	for _, variant := range model.Variants {
		if selector.Sel.Name == variant.Name {
			payload := model.Name + variant.Name
			literal.Type = p.generatedObject(prefix, path, payload, at)
			return call(&ast.SelectorExpr{X: literal, Sel: ast.NewIdent(model.Name)})
		}
	}
	p.fail(literal, "unknown variant %s.%s", model.Name, selector.Sel.Name)
	return node
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
