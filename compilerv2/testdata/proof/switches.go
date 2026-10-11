package proof

func CaseSwitches(err error) error {
	switch effect("tag") {
	case effect("case one"):
		effect("matched")
		fallthrough
	case effect("case two"):
		effect("fallthrough")
	}
	var value any = 1
	switch bound := value.(type) {
	case int:
		consume(bound)
	}
	return nil
}
