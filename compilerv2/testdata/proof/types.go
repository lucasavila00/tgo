package proof

import "github.com/lucasavila00/tgo/compilerv2/testdata/proof/foreign"

type counter struct{ value int }

func (c *counter) add(value int) { c.value += value }
func identity[T any](value T) T  { return value }

func CaseTypes(err error) error {
	const huge = 1 << 100
	var small uint8 = huge >> 100
	a := [1]counter{}
	a[effect("receiver index")-1].add(identity(effect("method argument")))
	consume(int(small), a[0].value)
	foreign.Use(foreign.Value(effect("foreign argument")))
	foreign.Number = effect("foreign store")
	consume(foreign.Number)
	return nil
}
