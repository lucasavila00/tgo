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
