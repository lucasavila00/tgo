package main

import fmt_tgo3 "fmt"

type Failure interface{ Error() string }

func next() (int, Failure) { return 7, nil }

func run(error int) error {
	{
		var tgo1 int
		var tgo2 Failure
		tgo1, tgo2 = next()
		if tgo2 != nil {
			return fmt_tgo3.Errorf("next: %w", tgo2)
		}
		_ = tgo1
	}
	return nil
}

func main() {
	if run(0) != nil {
		panic("unexpected error")
	}
}
