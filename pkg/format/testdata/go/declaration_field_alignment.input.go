package alignment

const (
	z64USize zip64SubID = iota + 1
	z64CSize
	z64Offset
)

const (
	PR_GET_KEEPCAPS uintptr = 7
	PR_SET_KEEPCAPS = 8
)

var (
	shortValue int = 1 // short
	muchLongerValue = 2 // long
)

type (
	_C_short int16
	_C_int int32
	_C_long int64
	_C_long_long int64
)

type fields struct {
	CompressInstructions int `help:"compress"` // compress
	MayMoreStack string `help:"stack"` // stack
	Embedded `json:"embedded"` // embedded
	Short int // short
	LongerName string // long
	OtherEmbedded // other

	Separate int `json:"separate"`
	// Keep this field in a new section.
	X string `json:"x"`
}

type zip64SubID int
type Embedded int
type OtherEmbedded int
