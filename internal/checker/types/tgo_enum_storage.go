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

// enumStorageFamilies collects possible enum owners without following container elements.
func enumStorageFamilies(typ Type) map[*Enum]bool {
	owners := make(map[*Enum]bool)
	collectEnumStorage(typ, owners, make(map[Type]bool))
	return owners
}

// collectEnumStorage follows aliases, pointers, and each finite type-parameter term.
func collectEnumStorage(typ Type, owners map[*Enum]bool, seen map[Type]bool) {
	if typ == nil || seen[typ] {
		owners[nil] = true
		return
	}
	seen[typ] = true
	defer delete(seen, typ)
	typ = Unalias(typ)
	if parameter, ok := typ.(*TypeParam); ok {
		for _, term := range parameter.iface().typeSet().terms {
			if term != nil {
				collectEnumStorage(term.typ, owners, seen)
			}
		}
		return
	}
	if enum := EnumOf(typ); enum != nil {
		owners[enum] = true
		return
	}
	switch underlying := typ.Underlying().(type) {
	case *Pointer:
		collectEnumStorage(underlying.Elem(), owners, seen)
	case *Struct:
		owners[underlying.enumStorage] = true
	default:
		owners[nil] = true
	}
}

// hasEnumStorage reports whether any possible value uses private enum fields.
func hasEnumStorage(typ Type) bool {
	for owner := range enumStorageFamilies(typ) {
		if owner != nil {
			return true
		}
	}
	return false
}

// sameEnumStorage requires every source and destination choice to retain one storage owner.
func sameEnumStorage(source, target Type) bool {
	if Identical(source, target) {
		return true
	}
	for sourceOwner := range enumStorageFamilies(source) {
		for targetOwner := range enumStorageFamilies(target) {
			if sourceOwner != targetOwner {
				return false
			}
		}
	}
	return true
}

// hiddenEnumField prevents TGo field access through enum values or embedded enums.
func (check *Checker) hiddenEnumField(selector *ast.SelectorExpr, receiver Type, indexes []int) bool {
	if !check.nativeSource(selector.Pos()) {
		return false
	}
	for _, index := range indexes {
		if hasEnumStorage(receiver) {
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

// hiddenEnumConversion prevents structural assignments and conversions from exposing enum storage.
func (check *Checker) hiddenEnumConversion(value *operand, target Type) bool {
	if !check.nativeSource(value.Pos()) || value.isNil() || Identical(value.typ(), target) ||
		isNonTypeParamInterface(target) {
		return false
	}
	sourceBasic, _ := value.typ().Underlying().(*Basic)
	targetBasic, _ := target.Underlying().(*Basic)
	if sourceBasic != nil && sourceBasic.Kind() == UnsafePointer ||
		targetBasic != nil && targetBasic.Kind() == UnsafePointer {
		return false
	}
	if sameEnumStorage(value.typ(), target) {
		return false
	}
	check.errorf(value, InvalidConversion, "cannot use enum storage as a separate struct type")
	return true
}
