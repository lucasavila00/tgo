package compiler

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/types"
)

func (p *packageUnit) addDefaults(source *source) {
	for _, model := range source.Models {
		p.addFieldDefaults(source, model.Name, model.Fields)
		for _, variant := range model.Variants {
			p.addFieldDefaults(source, model.Name+variant.Name, variant.Fields)
		}
	}
}

func (p *packageUnit) addFieldDefaults(source *source, name string, fields []field) {
	for _, field := range fields {
		if field.Default == "" {
			continue
		}
		helper := "TgoDefault" + name + field.Name
		text := fmt.Sprintf(
			"package %s\n// %s evaluates the declared default.\nfunc %s() %s { return %s }",
			source.File.Name.Name, helper, helper, field.Type, field.Default,
		)
		name := source.Name + " (default)"
		file, err := parser.ParseFile(p.fs, name, text, parser.SkipObjectResolution)
		if err != nil {
			p.errors = append(p.errors, err)
			continue
		}
		source.File.Decls = append(source.File.Decls, file.Decls...)
	}
}

func (p *packageUnit) fillDefaults() {
	for _, source := range p.Sources {
		for _, declaration := range source.File.Decls {
			if p.generatedDecl(declaration) {
				continue
			}
			ast.Inspect(declaration, func(node ast.Node) bool {
				if literal, ok := node.(*ast.CompositeLit); ok {
					p.fillLiteralDefaults(literal)
				}
				return true
			})
		}
	}
}

func (p *packageUnit) fillLiteralDefaults(literal *ast.CompositeLit) {
	marked := false
	supplied := make(map[string]bool)
	elements := make([]ast.Expr, 0, len(literal.Elts))
	for _, element := range literal.Elts {
		name := fieldName(element)
		if name == "__tgo_defaults" {
			if marked {
				p.fail(literal, "duplicate ..default")
			}
			marked = true
			continue
		}
		supplied[name] = true
		elements = append(elements, element)
	}
	if !marked {
		return
	}
	name, fields := p.literalFields(p.info.TypeOf(literal))
	if name == "" {
		p.fail(literal, "..default needs a tgo struct with declared defaults")
		return
	}
	prefix := typeQualifier(literal.Type)
	for _, field := range fields {
		if supplied[field.Name] {
			continue
		}
		if field.Default == "" {
			p.fail(literal, "missing required field %s", field.Name)
			continue
		}
		helper := qualify(prefix, "TgoDefault"+name+field.Name)
		elements = append(elements, &ast.KeyValueExpr{
			Key:   ast.NewIdent(field.Name),
			Value: call(helper),
		})
	}
	literal.Elts = elements
}

func fieldName(expression ast.Expr) string {
	pair, ok := expression.(*ast.KeyValueExpr)
	if !ok {
		return ""
	}
	name, ok := pair.Key.(*ast.Ident)
	if !ok {
		return ""
	}
	return name.Name
}

func typeQualifier(expression ast.Expr) string {
	selector, ok := expression.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	name, ok := selector.X.(*ast.Ident)
	if !ok {
		return ""
	}
	return name.Name
}

func (p *packageUnit) literalFields(typ types.Type) (string, []field) {
	if model := p.modelForType(typ); model != nil {
		return model.Name, model.Fields
	}
	named, ok := types.Unalias(typ).(*types.Named)
	if !ok {
		return "", nil
	}
	owner := p
	if named.Obj().Pkg() != nil && named.Obj().Pkg().Path() != p.Path {
		owner = p.Imports[named.Obj().Pkg().Path()]
	}
	if owner == nil {
		return "", nil
	}
	for _, model := range owner.Models {
		for _, variant := range model.Variants {
			name := model.Name + variant.Name
			if name == named.Obj().Name() {
				return name, variant.Fields
			}
		}
	}
	return "", nil
}
