package nilsafety

import (
	"go/types"

	"tgo/internal/tgolint/typefacts"
)

const (
	goTypeTagNil           = typefacts.TypeTagNil
	goTypeTagBasic         = typefacts.TypeTagBasic
	goTypeTagArray         = typefacts.TypeTagArray
	goTypeTagSlice         = typefacts.TypeTagSlice
	goTypeTagStruct        = typefacts.TypeTagStruct
	goTypeTagPointer       = typefacts.TypeTagPointer
	goTypeTagTuple         = typefacts.TypeTagTuple
	goTypeTagSignature     = typefacts.TypeTagSignature
	goTypeTagMap           = typefacts.TypeTagMap
	goTypeTagChannel       = typefacts.TypeTagChannel
	goTypeTagInterface     = typefacts.TypeTagInterface
	goTypeTagNamed         = typefacts.TypeTagNamed
	goTypeTagTypeParameter = typefacts.TypeTagTypeParameter
	goTypeTagUnion         = typefacts.TypeTagUnion
	goTypeTagOther         = typefacts.TypeTagOther
)

func goTypeOf(typ types.Type) typefacts.Type { return typefacts.Of(typ) }

func coreType(typ types.Type) types.Type { return typefacts.Core(typ) }
