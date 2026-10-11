//go:build proofreference

package proof

// Reference states the expected effects without calling the fixture functions.
func Reference(name string, reached bool) []string {
	switch name {
	case "Calls":
		return []string{"left", "right", "consume", "pair", "consume"}
	case "Booleans":
		trace := []string{"left"}
		if reached {
			trace = append(trace, "right")
		}
		return append(trace, "observe", "named", "observe")
	case "Declarations":
		return []string{"first", "second", "shadow", "consume", "consume"}
	case "Assignments":
		return []string{"index", "key", "array value", "map value", "add", "consume"}
	case "AssignmentPanic":
		return []string{"bad index", "value before panic"}
	case "Loops":
		trace := []string{"init"}
		for i := 0; i < 2; i++ {
			trace = append(trace, "body", "post")
		}
		return append(trace, "consume")
	case "Switches":
		return []string{"tag", "case one", "matched", "fallthrough", "consume"}
	case "Ranges":
		return []string{"consume", "consume", "range target", "range body", "range target", "range body"}
	case "Selects":
		return []string{"send", "receive target", "received"}
	case "Scheduling":
		return []string{"defer argument", "go argument", "after wait", "consume"}
	case "Recover":
		return []string{"panic argument", "recovered"}
	case "Jumps":
		return []string{"labeled body", "labeled body", "labeled post", "labeled body", "labeled post"}
	case "Types":
		return []string{"receiver index", "method argument", "consume", "foreign argument"}
	case "Nested":
		return []string{"closure", "outside"}
	}
	panic("unknown reference case")
}
