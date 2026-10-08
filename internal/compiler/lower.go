package compiler

import (
	"go/ast"
	"go/types"
	"strconv"

	"golang.org/x/tools/go/ast/astutil"
)

// prepare marks generated declarations and lowers constructors and defaults.
func (p *packageUnit) prepare() {
	p.generated = make(map[ast.Decl]bool)
	p.erasedImports = make(map[*ast.ImportSpec]bool)
	p.references = nil
	p.usedIdentifiers = nil
	for _, source := range p.Sources {
		if len(source.Models) != 0 {
			for _, specification := range source.File.Imports {
				path, err := strconv.Unquote(specification.Path.Value)
				if err == nil && path == p.Module+"/internal/tgoruntime" {
					p.fail(specification, "generated validation runtime import is reserved")
				}
			}
			astutil.AddNamedImport(
				p.fs,
				source.File,
				source.RuntimeAlias,
				p.Module+"/internal/tgoruntime",
			)
		}
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
		names["Validate"+model.Name] = true
		names["tgo"+model.Name+"ValidationError"] = true
		if len(model.Variants) > 0 {
			for _, variant := range model.Variants {
				names[model.Name+variant.Name] = true
				names["New"+model.Name+variant.Name] = true
				names["tgoReconstruct"+model.Name+variant.Name] = true
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
		if receiver == model.Name && function.Name.Name == "TgoReconstruct" {
			return true
		}
		if generatedCheckedMethod(receiver, function.Name.Name, model) {
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
	if receiver != model.Name || len(model.Variants) == 0 {
		return false
	}
	if method == "TgoTag" {
		return true
	}
	for _, variant := range model.Variants {
		if method == "Tgo"+variant.Name {
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
			constructor := p.generatedObject(prefix, path, "New"+payload, at)
			return call(constructor, literal)
		}
	}
	p.fail(literal, "unknown variant %s.%s", model.Name, selector.Sel.Name)
	return node
}
