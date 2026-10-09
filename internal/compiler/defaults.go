package compiler

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/types"
)

// addDefaults emits one helper for each declared field default.
func (p *packageUnit) addDefaults(source *source) {
	for _, model := range source.Models {
		p.addFieldDefaults(source, model.Name, model.Fields)
		for _, variant := range model.Variants {
			p.addFieldDefaults(source, model.Name+variant.Name, variant.Fields)
		}
	}
}

// addFieldDefaults adds default helpers for one struct or variant payload.
func (p *packageUnit) addFieldDefaults(source *source, name string, fields []field) {
	for _, field := range fields {
		if field.Default == "" {
			continue
		}
		helper := "TgoDefault" + name + field.Name
		text := fmt.Sprintf(
			"package %s\n// %s evaluates the declared default.\n"+
				"func %s() %s { return (\n%s%s) }",
			source.File.Name.Name,
			helper,
			helper,
			field.Type,
			lineDirective(source.Name, field.DefaultLine, field.DefaultColumn),
			field.Default,
		)
		name := source.Name + " (default)"
		file, err := parser.ParseFile(p.fs, name, text, parser.SkipObjectResolution)
		if err != nil {
			p.errors = append(p.errors, err)
			continue
		}
		removeDefaultParentheses(file)
		source.File.Decls = append(source.File.Decls, file.Decls...)
	}
}

// removeDefaultParentheses restores the normal generated helper shape.
func removeDefaultParentheses(file *ast.File) {
	function := file.Decls[0].(*ast.FuncDecl)
	statement := function.Body.List[0].(*ast.ReturnStmt)
	parenthesized := statement.Results[0].(*ast.ParenExpr)
	statement.Results[0] = parenthesized.X
}

// fillDefaults replaces each marker with calls to declared default helpers.
func (p *packageUnit) fillDefaults() {
	for _, source := range p.Sources {
		for _, declaration := range source.File.Decls {
			if p.generatedDecl(declaration) {
				continue
			}
			ast.Inspect(declaration, func(node ast.Node) bool {
				if literal, ok := node.(*ast.CompositeLit); ok {
					p.fillLiteralDefaults(source, literal, source.DefaultMarker)
				}
				return true
			})
		}
	}
}

// fillEnumDefaults expands selected defaults before namespace literals lower.
func (p *packageUnit) fillEnumDefaults() {
	for _, source := range p.Sources {
		for _, declaration := range source.File.Decls {
			if p.generatedDecl(declaration) {
				continue
			}
			ast.Inspect(declaration, func(node ast.Node) bool {
				literal, ok := node.(*ast.CompositeLit)
				if !ok {
					return true
				}
				selector, owner, model, variant := p.enumLiteral(literal)
				if model == nil {
					return true
				}
				named, _ := types.Unalias(p.info.TypeOf(selector.X)).(*types.Named)
				if named == nil {
					return true
				}
				prefix := p.ownerQualifier(source.File, named.Obj().Pkg())
				p.fillLiteralDefaultsFor(
					source,
					literal,
					source.DefaultMarker,
					owner,
					model.Name+variant.Name,
					variant.Fields,
					prefix,
				)
				return true
			})
		}
	}
}

// fillLiteralDefaults adds omitted default fields to one composite literal.
func (p *packageUnit) fillLiteralDefaults(
	source *source,
	literal *ast.CompositeLit,
	marker string,
) {
	owner, name, fields := p.literalFields(p.info.TypeOf(literal))
	if name == "" {
		if hasDefaultMarker(literal, marker) {
			p.fail(literal, "..default needs a tgo struct with declared defaults")
		}
		return
	}
	named := types.Unalias(p.info.TypeOf(literal)).(*types.Named)
	prefix := p.ownerQualifier(source.File, named.Obj().Pkg())
	p.fillLiteralDefaultsFor(source, literal, marker, owner, name, fields, prefix)
}

func (p *packageUnit) fillLiteralDefaultsFor(
	source *source,
	literal *ast.CompositeLit,
	marker string,
	owner *packageUnit,
	name string,
	fields []field,
	prefix string,
) {
	marked := false
	supplied := make(map[string]bool)
	elements := make([]ast.Expr, 0, len(literal.Elts))
	for _, element := range literal.Elts {
		name := fieldName(element)
		if name == marker {
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
	source.Lowered = true
	for _, field := range fields {
		if supplied[field.Name] {
			continue
		}
		if field.Default == "" {
			continue
		}
		helper := p.generatedObject(
			prefix,
			owner.Path,
			"TgoDefault"+name+field.Name,
			literal.Lbrace,
		)
		elements = append(elements, &ast.KeyValueExpr{
			Key:   ast.NewIdent(field.Name),
			Value: call(helper),
		})
	}
	literal.Elts = elements
}

func hasDefaultMarker(literal *ast.CompositeLit, marker string) bool {
	for _, element := range literal.Elts {
		if fieldName(element) == marker {
			return true
		}
	}
	return false
}

// fieldName returns the key name from a keyed literal element.
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

// literalFields returns tgo fields for a struct or variant payload type.
func (p *packageUnit) literalFields(typ types.Type) (*packageUnit, string, []field) {
	if owner, model := p.modelOwner(typ); model != nil {
		return owner, model.Name, model.Fields
	}
	named, ok := types.Unalias(typ).(*types.Named)
	if !ok {
		return nil, "", nil
	}
	owner := p
	if named.Obj().Pkg() != nil && named.Obj().Pkg().Path() != p.Path {
		owner = p.Imports[named.Obj().Pkg().Path()]
	}
	if owner == nil {
		return nil, "", nil
	}
	for _, model := range owner.Models {
		for _, variant := range model.Variants {
			name := model.Name + variant.Name
			if name == named.Obj().Name() {
				return owner, name, variant.Fields
			}
		}
	}
	return nil, "", nil
}
