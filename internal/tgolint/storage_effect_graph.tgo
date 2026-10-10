package tgolint

// StorageEffectGraph is the ordered storage behavior of one function.
// Known distinguishes an empty summary from an absent summary.
type StorageEffectGraph struct {
	Known     bool
	Entry     int
	Functions []StorageEffectFunction
}

// StorageEffectFunction is one finite function or closure graph.
type StorageEffectFunction struct {
	ID            int
	Captures      []StorageEffectRegion
	ZeroEffects   []GenericEffect
	AccessEffects []GenericEffect
	Blocks        []StorageEffectBlock
}

// StorageEffectBlock keeps operations in source evaluation order.
type StorageEffectBlock struct {
	ID         int
	Operations []StorageEffectOperation
	Successors []StorageEffectEdge
}

// StorageEffectEdge is one possible control-flow successor.
type StorageEffectEdge struct {
	Block     int
	Condition int
	Expected  bool
}

// StorageEffectOperation is one ordered wire operation.
type StorageEffectOperation struct {
	Kind              int
	Position          int
	Target            StorageEffectRegion
	Source            StorageEffectRegion
	Regions           []StorageEffectRegion
	Inputs            []int
	Results           []int
	TypeArguments     []StorageEffectType
	ReceiverArguments []StorageEffectType
	CalledParameters  []int
	Function          int
	Field             string
	Offset            int64
	Length            int64
	Capacity          int64
	KnownOffset       bool
	KnownLength       bool
	KnownCapacity     bool
	Boolean           bool
	Variadic          bool
	ProjectArguments  bool
	Operator          int
	ZeroEffects       []GenericEffect
	AccessEffects     []GenericEffect
}

// StorageEffectType is a serializable type expression in a call graph.
type StorageEffectType struct {
	Kind         int
	Parameter    int
	Position     int
	Name         string
	Package      string
	Basic        int
	Length       int64
	Direction    int
	Variadic     bool
	Embedded     bool
	PackageLevel bool
	Tag          string
	Element      []StorageEffectType
	Key          []StorageEffectType
	Fields       []StorageEffectType
	Parameters   []StorageEffectType
	Results      []StorageEffectType
	Arguments    []StorageEffectType
	Underlying   []StorageEffectType
	FieldNames   []string
	FieldPkgs    []string
	FieldTags    []string
	FieldEmbed   []bool
}

const (
	storageTypeParameter = iota + 1
	storageTypeReceiver
	storageTypeBasic
	storageTypeSlice
	storageTypeArray
	storageTypeMap
	storageTypePointer
	storageTypeChannel
	storageTypeStruct
	storageTypeSignature
	storageTypeNamed
	storageTypeInterface
)

const (
	storageEffectRead = iota + 1
	storageEffectWrite
	storageEffectCall
	storageEffectAllocate
	storageEffectSlice
	storageEffectAppend
	storageEffectCopy
	storageEffectReturn
	storageEffectFunction
	storageEffectObserve
	storageEffectIndexRead
	storageEffectIndexWrite
	storageEffectFieldRead
	storageEffectFieldWrite
	storageEffectBoolean
	storageEffectInteger
	storageEffectUnary
	storageEffectBinary
)

// StorageEffectRegion names parameter, capture, temporary, or allocation storage.
type StorageEffectRegion struct {
	Root int
	ID   int
	Path []StorageEffectPath
}

const (
	storageRootUnknown = iota
	storageRootParameter
	storageRootCapture
	storageRootTemporary
	storageRootAllocation
	storageRootResult
)

// StorageEffectPath selects one field or index slot.
type StorageEffectPath struct {
	Kind  int
	Field string
	Index int64
}

const (
	storagePathField = iota + 1
	storagePathIndex
	storagePathAnyIndex
)
