package types

import "go/ast"

// builtinArguments supplies declared parameter types to added expressions.
func (check *Checker) builtinArguments(call *ast.CallExpr, id builtinId) []*operand {
	previous := check.sliceSourceContext
	check.sliceSourceContext = nil
	defer func() { check.sliceSourceContext = previous }()
	if id == _Min || id == _Max || id == _Complex {
		return check.numericBuiltinArguments(call)
	}
	if id == _Panic && len(call.Args) == 1 {
		check.expressionType(call.Args[0], &emptyInterface)
		return check.exprList(call.Args)
	}
	if (id != _Append && id != _Delete && id != _Copy) || len(call.Args) < 2 {
		return check.exprList(call.Args)
	}
	args := make([]*operand, len(call.Args))
	for i, expression := range call.Args {
		args[i] = new(operand)
		check.expr(nil, args[i], expression)
		if i == 0 && args[i].isValid() {
			typ := check.collectionArgumentType(call, id, args[i])
			for _, value := range call.Args[1:] {
				check.expressionType(value, typ)
			}
		}
	}
	return args
}

func (check *Checker) collectionArgumentType(call *ast.CallExpr, id builtinId, argument *operand) Type {
	var typ Type
	switch {
	case id == _Delete && !hasDots(call):
		typ = deleteKeyType(argument.typ())
	case id == _Append && !hasDots(call):
		typ, _ = sliceElem(argument)
	case id == _Copy || id == _Append && hasDots(call):
		if elem, _ := sliceElem(argument); elem != nil {
			typ = NewSlice(elem)
			if Identical(elem, universeByte) {
				check.sliceSourceContext = typ
			}
		}
	}
	return typ
}

// numericBuiltinArguments lets another argument supply a terminating if's type.
func (check *Checker) numericBuiltinArguments(call *ast.CallExpr) []*operand {
	conditional := false
	for _, expression := range call.Args {
		if marker, ok := ast.Unparen(expression).(*ast.CallExpr); ok {
			if operation, found := check.tgoOperation(marker); found && operation.Conditional {
				conditional = true
			}
		}
	}
	if !conditional {
		return check.exprList(call.Args)
	}
	args := make([]*operand, len(call.Args))
	var context Type
	for i, expression := range call.Args {
		if marker, ok := ast.Unparen(expression).(*ast.CallExpr); ok {
			if operation, found := check.tgoOperation(marker); found && operation.Conditional {
				continue
			}
		}
		args[i] = new(operand)
		check.expr(nil, args[i], expression)
		if args[i].isValid() && !isUntyped(args[i].typ()) && context == nil {
			context = args[i].typ()
		}
	}
	for i, expression := range call.Args {
		if args[i] == nil {
			check.expressionType(expression, context)
			args[i] = new(operand)
			check.expr(nil, args[i], expression)
		}
	}
	return args
}

func deleteKeyType(typ Type) Type {
	var key Type
	if !underIs(typ, func(underlying Type) bool {
		mapping, ok := underlying.(*Map)
		if !ok || (key != nil && !Identical(key, mapping.key)) {
			return false
		}
		key = mapping.key
		return true
	}) {
		return nil
	}
	return key
}
