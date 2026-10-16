package types

import "strconv"

// enumTagKind selects the smallest unsigned integer that holds every variant tag.
func enumTagKind(variants int) BasicKind {
	last := uint64(variants - 1)
	switch {
	case last <= 1<<8-1:
		return Uint8
	case last <= 1<<16-1:
		return Uint16
	case last <= 1<<32-1:
		return Uint32
	default:
		return Uint64
	}
}

// enumStorage builds typed private fields for the selected inline and boxed payloads.
func enumStorage(enum *Enum) *Struct {
	pkg := enum.named.Obj().Pkg()
	fields := []*Var{NewVar(0, pkg, "tgoTag", enum.tag)}
	boxed := false
	for _, variant := range enum.variants {
		payload := variant.typ.Underlying().(*Struct)
		if payload.NumFields() == 0 {
			continue
		}
		if variant.boxed {
			boxed = true
			continue
		}
		name := "tgoVariant" + strconv.Itoa(variant.index)
		fields = append(fields, NewVar(0, pkg, name, variant.typ))
	}
	if boxed {
		fields = append(fields, NewVar(0, pkg, "tgoPayload", NewInterfaceType(nil, nil).Complete()))
	}
	storage := NewStruct(fields, nil)
	if boxed && enumNeedsComparisonMarker(enum, storage) {
		marker := NewArray(NewSignatureType(nil, nil, nil, nil, nil, false), 0)
		fields = append(fields, NewVar(0, pkg, "_", marker))
		storage = NewStruct(fields, nil)
	}
	storage.enumStorage = enum
	return storage
}

// largestEnumPayload finds the largest remaining inline field, keeping declaration order on ties.
func largestEnumPayload(enum *Enum, sizes Sizes, first int) *EnumVariant {
	var largest *EnumVariant
	for _, variant := range enum.variants[first:] {
		if variant.boxed || variant.typ.Underlying().(*Struct).NumFields() == 0 {
			continue
		}
		if largest == nil || sizes.Sizeof(variant.typ) > sizes.Sizeof(largest.typ) {
			largest = variant
		}
	}
	return largest
}

// enumNeedsComparisonMarker keeps boxed noncomparable fields out of Go comparisons.
func enumNeedsComparisonMarker(enum *Enum, storage *Struct) bool {
	if !Comparable(storage) {
		return false
	}
	for _, variant := range enum.variants {
		if !Comparable(variant.typ) {
			return true
		}
	}
	return false
}
