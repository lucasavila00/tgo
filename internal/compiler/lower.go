package compiler

import (
	"go/ast"
)

func (p *packageUnit) prepare() {
	p.generated = make(map[ast.Decl]bool)
	for _, source := range p.Sources {
		p.markGenerated(source)
		p.addDefaults(source)
		transform(source.File, func(node ast.Node) ast.Node {
			return p.lowerConstruction(source.File, node)
		})
	}
}

func generatedNames(models []*model) map[string]bool {
	names := make(map[string]bool)
	for _, model := range models {
		if len(model.Variants) > 0 {
			names[model.Name] = true
			for _, variant := range model.Variants {
				names[model.Name+variant.Name] = true
				names["New"+model.Name+variant.Name] = true
			}
		}
		if model.Predicate != "" {
			names[model.Name] = true
			names["New"+model.Name] = true
			names["tgo"+model.Name+"Error"] = true
		}
	}
	return names
}

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

func generatedMethod(function *ast.FuncDecl, models []*model) bool {
	if function.Recv == nil || len(function.Recv.List) != 1 {
		return false
	}
	receiver, ok := function.Recv.List[0].Type.(*ast.Ident)
	if !ok {
		return false
	}
	for _, model := range models {
		method := function.Name.Name
		if model.Predicate != "" {
			if receiver.Name == model.Name && method == "Value" {
				return true
			}
			if receiver.Name == "tgo"+model.Name+"Error" && method == "Error" {
				return true
			}
		}
		if receiver.Name != model.Name || len(model.Variants) == 0 {
			continue
		}
		if method == "TgoTag" {
			return true
		}
		for _, variant := range model.Variants {
			if method == "Tgo"+variant.Name {
				return true
			}
		}
	}
	return false
}

func (p *packageUnit) lowerConstruction(file *ast.File, node ast.Node) ast.Node {
	literal, ok := node.(*ast.CompositeLit)
	if !ok {
		return node
	}
	selector, ok := literal.Type.(*ast.SelectorExpr)
	if !ok {
		return node
	}
	_, model, prefix := p.modelExpr(file, selector.X)
	if model == nil || len(model.Variants) == 0 {
		return node
	}
	for _, variant := range model.Variants {
		if selector.Sel.Name == variant.Name {
			payload := model.Name + variant.Name
			literal.Type = qualify(prefix, payload)
			return call(qualify(prefix, "New"+payload), literal)
		}
	}
	p.fail(literal, "unknown variant %s.%s", model.Name, selector.Sel.Name)
	return node
}
