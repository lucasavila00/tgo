package compiler

import (
	"go/ast"
	"go/types"
)

// applyEnumLayouts selects storage and updates generated declarations.
func (p *packageUnit) applyEnumLayouts() {
	seen := make(map[types.Type]bool)
	for _, source := range p.Sources {
		for _, declaration := range source.Models {
			if declaration.Enum {
				typ := p.typed.Scope().Lookup(declaration.Name).Type()
				p.selectEnumLayouts(typ, seen)
			}
		}
	}
	for _, source := range p.Sources {
		for _, declaration := range source.Models {
			if declaration.Enum {
				boxEnumDeclarations(source.File, declaration)
			}
		}
	}
}

// boxEnumDeclarations updates only storage, constructors, and accessors.
func boxEnumDeclarations(file *ast.File, declaration *model) {
	boxed := make(map[string]bool)
	for _, variant := range declaration.Variants {
		if variant.Boxed {
			boxed[variant.Name] = true
		}
	}
	if len(boxed) == 0 {
		return
	}
	for _, node := range file.Decls {
		switch node := node.(type) {
		case *ast.GenDecl:
			if declarationKey(node) == "type "+declaration.Name {
				structure := node.Specs[0].(*ast.TypeSpec).Type.(*ast.StructType)
				boxEnumFields(structure, declaration)
			}
		case *ast.FuncDecl:
			boxEnumFunction(node, declaration.Name, boxed)
		}
	}
}

// boxEnumFields retains field positions so removed storage leaves no blank lines.
func boxEnumFields(structure *ast.StructType, declaration *model) {
	original := structure.Fields.List
	fields := []*ast.Field{original[0]}
	index := 1
	for _, variant := range declaration.Variants {
		if len(variant.Fields) == 0 {
			continue
		}
		if !variant.Boxed {
			slot := original[len(fields)]
			fields = append(fields, &ast.Field{
				Names: []*ast.Ident{{
					Name: original[index].Names[0].Name, NamePos: slot.Names[0].NamePos,
				}},
				Type: &ast.Ident{
					Name: original[index].Type.(*ast.Ident).Name, NamePos: slot.Type.Pos(),
				},
			})
		}
		index++
	}
	position := original[len(fields)].Type.Pos()
	box := &ast.Field{
		Names: []*ast.Ident{{
			Name: "tgoPayload", NamePos: original[len(fields)].Names[0].NamePos,
		}},
		Type: &ast.InterfaceType{
			Interface: position,
			Methods:   &ast.FieldList{Opening: position, Closing: position},
		},
	}
	fields = append(fields, box)
	structure.Fields.List = fields
}

// boxEnumFunction changes the stored value or the returned payload.
func boxEnumFunction(function *ast.FuncDecl, enum string, boxed map[string]bool) {
	for variant := range boxed {
		if function.Recv == nil && function.Name.Name == "New"+enum+variant {
			statement := function.Body.List[0].(*ast.ReturnStmt)
			literal := statement.Results[0].(*ast.CompositeLit)
			literal.Elts[1].(*ast.KeyValueExpr).Key.(*ast.Ident).Name = "tgoPayload"
			return
		}
		receiver, ok := receiverName(function)
		if !ok || receiver != enum || function.Name.Name != "Tgo"+variant {
			continue
		}
		statement := function.Body.List[0].(*ast.ReturnStmt)
		selector := statement.Results[0].(*ast.SelectorExpr)
		selector.Sel.Name = "tgoPayload"
		statement.Results[0] = &ast.TypeAssertExpr{
			X:      selector,
			Lparen: selector.Pos(),
			Type:   &ast.Ident{Name: enum + variant, NamePos: selector.Pos()},
			Rparen: selector.Pos(),
		}
		return
	}
}
