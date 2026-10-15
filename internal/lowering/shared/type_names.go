package shared

import (
	"github.com/lucasavila00/tgo/internal/ast"
	"go/token"

	"github.com/lucasavila00/tgo/internal/checker/types"
)

// temporaryTypeErrorPosition selects the nearest propagation operator for an inaccessible-type diagnostic.
func (r *Context) temporaryTypeErrorPosition(original ast.Expr) token.Position {
	position := r.input.Inventory.Checked.Fset.Position(original.Pos())
	var selected token.Pos
	var best int
	for _, expression := range r.input.Inventory.Expressions {
		propagation, ok := expression.(*ast.PropagateExpr)
		if !ok {
			continue
		}
		operationPosition := r.input.Inventory.Checked.Fset.Position(propagation.OpPos)
		distance := int(expression.Pos() - original.Pos())
		if distance < 0 {
			distance = -distance
		}
		closer := selected == 0 || distance < best
		earlierAtSameDistance := distance == best && operationPosition.Offset < position.Offset
		sameFunction := r.functionState.frame != nil && contains(r.functionState.frame.function, expression)
		if operationPosition.IsValid() && sameFunction && (closer || earlierAtSameDistance) {
			position, selected, best = operationPosition, expression.Pos(), distance
		}
	}

	return position
}

// typeNameCheck records source bindings that prevent a temporary type name.
type typeNameCheck struct {
	Context        *Context
	position       token.Pos
	scope          *types.Scope
	problem        string
	unsafePointer  bool
	parameterNames map[*types.TypeParam]string
}

// checkTypeNames checks whether a temporary's type can be named at this source position and records required aliases.
func (r *Context) checkTypeNames(typ types.Type, original ast.Expr, position token.Pos) *typeNameCheck {
	check := &typeNameCheck{
		Context:        r,
		position:       position,
		scope:          r.scopeAt(original, position),
		parameterNames: make(map[*types.TypeParam]string),
	}
	check.visit(types.Default(typ))
	return check
}

// visit visits component types and records inaccessible names, type-parameter aliases, and unsafe-pointer use.
func (c *typeNameCheck) visit(typ types.Type) {
	switch typ := typ.(type) {
	case *types.Basic:
		if typ.Kind() == types.UnsafePointer {
			c.unsafePointer = true
			return
		}
		c.visitName(types.Universe.Lookup(types.Default(typ).String()).(*types.TypeName))
	case *types.TypeParam:
		resolved := c.Context.lookupBinding(c.scope, typ.Obj().Name(), c.position)
		if resolved != typ.Obj() {
			c.parameterNames[typ] = c.Context.parameterAlias(typ).Name
		}
	case *types.Named:
		c.visitName(typ.Obj())
		for i := 0; i < typ.TypeArgs().Len(); i++ {
			c.visit(typ.TypeArgs().At(i))
		}
	case *types.Alias:
		c.visitName(typ.Obj())
		for i := 0; i < typ.TypeArgs().Len(); i++ {
			c.visit(typ.TypeArgs().At(i))
		}
	case *types.Pointer:
		c.visit(typ.Elem())
	case *types.Array:
		c.visit(typ.Elem())
	case *types.Slice:
		c.visit(typ.Elem())
	case *types.Map:
		c.visit(typ.Key())
		c.visit(typ.Elem())
	case *types.Chan:
		c.visit(typ.Elem())
	case *types.Signature:
		c.visit(typ.Params())
		c.visit(typ.Results())
	case *types.Tuple:
		for i := 0; i < typ.Len(); i++ {
			c.visit(typ.At(i).Type())
		}
	case *types.Struct:
		c.visitStruct(typ)
	case *types.Interface:
		c.visitInterface(typ)
	}
}

// visitName checks imported type visibility or local type binding identity at the generated declaration position.
func (c *typeNameCheck) visitName(object *types.TypeName) {
	checked := c.Context.input.Inventory.Checked
	if object.Pkg() != nil && object.Pkg() != checked.Types {
		if !object.Exported() {
			c.problem = "the type is private to " + object.Pkg().Path()
		}
		return
	}
	resolved := c.Context.lookupBinding(c.scope, object.Name(), c.position)
	if resolved != object {
		c.problem = "the source hides the type name " + object.Name()
	}
}

// visitStruct checks field visibility and field types before reconstructing an unnamed struct type.
func (c *typeNameCheck) visitStruct(typ *types.Struct) {
	for i := 0; i < typ.NumFields(); i++ {
		field := typ.Field(i)
		if field.Pkg() != nil && field.Pkg() != c.Context.input.Inventory.Checked.Types && !field.Exported() {
			c.problem = "the type has a private field from " + field.Pkg().Path()
		}
		c.visit(field.Type())
	}
}

// visitInterface checks method visibility and component types before reconstructing an unnamed interface type.
func (c *typeNameCheck) visitInterface(typ *types.Interface) {
	for i := 0; i < typ.NumExplicitMethods(); i++ {
		method := typ.ExplicitMethod(i)
		if method.Pkg() != nil && method.Pkg() != c.Context.input.Inventory.Checked.Types &&
			!method.Exported() {
			c.problem = "the type has a private method from " + method.Pkg().Path()
		}
		c.visit(method.Type())
	}
	for i := 0; i < typ.NumEmbeddeds(); i++ {
		c.visit(typ.EmbeddedType(i))
	}
}
