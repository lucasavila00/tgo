package types

import (
	"reflect"

	"github.com/lucasavila00/tgo/internal/ast"
)

// ConstInitializer records the syntax checked for one constant declaration.
// Inherited declarations have separate nodes because iota can change their types and values.
type ConstInitializer struct {
	Type  ast.Expr
	Value ast.Expr
}

// constInitializer separates inherited TGo syntax before checking types, values, and nested functions.
func (check *Checker) constInitializer(object *Const, typ, value ast.Expr, inherited bool) (ast.Expr, ast.Expr) {
	if !check.nativeSource(object.Pos()) {
		return typ, value
	}
	if inherited {
		typ, value = copyConstExpression(typ), copyConstExpression(value)
	}
	if check.ConstInitializers == nil {
		check.ConstInitializers = make(map[*Const]ConstInitializer)
	}
	check.ConstInitializers[object] = ConstInitializer{Type: typ, Value: value}
	return typ, value
}

// copyConstExpression keeps positions but gives the checker a separate syntax tree.
func copyConstExpression(expression ast.Expr) ast.Expr {
	if expression == nil {
		return nil
	}
	return copyConstSyntax(reflect.ValueOf(expression)).Interface().(ast.Expr)
}

// copyConstSyntax copies syntax fields and drops the parser's deprecated binding objects.
func copyConstSyntax(value reflect.Value) reflect.Value {
	switch value.Kind() {
	case reflect.Interface:
		if value.IsNil() {
			return value
		}
		result := reflect.New(value.Type()).Elem()
		result.Set(copyConstSyntax(value.Elem()))
		return result
	case reflect.Pointer:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		result := reflect.New(value.Type().Elem())
		result.Elem().Set(copyConstSyntax(value.Elem()))
		return result
	case reflect.Slice:
		if value.IsNil() {
			return value
		}
		result := reflect.MakeSlice(value.Type(), value.Len(), value.Len())
		for index := range value.Len() {
			result.Index(index).Set(copyConstSyntax(value.Index(index)))
		}
		return result
	case reflect.Struct:
		result := reflect.New(value.Type()).Elem()
		for index := range value.NumField() {
			name := value.Type().Field(index).Name
			if name == "Obj" || name == "Scope" {
				continue
			}
			result.Field(index).Set(copyConstSyntax(value.Field(index)))
		}
		return result
	default:
		return value
	}
}
