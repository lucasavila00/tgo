package proof

func CaseJumps(err error) error {
	again := false
Loop:
	for i := 0; i < 2; i += effect("labeled post") {
		effect("labeled body")
		if !again {
			again = true
			goto Loop
		}
		continue Loop
	}
	return nil
}
