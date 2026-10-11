package proof

func CaseLoops(err error) error {
	var captured []func() int
	for i := effect("init") - 1; i < 2; i += effect("post") {
		captured = append(captured, func() int { return i })
		effect("body")
		continue
	}
	consume(captured[0](), captured[1]())
	return nil
}
