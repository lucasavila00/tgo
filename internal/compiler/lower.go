package compiler

import (
	"go/ast"
	"go/types"
)

// prepare marks generated declarations and lowers constructors and defaults.
func (p *packageUnit) prepare() {
	p.generated = make(map[ast.Decl]bool)
	p.generatedValues = make(map[*ast.ValueSpec]bool)
	p.erasedImports = make(map[*ast.ImportSpec]bool)
	p.references = nil
	p.usedIdentifiers = nil
	for _, source := range p.Sources {
		p.markGenerated(source)
		p.addDefaults(source)
	}
}

// lowerConstructions resolves and lowers enum variant literals.
func (p *packageUnit) lowerConstructions() {
	for _, source := range p.Sources {
		transform(source.File, func(node ast.Node) ast.Node {
			return p.lowerConstruction(source.File, node)
		})
	}
}

// generatedNames returns type and constructor names emitted for all models.
func generatedNames(models []*model) map[string]bool {
	names := make(map[string]bool)
	for _, model := range models {
		if len(model.Variants) > 0 {
			names[model.Name+"Tag"] = true
			names[model.Name+"Zero"] = true
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
			if names[node.Name.Name] || generatedMethod(node, source.Models) {
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
			(function.Name.Name == "MarshalJSON" || function.Name.Name == "UnmarshalJSON") {
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
	if method == model.Name && receiver == model.Name+"Zero" {
		return true
	}
	for _, variant := range model.Variants {
		if method == model.Name && receiver == model.Name+variant.Name {
			return true
		}
	}
	if receiver != model.Name {
		return false
	}
	if method == "Tag" || method == "IsZero" || method == "UnknownTag" {
		return true
	}
	for _, variant := range model.Variants {
		if method == variant.Name+"Payload" {
			return true
		}
	}
	return false
}

// lowerConstruction replaces variant literals with generated constructor calls.
func (p *packageUnit) lowerConstruction(file *ast.File, node ast.Node) ast.Node {
	literal, ok := node.(*ast.CompositeLit)
	if !ok {
		return node
	}
	selector, ok := literal.Type.(*ast.SelectorExpr)
	if !ok {
		return node
	}
	if !p.info.Types[selector.X].IsType() {
		return node
	}
	typ := p.info.TypeOf(selector.X)
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
