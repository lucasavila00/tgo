package lowering

// Propagation defines the error action for a typed marker call.
type Propagation struct {
	Wrap     bool
	CallName string
}
