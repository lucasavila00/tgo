package lowering

import "go/token"

// Propagation defines the error action for a typed marker call.
type Propagation struct {
	Position token.Position
	Wrap     bool
	CallName string
}
