package types

import (
	"go/token"

	"github.com/lucasavila00/tgo/internal/ast"
	. "github.com/lucasavila00/tgo/internal/checker/upstream/errors"
)

// nativeSource reports whether a position belongs to source parsed as TGo.
func (check *Checker) nativeSource(position token.Pos) bool {
	for _, file := range check.files {
		if file.FileStart <= position && position <= file.FileEnd {
			return file.TGo
		}
	}
	return false
}

// enumStorageOwner follows pointer types to the declaration that owns private fields.
func enumStorageOwner(typ Type) *Enum {
	for typ != nil {
		if enum := EnumOf(typ); enum != nil {
			return enum
		}
		switch underlying := typ.Underlying().(type) {
		case *Pointer:
			typ = underlying.Elem()
		case *Struct:
			return underlying.enumStorage
		default:
			return nil
		}
	}
	return nil
}

// hiddenEnumField prevents TGo field access through enum values or embedded enums.
func (check *Checker) hiddenEnumField(selector *ast.SelectorExpr, receiver Type, indexes []int) bool {
	if !check.nativeSource(selector.Pos()) {
		return false
	}
	for _, index := range indexes {
		if enumStorageOwner(receiver) != nil {
			check.errorf(selector.Sel, MissingFieldOrMethod, "enum storage is private; use an enum type switch")
			return true
		}
		receiver, _ = deref(receiver)
		structure, ok := receiver.Underlying().(*Struct)
		if !ok {
			return false
		}
		receiver = structure.Field(index).Type()
	}
	return false
}

// hiddenEnumConversion prevents ordinary struct conversions from exposing enum storage.
func (check *Checker) hiddenEnumConversion(value *operand, target Type) bool {
	if !check.nativeSource(value.Pos()) || value.isNil() || IsInterface(target) {
		return false
	}
	sourceOwner := enumStorageOwner(value.typ())
	targetOwner := enumStorageOwner(target)
	if sourceOwner == targetOwner {
		return false
	}
	check.errorf(value, InvalidConversion, "cannot convert between enum storage and a separate struct type")
	return true
}
