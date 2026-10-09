package modernizegood

type Permission uint8

const (
	PermissionRead Permission = 1 << iota
	PermissionWrite
)

type Code uint8

const (
	CodeOne Code = iota * 2
	CodeThree
)

type Duplicate uint8

const (
	DuplicateFirst Duplicate = iota / 2
	DuplicateSecond
)

type Manual uint8

const (
	ManualZero Manual = 0
	ManualOne Manual = 1
)

type Untyped = uint8

const (
	UntypedZero Untyped = iota
	UntypedOne
)

type Single uint8

const SingleOnly Single = iota

type Mixed uint8

const (
	MixedZero Mixed = iota
	MixedOne Mixed = 1
)
