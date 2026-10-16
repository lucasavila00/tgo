package types

import (
	"github.com/lucasavila00/tgo/internal/ast"
	. "github.com/lucasavila00/tgo/internal/checker/upstream/errors"
)

// enumSizes resolves inline enum dependencies before Go observes their memory size.
func (check *Checker) enumSizes(typ Type, position ast.Node) bool {
	return check.enumSizeType(typ, position, make(map[Type]bool))
}

// enumSizeType follows only fields and arrays, because pointers have a fixed size.
func (check *Checker) enumSizeType(typ Type, position ast.Node, seen map[Type]bool) bool {
	typ = Unalias(typ)
	if seen[typ] {
		return true
	}
	seen[typ] = true
	if named, ok := typ.(*Named); ok {
		if !isValid(named.rhs()) {
			return false
		}
		if named.rhs() == nil {
			check.error(position, InvalidConstInit, "enum size depends on an incomplete type declaration")
			return false
		}
		if enum := named.enum; enum != nil && !enum.layoutReady {
			return check.finishEnumLayout(enum, position, seen)
		}
	}
	switch underlying := typ.Underlying().(type) {
	case *Array:
		return check.enumSizeType(underlying.Elem(), position, seen)
	case *Struct:
		for _, field := range underlying.fields {
			if !check.enumSizeType(field.Type(), position, seen) {
				return false
			}
		}
	}
	return true
}

// finishEnumLayout checks source cycles, then lays out nested enum values before their parent.
func (check *Checker) finishEnumLayout(enum *Enum, position ast.Node, seen map[Type]bool) bool {
	if !check.validType0(enum.object.Pos(), enum.named, nil, nil) {
		return false
	}
	for _, variant := range enum.variants {
		if !check.enumSizeType(variant.typ, position, seen) {
			return false
		}
	}
	storage := enum.named.Underlying().(*Struct)
	*storage = *enumLayout(enum)
	enum.layoutReady = true
	return true
}
