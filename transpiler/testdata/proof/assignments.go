package proof

func CaseAssignments(err error) error {
	a := [3]int{}
	p := &a
	m := map[int]int{}
	a[effect("index")], m[effect("key")] = effect("array value"), effect("map value")
	p[1] += effect("add")
	consume(a[1], m[1])
	return nil
}

func CaseAssignmentPanic(err error) error {
	a := [1]int{}
	a[effect("bad index")] = effect("value before panic")
	return nil
}
