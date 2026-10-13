package shared

import (
	"go/ast"
	"go/token"

	"github.com/lucasavila00/tgo/internal/checker/types"
)

func (r *Context) temporaryTypeErrorPosition(original ast.Expr) token.Position {
	position := r.input.Inventory.Checked.Fset.Position(original.Pos())
	var selected token.Pos
	var best int
	for expression, operation := range r.input.Propagations {
		distance := int(expression.Pos() - original.Pos())
		if distance < 0 {
			distance = -distance
		}
		closer := selected == 0 || distance < best
		earlierAtSameDistance := distance == best && operation.Position.Offset < position.Offset
		if operation.Position.IsValid() && r.functionState.frame != nil &&
			contains(r.functionState.frame.function, expression) && (closer || earlierAtSameDistance) {
			position = operation.Position
			selected = expression.Pos()
			best = distance
		}
	}
	return position
}

// typeNameCheck records source bindings that prevent a temporary type name.
type typeNameCheck struct {
	Context        *Context
	position       token.Pos
	problem        string
	unsafePointer  bool
	parameterNames map[*types.TypeParam]string
}

func (r *Context) checkTypeNames(typ types.Type, position token.Pos) *typeNameCheck {
	check := &typeNameCheck{
		Context:        r,
		position:       position,
		parameterNames: make(map[*types.TypeParam]string),
	}
	check.visit(types.Default(typ))
	return check
}

func (c *typeNameCheck) visit(typ types.Type) {
	switch typ := typ.(type) {
	case *types.Basic:
		if typ.Kind() == types.UnsafePointer {
			c.unsafePointer = true
			return
		}
		c.visitName(types.Universe.Lookup(types.Default(typ).String()).(*types.TypeName))
	case *types.TypeParam:
		scope := c.Context.input.Inventory.Checked.Types.Scope().Innermost(c.position)
		_, resolved := scope.LookupParent(typ.Obj().Name(), c.position)
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

func (c *typeNameCheck) visitName(object *types.TypeName) {
	checked := c.Context.input.Inventory.Checked
	if object.Pkg() != nil && object.Pkg() != checked.Types {
		if !object.Exported() {
			c.problem = "the type is private to " + object.Pkg().Path()
		}
		return
	}
	scope := checked.Types.Scope().Innermost(c.position)
	_, resolved := scope.LookupParent(object.Name(), c.position)
	if resolved != object {
		c.problem = "the source hides the type name " + object.Name()
	}
}

func (c *typeNameCheck) visitStruct(typ *types.Struct) {
	for i := 0; i < typ.NumFields(); i++ {
		field := typ.Field(i)
		if field.Pkg() != nil && field.Pkg() != c.Context.input.Inventory.Checked.Types && !field.Exported() {
			c.problem = "the type has a private field from " + field.Pkg().Path()
		}
		c.visit(field.Type())
	}
}

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
