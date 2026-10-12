package foreign

type hidden struct{ Number int }

var Calls int

func Pair() (hidden, hidden) {
	Calls++
	return hidden{1}, hidden{2}
}
