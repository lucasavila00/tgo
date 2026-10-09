package modernizeiota

type State uint8

const (
	StateReady State = iota
	StateRunning
	StateStopped
)

type Phase int16

const (
	PhaseWarm Phase = iota + 7
	PhaseHot
)

type Pair int

const (
	PairA, PairB Pair = iota * 2, iota*2 + 1
	PairC, PairD
)

type Reset uint8

const (
	ResetA Reset = iota
	ResetB Reset = iota
)
