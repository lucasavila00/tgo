package compiler

import (
	"go/ast"
	"go/token"
	"go/types"
)

// addExactEnumConstraintMethods adds the generated enum operations to exact constraints.
func (p *packageUnit) addExactEnumConstraintMethods() bool {
	changed := false
	for _, source := range p.Sources {
		ast.Inspect(source.File, func(node ast.Node) bool {
			constraint, ok := node.(*ast.InterfaceType)
			if !ok {
				return true
			}
			owner, declaration, named := p.exactEnumConstraint(constraint)
			if declaration == nil {
				return true
			}
			prefix := p.ownerQualifier(source.File, named.Obj().Pkg())
			if addEnumConstraintMethods(
				p, constraint, prefix, owner.Path, declaration,
			) {
				source.Lowered = true
				changed = true
			}
			return true
		})
	}
	return changed
}

func (p *packageUnit) exactEnumConstraint(
	constraint *ast.InterfaceType,
) (*packageUnit, *model, *types.Named) {
	var resultOwner *packageUnit
	var result *model
	var resultType *types.Named
	for _, field := range constraint.Methods.List {
		if len(field.Names) != 0 {
			continue
		}
		typ := p.info.TypeOf(field.Type)
		owner, declaration := p.modelOwner(typ)
		if declaration == nil || len(declaration.Variants) == 0 || result != nil {
			return nil, nil, nil
		}
		named, ok := types.Unalias(typ).(*types.Named)
		if !ok {
			return nil, nil, nil
		}
		resultOwner = owner
		result = declaration
		resultType = named
	}
	return resultOwner, result, resultType
}

func addEnumConstraintMethods(
	p *packageUnit,
	constraint *ast.InterfaceType,
	prefix string,
	path string,
	declaration *model,
) bool {
	existing := make(map[string]bool)
	for _, field := range constraint.Methods.List {
		for _, name := range field.Names {
			existing[name.Name] = true
		}
	}
	changed := false
	add := func(name string, result ast.Expr) {
		if existing[name] {
			return
		}
		constraint.Methods.List = append(constraint.Methods.List, &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(name)},
			Type: &ast.FuncType{
				Func: token.NoPos,
				Params: &ast.FieldList{
					Opening: token.NoPos,
					Closing: token.NoPos,
				},
				Results: &ast.FieldList{List: []*ast.Field{{Type: result}}},
			},
		})
		changed = true
	}
	at := constraint.Interface
	add("Tag", p.generatedObject(prefix, path, declaration.Name+"Tag", at))
	for _, variant := range declaration.Variants {
		add(
			variant.Name+"Payload",
			p.generatedObject(prefix, path, declaration.Name+variant.Name, at),
		)
	}
	return changed
}
