package frontend

import "go/token"

// ShortReturn records which results a return statement omits.
type ShortReturn struct {
	ZeroSuccess bool
	Position    token.Position
}
