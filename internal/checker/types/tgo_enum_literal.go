package types

import (
	"go/token"

	"github.com/lucasavila00/tgo/internal/ast"
	. "github.com/lucasavila00/tgo/internal/checker/upstream/errors"
)

// enumVariant resolves a qualified variant in a constructor or enum case.
func (check *Checker) enumVariant(expression ast.Expr) (*EnumVariant, bool) {
	selector, ok := ast.Unparen(expression).(*ast.SelectorExpr)
	if !ok {
		return nil, false
	}
	if identifier, ok := selector.X.(*ast.Ident); ok {
		if _, imported := check.lookup(identifier.Name).(*PkgName); imported {
			return nil, false
		}
	}
	var owner operand
	check.exprOrType(&owner, selector.X, false)
	if owner.mode() != typexpr {
		return nil, false
	}
	enum := EnumOf(owner.typ())
	if enum == nil {
		return nil, false
	}
	for _, variant := range enum.variants {
		if variant.name != selector.Sel.Name {
			continue
		}
		if variant.enum.named.Obj().Pkg() != check.pkg && !token.IsExported(variant.name) {
			check.errorf(selector.Sel, UnexportedName, "enum variant %s is not exported", variant.name)
			return nil, true
		}
		check.recordUse(selector.Sel, variant.object)
		check.recordTypeAndValue(expression, typexpr, variant.typ, nil)
		return variant, true
	}
	check.errorf(selector.Sel, UndeclaredName, "%s has no variant %s", enum.named, selector.Sel.Name)
	return nil, true
}

// recordEnumLiteral keeps the source field checks and gives construction the parent enum type.
func (check *Checker) recordEnumLiteral(expression *ast.CompositeLit, value *operand, variant *EnumVariant) {
	if variant == nil || !value.isValid() {
		return
	}
	value.typ_ = variant.enum.named
	if check.EnumConstructors == nil {
		check.EnumConstructors = make(map[*ast.CompositeLit]*EnumVariant)
	}
	check.EnumConstructors[expression] = variant
}
