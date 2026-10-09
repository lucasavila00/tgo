package tgolint

import (
	"go/types"
	"strconv"
	"unicode"
)

func validCheckedCarrier(typ types.Type, fields []sourceField) bool {
	named, ok := types.Unalias(typ).(*types.Named)
	if !ok || named.Obj().Pkg() == nil { return false }
	object, ok := named.Obj().Pkg().Scope().Lookup(
		"Tgo" + named.Obj().Name() + "Input",
	).(*types.TypeName)
	if !ok { return false }
	carrier, ok := types.Unalias(object.Type()).(*types.Named)
	if !ok { return false }
	staging, ok := carrier.Underlying().(*types.Struct)
	value, valueOK := named.Underlying().(*types.Struct)
	if !ok || !valueOK || staging.NumFields() != value.NumFields() ||
		len(fields) != value.NumFields() {
		return false
	}
	for index, field := range fields {
		name := "Field" + strconv.Itoa(index)
		if field.name != "" && field.name != "_" {
			runes := []rune(field.name)
			runes[0] = unicode.ToUpper(runes[0])
			name = "Field" + string(runes)
		}
		if staging.Field(index).Name() != name ||
			!types.Identical(staging.Field(index).Type(), value.Field(index).Type()) ||
			staging.Tag(index) != "" {
			return false
		}
	}
	return true
}
