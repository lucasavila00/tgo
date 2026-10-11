package main

import fmt_tgo6 "fmt"

import "fmt"

func target() (int, error) {
	return 0, fmt.Errorf("failure")
}

func run() (tgo3 error) {
	var tgo4 error = tgo3
	{
		var tgo1 int
		var tgo2 error
		tgo1, tgo2 = target()
		if tgo2 != nil {
			tgo4 = tgo2
			tgo4 = fmt_tgo6.Errorf("%s: %w", "target", tgo4)
			return tgo4
		}
		_ = tgo1
	}
	return nil
}

func main() {
	fmt.Println(run())
	fmt.Println(fmt_tgo5)
}
