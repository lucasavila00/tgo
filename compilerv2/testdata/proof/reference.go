//go:build proofreference

package proof

// Reference states the expected effects without calling the fixture functions.
func Reference(name string, reached bool) []string {
	switch name {
	case "Calls":
		return []string{"left", "right", "consume:[1 1]", "pair", "consume:[1 2]"}
	case "Booleans":
		trace := []string{"left"}
		if reached {
			trace = append(trace, "right")
		}
		if reached {
			trace = append(trace, "observe:true")
		} else {
			trace = append(trace, "observe:false")
		}
		return append(trace, "named", "observe:true")
	case "Declarations":
		return []string{"first", "second", "shadow", "consume:[1 2]", "consume:[1 1]"}
	case "Assignments":
		return []string{"index", "key", "array value", "map value", "add", "consume:[2 1]"}
	case "AssignmentPanic":
		return []string{"bad index", "value before panic"}
	case "Loops":
		trace := []string{"init"}
		for i := 0; i < 2; i++ {
			trace = append(trace, "body", "post")
		}
		return append(trace, "consume:[0 1]")
	case "Switches":
		return []string{"tag", "case one", "matched", "fallthrough", "consume:[1]"}
	case "Ranges":
		return []string{"consume:[0]", "consume:[1]", "range target", "range body", "range target", "range body"}
	case "Selects":
		return []string{"send", "receive target", "received"}
	case "Scheduling":
		return []string{"defer argument", "go argument", "after wait", "consume:[1]"}
	case "Recover":
		return []string{"panic argument", "recovered"}
	case "Jumps":
		return []string{"labeled body", "labeled body", "labeled post", "labeled body", "labeled post"}
	case "Types":
		return []string{"receiver index", "method argument", "consume:[1 1]", "foreign argument"}
	case "Nested":
		return []string{"closure", "outside"}
	}
	panic("unknown reference case")
}

// ReferenceMarkers records handwritten completion points, measured in effects.
func ReferenceMarkers(file string, start, end int, name string, reached bool) ([]int, bool) {
	if file != "calls.go" {
		return nil, false
	}
	if name != "Calls" {
		return nil, true
	}
	positions := map[[2]int]int{
		{50, 90}:   3, // Both arguments and consume have completed.
		{50, 57}:   0,
		{58, 72}:   1,
		{58, 64}:   0,
		{65, 71}:   0,
		{74, 89}:   2,
		{74, 80}:   1,
		{81, 88}:   1,
		{92, 107}:  5,
		{92, 99}:   3,
		{100, 106}: 4,
		{100, 104}: 3,
		{116, 119}: 5,
	}
	position, ok := positions[[2]int{start, end}]
	if !ok {
		panic("missing calls marker reference")
	}
	return []int{position}, true
}
